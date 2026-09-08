// Package papertrading contains the isolated, deterministic Phase-11 paper
// execution domain. It does not expose a live broker client or mutate the
// Phase-09 observed portfolio.
package papertrading

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	CapabilityContractVersion = "jax.paper.capability/v1"
	OrderContractVersion      = "jax.paper.order/v1"
	FillContractVersion       = "jax.paper.fill/v1"
	LedgerContractVersion     = "jax.paper.ledger/v1"
	CostModelContractVersion  = "jax.paper.cost_model/v1"
	ReconciliationVersion     = "jax.paper.reconciliation/v1"
	AttributionVersion        = "jax.paper.attribution/v1"
	SoakProtocolVersion       = "jax.paper.soak/v1"
	ExecutionAuthorityNone    = "NONE"
	ExecutionStatusNotLive    = "PAPER_ONLY"
)

var (
	ErrInvalidContract        = errors.New("paper capability contract is invalid")
	ErrUnsupportedCapability  = errors.New("paper capability is unsupported")
	ErrInvalidEnvironment     = errors.New("paper execution environment is invalid")
	ErrLiveExecutionDisabled  = errors.New("live execution is disabled")
	ErrInvalidArtifact        = errors.New("paper artifact is invalid")
	ErrDuplicate              = errors.New("paper artifact is already recorded")
	ErrIdempotencyConflict    = errors.New("paper idempotency key conflicts")
	ErrInvalidMarketData      = errors.New("paper market data is invalid")
	ErrStaleMarketData        = errors.New("paper market data is stale")
	ErrMarketClosed           = errors.New("paper market is closed")
	ErrMarketUnknown          = errors.New("paper market session is unknown")
	ErrInsufficientCapital    = errors.New("paper account has insufficient capital")
	ErrOverfill               = errors.New("paper fill exceeds remaining quantity")
	ErrReconciliationRequired = errors.New("paper reconciliation is required")
	ErrOrderNotFound          = errors.New("paper order not found")
	ErrInvalidOrderTransition = errors.New("paper order transition is invalid")
	ErrPaperExecutionBlocked  = errors.New("paper execution is blocked by safety state")
)

type Environment string

const (
	EnvironmentUnknown Environment = ""
	EnvironmentPaper   Environment = "PAPER"
	EnvironmentLive    Environment = "LIVE"
)

func (environment Environment) Validate() error {
	if environment != EnvironmentPaper && environment != EnvironmentLive {
		return ErrInvalidEnvironment
	}
	return nil
}

type OrderType string

const (
	OrderMarket OrderType = "MARKET"
	OrderLimit  OrderType = "LIMIT"
)

func (orderType OrderType) Validate() error {
	if orderType != OrderMarket && orderType != OrderLimit {
		return fmt.Errorf("%w: order type %q", ErrUnsupportedCapability, orderType)
	}
	return nil
}

type Session string

const (
	SessionOpen    Session = "OPEN"
	SessionClosed  Session = "CLOSED"
	SessionUnknown Session = "UNKNOWN"
)

type CapabilityContract struct {
	VenueID                string        `json:"venue_id"`
	ContractVersion        string        `json:"contract_version"`
	Environment            Environment   `json:"environment"`
	SupportsMarket         bool          `json:"supports_market"`
	SupportsLimit          bool          `json:"supports_limit"`
	SupportsPartialFills   bool          `json:"supports_partial_fills"`
	SupportsCancellation   bool          `json:"supports_cancellation"`
	SupportsAmendment      bool          `json:"supports_amendment"`
	SupportsAccountState   bool          `json:"supports_account_state"`
	SupportsPositionState  bool          `json:"supports_position_state"`
	SupportsOrderStatus    bool          `json:"supports_order_status"`
	SupportsFillStatus     bool          `json:"supports_fill_status"`
	SupportsReconciliation bool          `json:"supports_reconciliation"`
	SupportsFractionalQty  bool          `json:"supports_fractional_quantity"`
	MinimumQuantity        float64       `json:"minimum_quantity"`
	PriceIncrement         float64       `json:"price_increment"`
	MarketHours            string        `json:"market_hours"`
	MaxQuoteAge            time.Duration `json:"max_quote_age"`
	ExecutionAuthority     string        `json:"execution_authority"`
	Algorithm              string        `json:"algorithm"`
}

func DefaultPaperCapabilityContract() CapabilityContract {
	return CapabilityContract{
		VenueID: "jax-deterministic-paper-venue", ContractVersion: CapabilityContractVersion,
		Environment: EnvironmentPaper, SupportsMarket: true, SupportsLimit: true,
		SupportsPartialFills: true, SupportsCancellation: true, SupportsAccountState: true,
		SupportsPositionState: true, SupportsOrderStatus: true, SupportsFillStatus: true,
		SupportsReconciliation: true, SupportsFractionalQty: true, MinimumQuantity: 0.000001,
		PriceIncrement: 0.000001, MarketHours: "EXPLICIT_TICK_SESSION", MaxQuoteAge: time.Minute,
		ExecutionAuthority: ExecutionAuthorityNone, Algorithm: "jax.paper.venue/v1",
	}
}

func (contract CapabilityContract) Validate() error {
	if strings.TrimSpace(contract.VenueID) == "" || contract.ContractVersion != CapabilityContractVersion || contract.Algorithm == "" {
		return ErrInvalidContract
	}
	if err := contract.Environment.Validate(); err != nil {
		return err
	}
	if contract.Environment == EnvironmentLive || contract.ExecutionAuthority != ExecutionAuthorityNone {
		return ErrLiveExecutionDisabled
	}
	if !contract.SupportsMarket && !contract.SupportsLimit {
		return fmt.Errorf("%w: no supported order type", ErrInvalidContract)
	}
	if !finitePositive(contract.MinimumQuantity) || !finitePositive(contract.PriceIncrement) || contract.MaxQuoteAge <= 0 {
		return fmt.Errorf("%w: quantity, price increment and quote age must be positive", ErrInvalidContract)
	}
	if !contract.SupportsReconciliation || !contract.SupportsOrderStatus || !contract.SupportsFillStatus {
		return fmt.Errorf("%w: audit/reconciliation capabilities are required", ErrInvalidContract)
	}
	return nil
}

func (contract CapabilityContract) Supports(orderType OrderType) bool {
	switch orderType {
	case OrderMarket:
		return contract.SupportsMarket
	case OrderLimit:
		return contract.SupportsLimit
	default:
		return false
	}
}

func capabilityIdentity(contract CapabilityContract) string {
	data, _ := json.Marshal(contract)
	digest := sha256.Sum256(data)
	return "pcap_" + hex.EncodeToString(digest[:])
}

func finitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
}

func finiteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}
