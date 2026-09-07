package harness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

const ResearchBudgetContractV1 = "jax.research_budget/v1"

const (
	BudgetCostKnown         = "KNOWN"
	BudgetCostAmbiguous     = "AMBIGUOUS"
	BudgetTierDeterministic = "DETERMINISTIC"
	BudgetTierLocal         = "LOCAL"
	BudgetTierCheapHosted   = "CHEAP_HOSTED"
	BudgetTierStrongHosted  = "STRONG_HOSTED"
	BudgetTierPremium       = "PREMIUM"
)

type ResearchBudget struct {
	ContractVersion     string        `json:"contract_version"`
	Version             string        `json:"version"`
	MaxWallClock        time.Duration `json:"max_wall_clock"`
	ToolTimeout         time.Duration `json:"tool_timeout"`
	MaxSteps            int           `json:"max_steps"`
	MaxToolCalls        int           `json:"max_tool_calls"`
	MaxModelCalls       int           `json:"max_model_calls"`
	MaxRetries          int           `json:"max_retries"`
	MaxInputTokens      int           `json:"max_input_tokens"`
	MaxOutputTokens     int           `json:"max_output_tokens"`
	MaxReasoningTokens  int           `json:"max_reasoning_tokens"`
	MaxEstimatedCostUSD float64       `json:"max_estimated_cost_usd"`
	MaxActualCostUSD    float64       `json:"max_actual_cost_usd"`
	MaximumModelTier    string        `json:"maximum_model_tier"`
	RequireKnownCost    bool          `json:"require_known_cost"`
}

func NewResearchBudget(budget ResearchBudget) (ResearchBudget, error) {
	budget.ContractVersion = ResearchBudgetContractV1
	if err := budget.Validate(); err != nil {
		return ResearchBudget{}, err
	}
	return budget, nil
}

func (budget ResearchBudget) Validate() error {
	if budget.ContractVersion != ResearchBudgetContractV1 || strings.TrimSpace(budget.Version) == "" || budget.MaxWallClock <= 0 || budget.ToolTimeout <= 0 || budget.ToolTimeout > budget.MaxWallClock || budget.MaxSteps <= 0 || budget.MaxToolCalls < 0 || budget.MaxModelCalls < 0 || budget.MaxRetries < 0 || budget.MaxInputTokens < 0 || budget.MaxOutputTokens < 0 || budget.MaxReasoningTokens < 0 || !finiteBudgetNumber(budget.MaxEstimatedCostUSD) || budget.MaxEstimatedCostUSD < 0 || !finiteBudgetNumber(budget.MaxActualCostUSD) || budget.MaxActualCostUSD < 0 || strings.TrimSpace(budget.MaximumModelTier) == "" {
		return fmt.Errorf("research budget requires bounded durations, counts, costs and model tier")
	}
	if !validBudgetTier(budget.MaximumModelTier) {
		return fmt.Errorf("unsupported maximum model tier %q", budget.MaximumModelTier)
	}
	return nil
}

type BudgetState struct {
	Steps              int     `json:"steps"`
	ToolCalls          int     `json:"tool_calls"`
	ModelCalls         int     `json:"model_calls"`
	Retries            int     `json:"retries"`
	EstimatedInput     int     `json:"estimated_input_tokens"`
	EstimatedOutput    int     `json:"estimated_output_tokens"`
	EstimatedReasoning int     `json:"estimated_reasoning_tokens"`
	EstimatedCostUSD   float64 `json:"estimated_cost_usd"`
	ActualInput        int     `json:"actual_input_tokens"`
	ActualOutput       int     `json:"actual_output_tokens"`
	ActualReasoning    int     `json:"actual_reasoning_tokens"`
	ActualCostUSD      float64 `json:"actual_cost_usd"`
}

type BudgetController struct {
	budget ResearchBudget
	mu     sync.Mutex
	state  BudgetState
}

func NewBudgetController(budget ResearchBudget) (*BudgetController, error) {
	if err := budget.Validate(); err != nil {
		return nil, err
	}
	return &BudgetController{budget: budget}, nil
}

func (controller *BudgetController) Context(parent context.Context) (context.Context, context.CancelFunc, error) {
	if controller == nil {
		return nil, nil, fmt.Errorf("budget controller is required")
	}
	if err := parent.Err(); err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithTimeout(parent, controller.budget.MaxWallClock)
	return ctx, cancel, nil
}

func (controller *BudgetController) AdvanceStep() error {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if controller.state.Steps >= controller.budget.MaxSteps {
		return fmt.Errorf("research step budget exhausted")
	}
	controller.state.Steps++
	return nil
}

func (controller *BudgetController) ReserveToolCall() error {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if controller.state.ToolCalls >= controller.budget.MaxToolCalls {
		return fmt.Errorf("research tool-call budget exhausted")
	}
	controller.state.ToolCalls++
	return nil
}

func (controller *BudgetController) ReserveRetry() error {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if controller.state.Retries >= controller.budget.MaxRetries {
		return fmt.Errorf("research retry budget exhausted")
	}
	controller.state.Retries++
	return nil
}

func (controller *BudgetController) ReserveModelCall(tier string, inputTokens, outputTokens, reasoningTokens int, estimatedCostUSD float64) error {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if !validBudgetTier(tier) || !tierAtMost(tier, controller.budget.MaximumModelTier) {
		return fmt.Errorf("model tier is outside the research budget ceiling")
	}
	if inputTokens < 0 || outputTokens < 0 || reasoningTokens < 0 || !finiteBudgetNumber(estimatedCostUSD) || estimatedCostUSD < 0 {
		return fmt.Errorf("model call estimate is invalid")
	}
	if controller.state.ModelCalls >= controller.budget.MaxModelCalls || controller.state.EstimatedInput+inputTokens > controller.budget.MaxInputTokens || controller.state.EstimatedOutput+outputTokens > controller.budget.MaxOutputTokens || controller.state.EstimatedReasoning+reasoningTokens > controller.budget.MaxReasoningTokens || controller.state.EstimatedCostUSD+estimatedCostUSD > controller.budget.MaxEstimatedCostUSD {
		return fmt.Errorf("model-call budget exhausted")
	}
	controller.state.ModelCalls++
	controller.state.EstimatedInput += inputTokens
	controller.state.EstimatedOutput += outputTokens
	controller.state.EstimatedReasoning += reasoningTokens
	controller.state.EstimatedCostUSD += estimatedCostUSD
	return nil
}

func (controller *BudgetController) RecordUsage(usage NormalizedProviderUsage) error {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if controller.budget.RequireKnownCost && (usage.CostStatus == BudgetCostAmbiguous || !usage.InputTokensKnown || !usage.OutputTokensKnown || !usage.ReasoningKnown) {
		return fmt.Errorf("provider usage is incomplete and strict budget requires known cost and token categories")
	}
	if usage.InputTokens < 0 || usage.OutputTokens < 0 || usage.ReasoningTokens < 0 || !finiteBudgetNumber(usage.ActualCostUSD) || usage.ActualCostUSD < 0 || controller.state.ActualInput+usage.InputTokens > controller.budget.MaxInputTokens || controller.state.ActualOutput+usage.OutputTokens > controller.budget.MaxOutputTokens || controller.state.ActualReasoning+usage.ReasoningTokens > controller.budget.MaxReasoningTokens || controller.state.ActualCostUSD+usage.ActualCostUSD > controller.budget.MaxActualCostUSD {
		return fmt.Errorf("actual provider usage exceeds research budget")
	}
	controller.state.ActualInput += usage.InputTokens
	controller.state.ActualOutput += usage.OutputTokens
	controller.state.ActualReasoning += usage.ReasoningTokens
	controller.state.ActualCostUSD += usage.ActualCostUSD
	return nil
}

func (controller *BudgetController) State() BudgetState {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	return controller.state
}

type PricingSnapshot struct {
	Provider            string    `json:"provider"`
	Model               string    `json:"model"`
	EffectiveAt         time.Time `json:"effective_at"`
	InputUSDPer1K       float64   `json:"input_usd_per_1k"`
	CachedInputUSDPer1K float64   `json:"cached_input_usd_per_1k"`
	OutputUSDPer1K      float64   `json:"output_usd_per_1k"`
	ReasoningUSDPer1K   float64   `json:"reasoning_usd_per_1k"`
	Currency            string    `json:"currency"`
}

func (pricing PricingSnapshot) Validate() error {
	if strings.TrimSpace(pricing.Provider) == "" || strings.TrimSpace(pricing.Model) == "" || pricing.EffectiveAt.IsZero() || pricing.EffectiveAt.Location() != time.UTC || pricing.Currency != "USD" {
		return fmt.Errorf("pricing snapshot requires provider, model, UTC effective time and USD")
	}
	for _, value := range []float64{pricing.InputUSDPer1K, pricing.CachedInputUSDPer1K, pricing.OutputUSDPer1K, pricing.ReasoningUSDPer1K} {
		if !finiteBudgetNumber(value) || value < 0 {
			return fmt.Errorf("pricing rates must be finite and non-negative")
		}
	}
	return nil
}

type ProviderUsage struct {
	Provider          string
	Model             string
	InputTokens       *int
	CachedInputTokens *int
	CacheWriteTokens  *int
	OutputTokens      *int
	ReasoningTokens   *int
	ReportedCostUSD   *float64
	FinishStatus      string
	RawPayloadSHA256  string
}

type NormalizedProviderUsage struct {
	Provider          string  `json:"provider"`
	Model             string  `json:"model"`
	InputTokens       int     `json:"input_tokens"`
	InputTokensKnown  bool    `json:"input_tokens_known"`
	CachedInputTokens int     `json:"cached_input_tokens"`
	CachedInputKnown  bool    `json:"cached_input_known"`
	CacheWriteTokens  int     `json:"cache_write_tokens"`
	CacheWriteKnown   bool    `json:"cache_write_known"`
	OutputTokens      int     `json:"output_tokens"`
	OutputTokensKnown bool    `json:"output_tokens_known"`
	ReasoningTokens   int     `json:"reasoning_tokens"`
	ReasoningKnown    bool    `json:"reasoning_tokens_known"`
	ActualCostUSD     float64 `json:"actual_cost_usd"`
	CostStatus        string  `json:"cost_status"`
	FinishStatus      string  `json:"finish_status"`
	RawPayloadSHA256  string  `json:"raw_payload_sha256"`
}

func NormalizeProviderUsage(raw ProviderUsage, pricing *PricingSnapshot) (NormalizedProviderUsage, error) {
	if strings.TrimSpace(raw.Provider) == "" || strings.TrimSpace(raw.Model) == "" || !validSHA256Usage(raw.RawPayloadSHA256) || raw.InputTokens != nil && *raw.InputTokens < 0 || raw.CachedInputTokens != nil && *raw.CachedInputTokens < 0 || raw.CacheWriteTokens != nil && *raw.CacheWriteTokens < 0 || raw.OutputTokens != nil && *raw.OutputTokens < 0 || raw.ReasoningTokens != nil && *raw.ReasoningTokens < 0 {
		return NormalizedProviderUsage{}, fmt.Errorf("provider usage identity or counters are invalid")
	}
	result := NormalizedProviderUsage{Provider: raw.Provider, Model: raw.Model, FinishStatus: raw.FinishStatus, RawPayloadSHA256: raw.RawPayloadSHA256, CostStatus: BudgetCostAmbiguous}
	if raw.InputTokens != nil {
		result.InputTokens = *raw.InputTokens
		result.InputTokensKnown = true
	}
	if raw.CachedInputTokens != nil {
		result.CachedInputTokens = *raw.CachedInputTokens
		result.CachedInputKnown = true
		if result.CachedInputTokens > result.InputTokens {
			return NormalizedProviderUsage{}, fmt.Errorf("cached input tokens exceed input tokens")
		}
	}
	if raw.CacheWriteTokens != nil {
		result.CacheWriteTokens = *raw.CacheWriteTokens
		result.CacheWriteKnown = true
	}
	if raw.OutputTokens != nil {
		result.OutputTokens = *raw.OutputTokens
		result.OutputTokensKnown = true
	}
	if raw.ReasoningTokens != nil {
		result.ReasoningTokens = *raw.ReasoningTokens
		result.ReasoningKnown = true
	}
	if raw.ReportedCostUSD != nil {
		if !finiteBudgetNumber(*raw.ReportedCostUSD) || *raw.ReportedCostUSD < 0 {
			return NormalizedProviderUsage{}, fmt.Errorf("reported provider cost is invalid")
		}
		result.ActualCostUSD = *raw.ReportedCostUSD
		result.CostStatus = BudgetCostKnown
	} else if raw.InputTokens != nil && raw.CachedInputTokens != nil && raw.OutputTokens != nil && raw.ReasoningTokens != nil && pricing != nil {
		if err := pricing.Validate(); err != nil {
			return NormalizedProviderUsage{}, err
		}
		uncached := result.InputTokens - result.CachedInputTokens
		result.ActualCostUSD = float64(uncached)/1000*pricing.InputUSDPer1K + float64(result.CachedInputTokens)/1000*pricing.CachedInputUSDPer1K + float64(result.OutputTokens)/1000*pricing.OutputUSDPer1K + float64(result.ReasoningTokens)/1000*pricing.ReasoningUSDPer1K
		result.CostStatus = BudgetCostKnown
	}
	return result, nil
}

func validSHA256Usage(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func UsagePayloadHash(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}

func finiteBudgetNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validBudgetTier(tier string) bool {
	switch tier {
	case BudgetTierDeterministic, BudgetTierLocal, BudgetTierCheapHosted, BudgetTierStrongHosted, BudgetTierPremium:
		return true
	default:
		return false
	}
}

func tierAtMost(actual, maximum string) bool {
	order := map[string]int{BudgetTierDeterministic: 0, BudgetTierLocal: 1, BudgetTierCheapHosted: 2, BudgetTierStrongHosted: 3, BudgetTierPremium: 4}
	return order[actual] <= order[maximum]
}
