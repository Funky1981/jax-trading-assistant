package aishadow

import "testing"

func lunaCostSchedule() OpenAIPricingSchedule {
	return OpenAIPricingSchedule{
		Version: OpenAILunaPricingScheduleV1, CheckedDate: OpenAILunaPricingCheckedDate, Model: OpenAIDiagnosticLunaModel,
		InputPriceMicrosPerMillion: 200_000, CachedPriceMicrosPerMillion: 20_000,
		CacheWritePriceMicrosPerMillion: 250_000, OutputPriceMicrosPerMillion: 1_200_000,
		LongContextThresholdTokens: OpenAILongContextThresholdTokens,
		LongContextInputNumerator:  2, LongContextInputDenominator: 1,
		LongContextOutputNumerator: 3, LongContextOutputDenominator: 2,
	}
}

func TestCalculateOpenAIRequestCostOrdinaryUncached(t *testing.T) {
	cost, err := CalculateOpenAIRequestCost(OpenAIRequestUsage{RequestID: "req-1", Model: OpenAIDiagnosticLunaModel, Usage: ProviderUsage{InputTokens: 1000, CacheMissTokens: 1000, OutputTokens: 100}}, lunaCostSchedule())
	if err != nil {
		t.Fatal(err)
	}
	if cost.BaseCostMicros != 320 || cost.TotalCostMicros != 320 || cost.LongContextApplied {
		t.Fatalf("unexpected ordinary cost: %+v", cost)
	}
}

func TestCalculateOpenAIRequestCostCachedAndCacheWriteInput(t *testing.T) {
	usage := ProviderUsage{InputTokens: 1000, CachedTokens: 100, CacheWriteTokens: 200, CacheMissTokens: 700, OutputTokens: 100}
	cost, err := CalculateOpenAIRequestCost(OpenAIRequestUsage{RequestID: "req-2", Model: OpenAIDiagnosticLunaModel, Usage: usage}, lunaCostSchedule())
	if err != nil {
		t.Fatal(err)
	}
	want := tokenCostMicros(700, 200_000) + tokenCostMicros(100, 20_000) + tokenCostMicros(200, 250_000) + tokenCostMicros(100, 1_200_000)
	if cost.BaseCostMicros != want || cost.TotalCostMicros != want || cost.LongContextSurchargeMicros != 0 {
		t.Fatalf("cached/cache-write cost=%+v want=%d", cost, want)
	}
}

func TestCalculateOpenAIRequestCostLongContextBoundary(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input int
		long  bool
	}{
		{"just below", OpenAILongContextThresholdTokens - 1, false},
		{"exact boundary", OpenAILongContextThresholdTokens, false},
		{"above", OpenAILongContextThresholdTokens + 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cost, err := CalculateOpenAIRequestCost(OpenAIRequestUsage{RequestID: tc.name, Model: OpenAIDiagnosticLunaModel, Usage: ProviderUsage{InputTokens: tc.input, CacheMissTokens: tc.input, OutputTokens: 1}}, lunaCostSchedule())
			if err != nil {
				t.Fatal(err)
			}
			if cost.LongContextApplied != tc.long || (tc.long && cost.LongContextSurchargeMicros <= 0) || (!tc.long && cost.LongContextSurchargeMicros != 0) {
				t.Fatalf("boundary cost=%+v", cost)
			}
		})
	}
}

func TestCalculateOpenAIExperimentCostIncludesRetriesAndMixedRequests(t *testing.T) {
	pricing := lunaCostSchedule()
	requests := []OpenAIRequestUsage{
		{RequestID: "req-ordinary", EventID: "evt-1", AttemptNumber: 1, Model: pricing.Model, Usage: ProviderUsage{InputTokens: 1000, CacheMissTokens: 1000, OutputTokens: 10}},
		{RequestID: "req-retry", EventID: "evt-1", AttemptNumber: 2, Model: pricing.Model, Usage: ProviderUsage{InputTokens: 2000, CachedTokens: 100, CacheMissTokens: 1900, OutputTokens: 20}},
		{RequestID: "req-long", EventID: "evt-2", AttemptNumber: 1, Model: pricing.Model, Usage: ProviderUsage{InputTokens: OpenAILongContextThresholdTokens + 1, CacheMissTokens: OpenAILongContextThresholdTokens + 1, OutputTokens: 30}},
	}
	report, err := CalculateOpenAIExperimentCost(requests, pricing)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Requests) != len(requests) || report.TotalCostMicros != report.Requests[0].TotalCostMicros+report.Requests[1].TotalCostMicros+report.Requests[2].TotalCostMicros {
		t.Fatalf("aggregate did not equal request sum: %+v", report)
	}
	if report.Requests[1].AttemptNumber != 2 || report.Requests[2].LongContextSurchargeMicros == 0 {
		t.Fatalf("retry/long request evidence missing: %+v", report.Requests)
	}
}

func TestCalculateOpenAIRequestCostRejectsUsageMismatch(t *testing.T) {
	_, err := CalculateOpenAIRequestCost(OpenAIRequestUsage{Model: OpenAIDiagnosticLunaModel, Usage: ProviderUsage{InputTokens: 100, CacheMissTokens: 99, OutputTokens: 1}}, lunaCostSchedule())
	if err == nil {
		t.Fatal("inconsistent provider usage was accepted")
	}
}
