package aishadow

import (
	"fmt"
	"strings"
)

// OpenAIPricingSchedule identifies the immutable pricing snapshot used for a
// request. Prices are integer micro-USD per million tokens. The long-context
// rule is applied per request, never to an experiment aggregate.
type OpenAIPricingSchedule struct {
	Version                         string
	CheckedDate                     string
	Model                           string
	InputPriceMicrosPerMillion      int64
	CachedPriceMicrosPerMillion     int64
	CacheWritePriceMicrosPerMillion int64
	OutputPriceMicrosPerMillion     int64
	LongContextThresholdTokens      int
	LongContextInputNumerator       int64
	LongContextInputDenominator     int64
	LongContextOutputNumerator      int64
	LongContextOutputDenominator    int64
}

const (
	OpenAILunaPricingScheduleV1      = "openai:gpt-5.6-luna:standard-text:v1"
	OpenAILunaPricingCheckedDate     = "2026-09-09"
	OpenAILongContextThresholdTokens = 272_000
)

func OpenAIPricingScheduleVersion(model string) string {
	return "openai:" + strings.TrimSpace(model) + ":standard-text:v1"
}

// OpenAIRequestUsage is one provider response, including a retry. It is an
// accounting unit and must not be replaced by aggregate token totals.
type OpenAIRequestUsage struct {
	RequestID     string
	EventID       string
	AttemptNumber int
	Model         string
	Usage         ProviderUsage
}

type OpenAIRequestCost struct {
	RequestID                  string `json:"request_id,omitempty"`
	EventID                    string `json:"event_id,omitempty"`
	AttemptNumber              int    `json:"attempt_number,omitempty"`
	Model                      string `json:"model"`
	PricingScheduleVersion     string `json:"pricing_schedule_version"`
	PricingCheckedDate         string `json:"pricing_checked_date"`
	InputTokens                int    `json:"input_tokens"`
	CachedInputTokens          int    `json:"cached_input_tokens"`
	CacheWriteTokens           int    `json:"cache_write_tokens"`
	UncachedInputTokens        int    `json:"uncached_input_tokens"`
	OutputTokens               int    `json:"output_tokens"`
	BaseCostMicros             int64  `json:"base_cost_micros"`
	LongContextSurchargeMicros int64  `json:"long_context_surcharge_micros"`
	TotalCostMicros            int64  `json:"total_cost_micros"`
	LongContextApplied         bool   `json:"long_context_applied"`
}

type OpenAIExperimentCost struct {
	Requests        []OpenAIRequestCost `json:"requests"`
	TotalCostMicros int64               `json:"total_cost_micros"`
}

func (p OpenAIPricingSchedule) Validate(model string) error {
	if strings.TrimSpace(p.Version) == "" || strings.TrimSpace(p.CheckedDate) == "" || p.Model == "" || p.Model != model ||
		p.InputPriceMicrosPerMillion <= 0 || p.CachedPriceMicrosPerMillion <= 0 ||
		p.CacheWritePriceMicrosPerMillion <= 0 || p.OutputPriceMicrosPerMillion <= 0 ||
		p.LongContextThresholdTokens <= 0 || p.LongContextInputNumerator <= 0 || p.LongContextInputDenominator <= 0 ||
		p.LongContextOutputNumerator <= 0 || p.LongContextOutputDenominator <= 0 {
		return fmt.Errorf("invalid OpenAI pricing schedule")
	}
	return nil
}

func CalculateOpenAIRequestCost(request OpenAIRequestUsage, pricing OpenAIPricingSchedule) (OpenAIRequestCost, error) {
	if err := pricing.Validate(request.Model); err != nil {
		return OpenAIRequestCost{}, err
	}
	u := request.Usage
	if u.InputTokens <= 0 || u.OutputTokens < 0 || u.CachedTokens < 0 || u.CacheWriteTokens < 0 ||
		u.CachedTokens+u.CacheWriteTokens > u.InputTokens {
		return OpenAIRequestCost{}, fmt.Errorf("invalid OpenAI request usage")
	}
	uncached := u.InputTokens - u.CachedTokens - u.CacheWriteTokens
	if u.CacheMissTokens != 0 && u.CacheMissTokens != uncached {
		return OpenAIRequestCost{}, fmt.Errorf("cache-miss tokens do not reconcile with input tokens")
	}
	base := openAIRequestCostAtRates(u, uncached, pricing.InputPriceMicrosPerMillion,
		pricing.CachedPriceMicrosPerMillion, pricing.CacheWritePriceMicrosPerMillion,
		pricing.OutputPriceMicrosPerMillion)
	total := base
	long := u.InputTokens > pricing.LongContextThresholdTokens
	if long {
		longCost := openAIRequestCostAtRates(u, uncached,
			scaleRate(pricing.InputPriceMicrosPerMillion, pricing.LongContextInputNumerator, pricing.LongContextInputDenominator),
			scaleRate(pricing.CachedPriceMicrosPerMillion, pricing.LongContextInputNumerator, pricing.LongContextInputDenominator),
			scaleRate(pricing.CacheWritePriceMicrosPerMillion, pricing.LongContextInputNumerator, pricing.LongContextInputDenominator),
			scaleRate(pricing.OutputPriceMicrosPerMillion, pricing.LongContextOutputNumerator, pricing.LongContextOutputDenominator))
		total = longCost
	}
	return OpenAIRequestCost{
		RequestID: request.RequestID, EventID: request.EventID, AttemptNumber: request.AttemptNumber,
		Model: request.Model, PricingScheduleVersion: pricing.Version, PricingCheckedDate: pricing.CheckedDate,
		InputTokens: u.InputTokens, CachedInputTokens: u.CachedTokens, CacheWriteTokens: u.CacheWriteTokens,
		UncachedInputTokens: uncached, OutputTokens: u.OutputTokens, BaseCostMicros: base,
		LongContextSurchargeMicros: total - base, TotalCostMicros: total, LongContextApplied: long,
	}, nil
}

func CalculateOpenAIExperimentCost(requests []OpenAIRequestUsage, pricing OpenAIPricingSchedule) (OpenAIExperimentCost, error) {
	result := OpenAIExperimentCost{Requests: make([]OpenAIRequestCost, 0, len(requests))}
	for _, request := range requests {
		cost, err := CalculateOpenAIRequestCost(request, pricing)
		if err != nil {
			return OpenAIExperimentCost{}, err
		}
		result.Requests = append(result.Requests, cost)
		result.TotalCostMicros += cost.TotalCostMicros
	}
	return result, nil
}

// EstimateOpenAIRequestCost is deliberately conservative for pre-call budget
// reservation: it prices all input as the most expensive input category. The
// authoritative result is CalculateOpenAIRequestCost on provider usage.
func EstimateOpenAIRequestCost(model string, inputTokens, outputTokens int, pricing OpenAIPricingSchedule) int64 {
	if inputTokens <= 0 || outputTokens < 0 || pricing.Validate(model) != nil {
		return 0
	}
	rate := pricing.InputPriceMicrosPerMillion
	if pricing.CacheWritePriceMicrosPerMillion > rate {
		rate = pricing.CacheWritePriceMicrosPerMillion
	}
	// Represent the conservative estimate as uncached input at the maximum
	// input-category rate; this keeps the request-level long-context rule.
	estimatePricing := pricing
	estimatePricing.InputPriceMicrosPerMillion = rate
	estimatePricing.CachedPriceMicrosPerMillion = rate
	estimatePricing.CacheWritePriceMicrosPerMillion = rate
	cost, err := CalculateOpenAIRequestCost(OpenAIRequestUsage{Model: model, Usage: ProviderUsage{InputTokens: inputTokens, CacheMissTokens: inputTokens, OutputTokens: outputTokens}}, estimatePricing)
	if err != nil {
		return 0
	}
	return cost.TotalCostMicros
}

func openAIRequestCostAtRates(u ProviderUsage, uncached int, inputRate, cachedRate, cacheWriteRate, outputRate int64) int64 {
	return tokenCostMicros(uncached, inputRate) + tokenCostMicros(u.CachedTokens, cachedRate) +
		tokenCostMicros(u.CacheWriteTokens, cacheWriteRate) + tokenCostMicros(u.OutputTokens, outputRate)
}

func scaleRate(rate, numerator, denominator int64) int64 {
	return (rate*numerator + denominator - 1) / denominator
}
