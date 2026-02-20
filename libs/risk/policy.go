// Package risk provides versioned risk policy loading and enforcement for the
// Jax Trading System. It implements L16 (Risk Philosophy & Policy) from the
// EJLayer architecture plan.
//
// Policies are loaded from config/risk-constraints.json and enforced at two
// points in the trading pipeline:
//   1. Signal generation — stop-distance and per-trade risk checks filter
//      signals before they are stored or presented for approval.
//   2. Pre-execution — portfolio-level gates (open positions, daily loss,
//      account size) block order submission when a constraint is breached.
//
// A Violation carries a machine-readable Code so callers can log, alert, or
// route on specific breach types without string matching.
package risk

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

// ─── Policy ──────────────────────────────────────────────────────────────────

// PortfolioConstraints mirrors the "portfolio_constraints" block in
// config/risk-constraints.json.
type PortfolioConstraints struct {
	// MaxPositionSize is the maximum dollar value for a single position.
	MaxPositionSize float64 `json:"max_position_size"`
	// MaxPositions is the maximum number of open positions at any time.
	MaxPositions int `json:"max_positions"`
	// MaxSectorExposure is the maximum portfolio fraction in one sector (0–1).
	MaxSectorExposure float64 `json:"max_sector_exposure"`
	// MaxCorrelatedExposure is the maximum fraction in correlated positions (0–1).
	MaxCorrelatedExposure float64 `json:"max_correlated_exposure"`
	// MaxPortfolioRisk is the maximum total portfolio risk across all positions (0–1).
	MaxPortfolioRisk float64 `json:"max_portfolio_risk"`
	// MaxDrawdown is the maximum allowed drawdown before trading halts (0–1).
	MaxDrawdown float64 `json:"max_drawdown"`
	// MinAccountSize is the minimum net liquidation value required to trade.
	MinAccountSize float64 `json:"min_account_size"`
}

// PositionLimits mirrors the "position_limits" block.
type PositionLimits struct {
	// MaxRiskPerTrade is the maximum fraction of account to risk per trade (0–1).
	MaxRiskPerTrade float64 `json:"max_risk_per_trade"`
	// MinRiskPerTrade is the minimum fraction — signals below this are ignored.
	MinRiskPerTrade float64 `json:"min_risk_per_trade"`
	// MaxLeverage is the maximum gross leverage ratio.
	MaxLeverage float64 `json:"max_leverage"`
	// MinStopDistance is the minimum stop-loss distance as a fraction of entry price.
	MinStopDistance float64 `json:"min_stop_distance"`
	// MaxStopDistance is the maximum stop-loss distance as a fraction of entry price.
	MaxStopDistance float64 `json:"max_stop_distance"`
}

// Policy is the immutable, loaded risk policy.  It is created once at startup
// and passed read-only through the system.
type Policy struct {
	Portfolio   PortfolioConstraints `json:"portfolio_constraints"`
	Position    PositionLimits       `json:"position_limits"`
	SizingModel string               `json:"sizing_model"`
	// LoadedFrom is the file path the policy was read from (empty for defaults).
	LoadedFrom string `json:"-"`
	// LoadedAt is the wall-clock time the policy was loaded.
	LoadedAt time.Time `json:"-"`
	// Version is a hash of the serialised JSON, used for audit trail.
	Version string `json:"-"`
}

// LoadPolicy reads a JSON file and returns a validated Policy.
// Returns DefaultPolicy if path is empty or the file does not exist, so the
// system can start without a config file in development.
func LoadPolicy(path string) (*Policy, error) {
	if path == "" {
		p := DefaultPolicy()
		return p, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			p := DefaultPolicy()
			return p, nil
		}
		return nil, fmt.Errorf("risk: read policy file %q: %w", path, err)
	}

	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("risk: parse policy file %q: %w", path, err)
	}

	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("risk: invalid policy in %q: %w", path, err)
	}

	p.LoadedFrom = path
	p.LoadedAt = time.Now().UTC()
	p.Version = policyVersion(data)
	return &p, nil
}

// DefaultPolicy returns a safe conservative policy used when no file exists.
func DefaultPolicy() *Policy {
	p := &Policy{
		Portfolio: PortfolioConstraints{
			MaxPositionSize:       50_000,
			MaxPositions:          10,
			MaxSectorExposure:     0.30,
			MaxCorrelatedExposure: 0.40,
			MaxPortfolioRisk:      0.15,
			MaxDrawdown:           0.20,
			MinAccountSize:        10_000,
		},
		Position: PositionLimits{
			MaxRiskPerTrade: 0.02,
			MinRiskPerTrade: 0.005,
			MaxLeverage:     2.0,
			MinStopDistance: 0.01,
			MaxStopDistance: 0.10,
		},
		SizingModel: "fixed_fractional",
		LoadedFrom:  "",
		LoadedAt:    time.Now().UTC(),
	}
	b, _ := json.Marshal(p)
	p.Version = policyVersion(b)
	return p
}

func (p *Policy) validate() error {
	var errs []string

	if p.Position.MaxRiskPerTrade <= 0 || p.Position.MaxRiskPerTrade > 1 {
		errs = append(errs, fmt.Sprintf("max_risk_per_trade must be in (0,1], got %.4f", p.Position.MaxRiskPerTrade))
	}
	if p.Position.MinStopDistance < 0 || p.Position.MinStopDistance >= p.Position.MaxStopDistance {
		errs = append(errs, fmt.Sprintf("min_stop_distance (%.4f) must be < max_stop_distance (%.4f)", p.Position.MinStopDistance, p.Position.MaxStopDistance))
	}
	if p.Portfolio.MaxPositions <= 0 {
		errs = append(errs, "max_positions must be > 0")
	}
	if p.Portfolio.MaxDrawdown <= 0 || p.Portfolio.MaxDrawdown > 1 {
		errs = append(errs, fmt.Sprintf("max_drawdown must be in (0,1], got %.4f", p.Portfolio.MaxDrawdown))
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// policyVersion returns a short deterministic identifier for the policy JSON.
func policyVersion(data []byte) string {
	// Simple FNV-like hash for audit labelling — not a security hash.
	h := uint64(14695981039346656037)
	for _, b := range data {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return fmt.Sprintf("v%x", h&0xffffffffffff)
}

// ─── Violation ────────────────────────────────────────────────────────────────

// ViolationCode is a machine-readable identifier for a specific breach.
type ViolationCode string

const (
	ViolationStopTooTight      ViolationCode = "STOP_TOO_TIGHT"
	ViolationStopTooWide       ViolationCode = "STOP_TOO_WIDE"
	ViolationRiskTooHigh       ViolationCode = "RISK_PER_TRADE_TOO_HIGH"
	ViolationRiskTooLow        ViolationCode = "RISK_PER_TRADE_TOO_LOW"
	ViolationPositionTooLarge  ViolationCode = "POSITION_VALUE_TOO_LARGE"
	ViolationTooManyPositions  ViolationCode = "TOO_MANY_OPEN_POSITIONS"
	ViolationDailyLossExceeded ViolationCode = "DAILY_LOSS_EXCEEDED"
	ViolationAccountTooSmall   ViolationCode = "ACCOUNT_TOO_SMALL"
	ViolationDrawdownHalt      ViolationCode = "DRAWDOWN_HALT"
)

// Violation describes a single policy breach.
type Violation struct {
	Code    ViolationCode
	Message string
	// Policy value that was breached and the observed value.
	Limit    float64
	Observed float64
}

func (v Violation) Error() string {
	return fmt.Sprintf("risk violation [%s]: %s (limit=%.4f, observed=%.4f)",
		v.Code, v.Message, v.Limit, v.Observed)
}

// Violations is a slice of Violation that also satisfies the error interface.
type Violations []Violation

func (vs Violations) Error() string {
	msgs := make([]string, len(vs))
	for i, v := range vs {
		msgs[i] = v.Error()
	}
	return strings.Join(msgs, " | ")
}

// IsEmpty returns true when there are no violations.
func (vs Violations) IsEmpty() bool { return len(vs) == 0 }

// ─── SignalInput / PortfolioState ────────────────────────────────────────────

// SignalInput carries the signal-level values needed for pre-signal checks.
type SignalInput struct {
	Symbol     string
	EntryPrice float64
	StopLoss   float64
	// AccountEquity is the net liquidation value — used to compute risk fraction.
	AccountEquity float64
	// PositionValue is the dollar value of the proposed position.
	PositionValue float64
}

// PortfolioState carries current portfolio values needed for portfolio-level gates.
type PortfolioState struct {
	NetLiquidation float64
	OpenPositions  int
	DailyLossDollar float64
	// CurrentDrawdown is the current peak-to-trough drawdown fraction (0–1).
	CurrentDrawdown float64
}

// ─── Enforcer ────────────────────────────────────────────────────────────────

// Enforcer applies a Policy to signals and portfolio state.
// Construct one with NewEnforcer and reuse it across requests.
type Enforcer struct {
	policy *Policy
}

// NewEnforcer creates an Enforcer backed by the given Policy.
func NewEnforcer(policy *Policy) *Enforcer {
	return &Enforcer{policy: policy}
}

// Policy returns a read-only copy of the enforcer's policy (for logging/audit).
func (e *Enforcer) Policy() *Policy { return e.policy }

// CheckSignal validates a single signal against the per-trade position limits.
// Returns Violations (which is nil when there are no breaches).
func (e *Enforcer) CheckSignal(sig SignalInput) Violations {
	var vs Violations
	p := e.policy.Position

	if sig.EntryPrice <= 0 {
		return vs // skip — structural validation handles zero prices
	}

	// Stop-distance fraction
	stopDist := math.Abs(sig.EntryPrice-sig.StopLoss) / sig.EntryPrice

	if p.MinStopDistance > 0 && stopDist < p.MinStopDistance {
		vs = append(vs, Violation{
			Code:     ViolationStopTooTight,
			Message:  fmt.Sprintf("stop distance %.2f%% is below minimum %.2f%%", stopDist*100, p.MinStopDistance*100),
			Limit:    p.MinStopDistance,
			Observed: stopDist,
		})
	}

	if p.MaxStopDistance > 0 && stopDist > p.MaxStopDistance {
		vs = append(vs, Violation{
			Code:     ViolationStopTooWide,
			Message:  fmt.Sprintf("stop distance %.2f%% exceeds maximum %.2f%%", stopDist*100, p.MaxStopDistance*100),
			Limit:    p.MaxStopDistance,
			Observed: stopDist,
		})
	}

	// Per-trade risk fraction
	if sig.AccountEquity > 0 {
		riskDollar := math.Abs(sig.EntryPrice-sig.StopLoss) * (sig.PositionValue / sig.EntryPrice)
		riskFrac := riskDollar / sig.AccountEquity

		if p.MaxRiskPerTrade > 0 && riskFrac > p.MaxRiskPerTrade {
			vs = append(vs, Violation{
				Code:     ViolationRiskTooHigh,
				Message:  fmt.Sprintf("trade risk %.2f%% exceeds maximum %.2f%%", riskFrac*100, p.MaxRiskPerTrade*100),
				Limit:    p.MaxRiskPerTrade,
				Observed: riskFrac,
			})
		}

		if p.MinRiskPerTrade > 0 && riskFrac < p.MinRiskPerTrade {
			vs = append(vs, Violation{
				Code:     ViolationRiskTooLow,
				Message:  fmt.Sprintf("trade risk %.2f%% is below minimum %.2f%%", riskFrac*100, p.MinRiskPerTrade*100),
				Limit:    p.MinRiskPerTrade,
				Observed: riskFrac,
			})
		}
	}

	// Absolute position value cap
	pc := e.policy.Portfolio
	if pc.MaxPositionSize > 0 && sig.PositionValue > pc.MaxPositionSize {
		vs = append(vs, Violation{
			Code:     ViolationPositionTooLarge,
			Message:  fmt.Sprintf("position value $%.2f exceeds maximum $%.2f", sig.PositionValue, pc.MaxPositionSize),
			Limit:    pc.MaxPositionSize,
			Observed: sig.PositionValue,
		})
	}

	return vs
}

// CheckPortfolio validates the current portfolio state against portfolio-level
// constraints.  These gates block order submission, not signal generation.
func (e *Enforcer) CheckPortfolio(state PortfolioState) Violations {
	var vs Violations
	pc := e.policy.Portfolio

	// Minimum account size
	if pc.MinAccountSize > 0 && state.NetLiquidation < pc.MinAccountSize {
		vs = append(vs, Violation{
			Code:     ViolationAccountTooSmall,
			Message:  fmt.Sprintf("account equity $%.2f is below minimum $%.2f", state.NetLiquidation, pc.MinAccountSize),
			Limit:    pc.MinAccountSize,
			Observed: state.NetLiquidation,
		})
	}

	// Max open positions
	if pc.MaxPositions > 0 && state.OpenPositions >= pc.MaxPositions {
		vs = append(vs, Violation{
			Code:     ViolationTooManyPositions,
			Message:  fmt.Sprintf("open positions %d has reached maximum %d", state.OpenPositions, pc.MaxPositions),
			Limit:    float64(pc.MaxPositions),
			Observed: float64(state.OpenPositions),
		})
	}

	// Daily loss gate — expressed as fraction of net liquidation
	if pc.MaxPortfolioRisk > 0 && state.NetLiquidation > 0 {
		dailyLossFrac := state.DailyLossDollar / state.NetLiquidation
		if dailyLossFrac >= pc.MaxPortfolioRisk {
			vs = append(vs, Violation{
				Code:     ViolationDailyLossExceeded,
				Message:  fmt.Sprintf("daily loss %.2f%% has reached portfolio risk limit %.2f%%", dailyLossFrac*100, pc.MaxPortfolioRisk*100),
				Limit:    pc.MaxPortfolioRisk,
				Observed: dailyLossFrac,
			})
		}
	}

	// Drawdown halt
	if pc.MaxDrawdown > 0 && state.CurrentDrawdown >= pc.MaxDrawdown {
		vs = append(vs, Violation{
			Code:     ViolationDrawdownHalt,
			Message:  fmt.Sprintf("drawdown %.2f%% has reached halt threshold %.2f%%", state.CurrentDrawdown*100, pc.MaxDrawdown*100),
			Limit:    pc.MaxDrawdown,
			Observed: state.CurrentDrawdown,
		})
	}

	return vs
}
