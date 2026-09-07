package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

const CostSlippagePolicyContractV1 = "jax.cost_slippage_policy/v1"

type CostSlippagePolicy struct {
	ContractVersion    string    `json:"contract_version"`
	ID                 string    `json:"id"`
	Version            string    `json:"version"`
	CommissionPerOrder float64   `json:"commission_per_order"`
	CommissionBPS      float64   `json:"commission_bps"`
	SpreadBPS          float64   `json:"spread_bps"`
	SlippageBPS        float64   `json:"slippage_bps"`
	MarketImpactBPS    float64   `json:"market_impact_bps"`
	BorrowBPSPerDay    float64   `json:"borrow_bps_per_day"`
	ReferencePriceRule string    `json:"reference_price_rule"`
	DiagnosticBaseline bool      `json:"diagnostic_baseline"`
	AssumptionSource   string    `json:"assumption_source"`
	CreatedAt          time.Time `json:"created_at"`
}

func NewCostSlippagePolicy(policy CostSlippagePolicy) (CostSlippagePolicy, error) {
	policy.ContractVersion = CostSlippagePolicyContractV1
	policy.ID = deriveCostPolicyID(policy)
	if err := policy.Validate(); err != nil {
		return CostSlippagePolicy{}, err
	}
	return policy, nil
}

func (policy CostSlippagePolicy) Validate() error {
	if policy.ContractVersion != CostSlippagePolicyContractV1 || !validIdentity("cost_", policy.ID) || strings.TrimSpace(policy.Version) == "" || strings.TrimSpace(policy.ReferencePriceRule) == "" || strings.TrimSpace(policy.AssumptionSource) == "" || policy.CreatedAt.IsZero() || policy.CreatedAt.Location() != time.UTC {
		return fmt.Errorf("cost policy requires version, reference-price rule, source and UTC creation time")
	}
	values := []float64{policy.CommissionPerOrder, policy.CommissionBPS, policy.SpreadBPS, policy.SlippageBPS, policy.MarketImpactBPS, policy.BorrowBPSPerDay}
	for _, value := range values {
		if !finite(value) || value < 0 {
			return fmt.Errorf("cost policy rates must be finite and non-negative")
		}
	}
	if !policy.DiagnosticBaseline && allZero(values) {
		return fmt.Errorf("zero-friction policy is allowed only as an explicit diagnostic baseline")
	}
	if policy.ID != deriveCostPolicyID(policy) {
		return fmt.Errorf("cost policy ID does not match its assumptions")
	}
	return nil
}

type HypotheticalTradingCost struct {
	ContractVersion    string  `json:"contract_version"`
	PolicyID           string  `json:"policy_id"`
	Side               string  `json:"side"`
	ReferencePrice     float64 `json:"reference_price"`
	Quantity           float64 `json:"quantity"`
	Notional           float64 `json:"notional"`
	Commission         float64 `json:"commission"`
	SpreadCost         float64 `json:"spread_cost"`
	SlippageCost       float64 `json:"slippage_cost"`
	ImpactCost         float64 `json:"impact_cost"`
	BorrowCost         float64 `json:"borrow_cost"`
	TotalCost          float64 `json:"total_cost"`
	AdjustedPrice      float64 `json:"adjusted_price"`
	ExecutionAuthority string  `json:"execution_authority"`
	CreatesFill        bool    `json:"creates_fill"`
}

func ApplyHypotheticalTradingCost(policy CostSlippagePolicy, side string, referencePrice, quantity float64, holdingDays int) (HypotheticalTradingCost, error) {
	if err := policy.Validate(); err != nil {
		return HypotheticalTradingCost{}, err
	}
	if side != "BUY" && side != "SELL" || !finite(referencePrice) || referencePrice <= 0 || !finite(quantity) || quantity <= 0 || holdingDays < 0 {
		return HypotheticalTradingCost{}, fmt.Errorf("hypothetical cost requires a valid side, positive price and quantity")
	}
	notional := referencePrice * quantity
	commission := policy.CommissionPerOrder + notional*policy.CommissionBPS/10000
	spread := notional * policy.SpreadBPS / 10000
	slippage := notional * policy.SlippageBPS / 10000
	impact := notional * policy.MarketImpactBPS / 10000
	borrow := notional * policy.BorrowBPSPerDay / 10000 * float64(holdingDays)
	marketBPS := policy.SpreadBPS + policy.SlippageBPS + policy.MarketImpactBPS
	direction := 1.0
	if side == "SELL" {
		direction = -1
	}
	adjustedPrice := referencePrice * (1 + direction*marketBPS/10000)
	return HypotheticalTradingCost{ContractVersion: CostSlippagePolicyContractV1, PolicyID: policy.ID, Side: side, ReferencePrice: referencePrice, Quantity: quantity, Notional: notional, Commission: commission, SpreadCost: spread, SlippageCost: slippage, ImpactCost: impact, BorrowCost: borrow, TotalCost: commission + spread + slippage + impact + borrow, AdjustedPrice: adjustedPrice, ExecutionAuthority: "NONE", CreatesFill: false}, nil
}

func (cost HypotheticalTradingCost) Validate(policy CostSlippagePolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	if cost.ContractVersion != CostSlippagePolicyContractV1 || cost.PolicyID != policy.ID || (cost.Side != "BUY" && cost.Side != "SELL") || !finite(cost.ReferencePrice) || cost.ReferencePrice <= 0 || !finite(cost.Quantity) || cost.Quantity <= 0 || !finite(cost.Notional) || cost.Notional <= 0 || !finite(cost.TotalCost) || cost.TotalCost < 0 || !finite(cost.AdjustedPrice) || cost.AdjustedPrice <= 0 || cost.ExecutionAuthority != "NONE" || cost.CreatesFill {
		return fmt.Errorf("hypothetical trading cost is invalid or crosses execution boundary")
	}
	if !closeEnough(cost.Notional, cost.ReferencePrice*cost.Quantity) || !closeEnough(cost.TotalCost, cost.Commission+cost.SpreadCost+cost.SlippageCost+cost.ImpactCost+cost.BorrowCost) {
		return fmt.Errorf("hypothetical trading cost arithmetic is inconsistent")
	}
	for _, value := range []float64{cost.Commission, cost.SpreadCost, cost.SlippageCost, cost.ImpactCost, cost.BorrowCost} {
		if !finite(value) || value < 0 {
			return fmt.Errorf("hypothetical trading cost components must be finite and non-negative")
		}
	}
	return nil
}

func closeEnough(left, right float64) bool {
	return math.Abs(left-right) <= 1e-9*math.Max(1, math.Max(math.Abs(left), math.Abs(right)))
}

func allZero(values []float64) bool {
	for _, value := range values {
		if math.Abs(value) > 0 {
			return false
		}
	}
	return true
}

func deriveCostPolicyID(policy CostSlippagePolicy) string {
	policy.ID = ""
	seed, _ := json.Marshal(policy)
	digest := sha256.Sum256(seed)
	return "cost_" + hex.EncodeToString(digest[:])
}
