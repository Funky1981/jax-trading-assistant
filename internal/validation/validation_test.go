package validation

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestEvaluatePromotionAllGatesPass(t *testing.T) {
	decision := EvaluatePromotion(PromotionInput{
		EpisodeFloor: 30, MatchedPairFloor: 30, EffectiveBlockFloor: 12, InstrumentFloor: 6, CalendarRegimeFloor: 3,
		EpisodeCount: 108, MatchedPairCount: 80, EffectiveBlockCount: 16, InstrumentCount: 9, CalendarRegimeSlices: 3,
		InstrumentConcentrationCeiling: .40, MaximumInstrumentContribution: .2222,
		PrimaryMeanNet: .008, PrimaryBootstrapLower: .001,
		MatchedMeanDifference: .002, PairedBootstrapLower: .001,
		RegisteredFalsifications: []FalsificationDisposition{{Name: "registered-null", Status: "PASS", Blocking: true}},
	})
	if decision.Classification != ForwardPaperEligible {
		t.Fatalf("classification = %s, want %s", decision.Classification, ForwardPaperEligible)
	}
	for _, gate := range decision.Gates {
		if gate.Status != GatePass {
			t.Errorf("gate %s = %s, want PASS (%s)", gate.Name, gate.Status, gate.Reason)
		}
	}
}

func TestEvaluatePromotionPositivePrimaryNegativeMatchedComparison(t *testing.T) {
	decision := completePromotionInput()
	decision.MatchedMeanDifference = -.01
	decision.PairedBootstrapLower = -.02
	result := EvaluatePromotion(decision)
	if result.Classification != FailedValidation {
		t.Fatalf("classification = %s, want FAILED_VALIDATION", result.Classification)
	}
	if status := gateStatus(result, "matched_mean_difference_positive"); status != GateFail {
		t.Fatalf("matched mean gate = %s, want FAIL", status)
	}
	if status := gateStatus(result, "paired_bootstrap_lower_positive"); status != GateFail {
		t.Fatalf("paired lower gate = %s, want FAIL", status)
	}
}

func TestEvaluatePromotionPositiveMatchedMeanNegativePairedLower(t *testing.T) {
	input := completePromotionInput()
	input.PairedBootstrapLower = -.001
	result := EvaluatePromotion(input)
	if result.Classification != FailedValidation {
		t.Fatalf("classification = %s, want FAILED_VALIDATION", result.Classification)
	}
	if status := gateStatus(result, "matched_mean_difference_positive"); status != GatePass {
		t.Fatalf("matched mean gate = %s, want PASS", status)
	}
	if status := gateStatus(result, "paired_bootstrap_lower_positive"); status != GateFail {
		t.Fatalf("paired lower gate = %s, want FAIL", status)
	}
}

func TestEvaluatePromotionInsufficientPairCount(t *testing.T) {
	input := completePromotionInput()
	input.MatchedPairCount = 2
	result := EvaluatePromotion(input)
	if result.Classification != InsufficientEvidence {
		t.Fatalf("classification = %s, want INSUFFICIENT_EVIDENCE", result.Classification)
	}
	for _, name := range []string{"matched_pair_floor", "matched_mean_difference_positive", "paired_bootstrap_lower_positive"} {
		if status := gateStatus(result, name); status != GateInsufficient {
			t.Errorf("%s = %s, want INSUFFICIENT", name, status)
		}
	}
}

func TestEvaluatePromotionInsufficientBlocks(t *testing.T) {
	input := completePromotionInput()
	input.EffectiveBlockCount = 1
	result := EvaluatePromotion(input)
	if result.Classification != InsufficientEvidence || gateStatus(result, "effective_block_floor") != GateInsufficient {
		t.Fatalf("decision = %+v, want insufficient block evidence", result)
	}
}

func TestEvaluatePromotionExcessiveConcentration(t *testing.T) {
	input := completePromotionInput()
	input.MaximumInstrumentContribution = .41
	result := EvaluatePromotion(input)
	if result.Classification != FailedValidation || gateStatus(result, "instrument_concentration_ceiling") != GateFail {
		t.Fatalf("decision = %+v, want concentration failure", result)
	}
}

func TestEvaluatePromotionBlockingFalsificationFailure(t *testing.T) {
	input := completePromotionInput()
	input.RegisteredFalsifications = []FalsificationDisposition{{Name: "top_5_percent_exclusion", Status: "FAIL", Blocking: true}}
	result := EvaluatePromotion(input)
	if result.Classification != FailedValidation || gateStatus(result, "registered_blocking_falsifications") != GateFail {
		t.Fatalf("decision = %+v, want blocking falsification failure", result)
	}
}

func TestEvaluatePromotionUnknownDispositionFailsClosed(t *testing.T) {
	input := completePromotionInput()
	input.RegisteredFalsifications = []FalsificationDisposition{{Name: "future-test", Status: "NOT_APPLICABLE", Blocking: true}}
	result := EvaluatePromotion(input)
	if result.Classification != InsufficientEvidence || gateStatus(result, "registered_blocking_falsifications") != GateInsufficient {
		t.Fatalf("decision = %+v, want fail-closed insufficient evidence", result)
	}
}

func TestEvaluatePromotionMultipleSimultaneousFailures(t *testing.T) {
	input := completePromotionInput()
	input.EpisodeCount = 1
	input.MaximumInstrumentContribution = .8
	input.PrimaryMeanNet = -.1
	input.RegisteredFalsifications = []FalsificationDisposition{{Name: "blocking-test", Status: "FAIL", Blocking: true}}
	result := EvaluatePromotion(input)
	if result.Classification != FailedValidation {
		t.Fatalf("classification = %s, want FAILED_VALIDATION", result.Classification)
	}
	failures := 0
	for _, gate := range result.Gates {
		failures += boolToInt(gate.Status == GateFail)
	}
	if failures < 3 {
		t.Fatalf("only %d independent failures exposed: %+v", failures, result.Gates)
	}
}

func TestEvaluateTopFiveExclusionUsesNeutralReasonAndExposesBothEffects(t *testing.T) {
	result := EvaluateTopFiveExclusion(TopFiveExclusionInput{
		PrimaryMeanBefore: .008, PrimaryMeanAfter: .004,
		PairedMeanBefore: -.01, PairedMeanAfter: -.013,
		RemovedCount: 6, RemainingPairs: 77, PairedPairFloor: 30,
	})
	if result.Disposition != GateFail {
		t.Fatalf("disposition = %s, want FAIL", result.Disposition)
	}
	if result.PrimaryMeanBefore != .008 || result.PrimaryMeanAfter != .004 || result.PairedMeanBefore != -.01 || result.PairedMeanAfter != -.013 || result.RemovedCount != 6 || result.RemainingPairs != 77 {
		t.Fatalf("metrics not preserved: %+v", result)
	}
	if strings.Contains(strings.ToLower(result.Reason), "eliminates") || strings.Contains(strings.ToLower(result.Reason), "caused") {
		t.Fatalf("causal wording leaked into reason: %q", result.Reason)
	}
}

func TestEvaluateTopFiveExclusionInsufficientPairs(t *testing.T) {
	result := EvaluateTopFiveExclusion(TopFiveExclusionInput{PrimaryMeanAfter: .1, PairedMeanAfter: .1, RemainingPairs: 2, PairedPairFloor: 30})
	if result.Disposition != GateInsufficient {
		t.Fatalf("disposition = %s, want INSUFFICIENT", result.Disposition)
	}
}

func TestTimestampPlaceboReferenceEquivalence(t *testing.T) {
	dates := map[string][]string{
		"BBB": {"2024-01-02", "2024-01-03", "2024-01-04", "2025-01-02", "2025-01-03"},
		"AAA": {"2024-01-01", "2024-01-02", "2024-01-03", "2025-01-01", "2025-01-02", "2025-01-03"},
	}
	excluded := map[string]TimestampEligibility{
		"AAA|2024-01-02": {RecoveryDateEligible: true, ActualSignalExcluded: true, StructuralEligible: true},
		"AAA|2024-01-03": {RecoveryDateEligible: true, ActiveIntervalExcluded: true, StructuralEligible: true},
		"BBB|2025-01-02": {RecoveryDateEligible: true, StructuralEligible: false},
		"AAA|2025-01-03": {RecoveryDateEligible: false, StructuralEligible: true},
	}
	evaluator := func(instrument, date string, _ int) TimestampEligibility {
		if value, ok := excluded[instrument+"|"+date]; ok {
			return value
		}
		return TimestampEligibility{RecoveryDateEligible: true, StructuralEligible: true}
	}
	prepared, err := PrepareTimestampPools(dates, "2024-01-01", "2025-01-03", false, evaluator)
	if err != nil {
		t.Fatal(err)
	}
	optimized := prepared.Select("manifest-fixture", 17)
	reference := referenceTimestampSelections(dates, "2024-01-01", "2025-01-03", "manifest-fixture", 17, evaluator)
	if !reflect.DeepEqual(optimized, reference) {
		t.Fatalf("optimized output differs from reference\noptimized=%s\nreference=%s", mustJSON(optimized), mustJSON(reference))
	}
	if len(optimized) != 4*17 {
		t.Fatalf("selection count = %d, want %d", len(optimized), 4*17)
	}
	if SelectionDigest(optimized) != SelectionDigest(reference) {
		t.Fatal("selection digest differs from reference")
	}
	for i, selection := range optimized {
		if selection.Replicate != reference[i].Replicate || selection.Instrument != reference[i].Instrument || selection.Year != reference[i].Year || selection.Date != reference[i].Date {
			t.Fatalf("selection %d differs: %+v vs %+v", i, selection, reference[i])
		}
	}
}

func TestTimestampPlaceboPrecomputesEligibility(t *testing.T) {
	dates := map[string][]string{"SPY": {"2024-01-01", "2024-01-02", "2024-01-03", "2024-01-04"}}
	calls := 0
	prepared, err := PrepareTimestampPools(dates, "2024-01-01", "2024-01-04", false, func(string, string, int) TimestampEligibility {
		calls++
		return TimestampEligibility{RecoveryDateEligible: true, StructuralEligible: true}
	})
	if err != nil {
		t.Fatal(err)
	}
	selections := prepared.Select("manifest", 100)
	if calls != 4 || prepared.Stats.EligibilityEvaluations != 4 {
		t.Fatalf("eligibility evaluations = calls %d/stats %d, want 4", calls, prepared.Stats.EligibilityEvaluations)
	}
	if len(selections) != 100 {
		t.Fatalf("selection count = %d, want 100", len(selections))
	}
}

func TestTimestampPlaceboBoundaryAndFutureMatching(t *testing.T) {
	if _, err := PrepareTimestampPools(map[string][]string{"SPY": {"2024-01-01"}}, "2024-01-02", "2024-01-01", false, nil); err == nil {
		t.Fatal("reversed boundary unexpectedly accepted")
	}
	if _, err := PrepareTimestampPools(map[string][]string{"SPY": {"2024-01-01"}}, "2024-01-01", "2024-01-01", true, nil); err == nil {
		t.Fatal("future_matching=true unexpectedly accepted")
	}
}

func completePromotionInput() PromotionInput {
	return PromotionInput{
		EpisodeFloor: 30, MatchedPairFloor: 30, EffectiveBlockFloor: 12, InstrumentFloor: 6, CalendarRegimeFloor: 3,
		EpisodeCount: 100, MatchedPairCount: 80, EffectiveBlockCount: 16, InstrumentCount: 9, CalendarRegimeSlices: 3,
		InstrumentConcentrationCeiling: .40, MaximumInstrumentContribution: .22,
		PrimaryMeanNet: .008, PrimaryBootstrapLower: .001,
		MatchedMeanDifference: .002, PairedBootstrapLower: .001,
		RegisteredFalsifications: []FalsificationDisposition{{Name: "blocking-test", Status: "PASS", Blocking: true}},
	}
}

func gateStatus(decision PromotionDecision, name string) GateStatus {
	for _, gate := range decision.Gates {
		if gate.Name == name {
			return gate.Status
		}
	}
	return "MISSING"
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// referenceTimestampSelections is test-only and intentionally mirrors the
// frozen per-replicate scan + complete sort algorithm. It is never called by
// production code or any formal runner.
func referenceTimestampSelections(dates map[string][]string, start, end, manifest string, replicates int, evaluator TimestampEligibilityEvaluator) []TimestampSelection {
	instruments := make([]string, 0, len(dates))
	years := map[int]struct{}{}
	for instrument := range dates {
		instruments = append(instruments, instrument)
		for _, date := range dates[instrument] {
			if date >= start && date <= end && len(date) >= 4 {
				years[parseYear(date)] = struct{}{}
			}
		}
	}
	sort.Strings(instruments)
	orderedYears := make([]int, 0, len(years))
	for year := range years {
		orderedYears = append(orderedYears, year)
	}
	sort.Ints(orderedYears)
	result := []TimestampSelection{}
	for replicate := 0; replicate < replicates; replicate++ {
		for _, instrument := range instruments {
			for _, year := range orderedYears {
				candidates := []string{}
				for _, date := range dates[instrument] {
					if date < start || date > end || len(date) < 4 || parseYear(date) != year {
						continue
					}
					eligibility := TimestampEligibility{RecoveryDateEligible: true, StructuralEligible: true}
					if evaluator != nil {
						eligibility = evaluator(instrument, date, year)
					}
					if eligibility.RecoveryDateEligible && !eligibility.ActualSignalExcluded && !eligibility.ActiveIntervalExcluded && eligibility.StructuralEligible {
						candidates = append(candidates, date)
					}
				}
				sort.Slice(candidates, func(i, j int) bool {
					a := timestampPlaceboScore(manifest, replicate, instrument, candidates[i])
					b := timestampPlaceboScore(manifest, replicate, instrument, candidates[j])
					if a != b {
						return a < b
					}
					return candidates[i] < candidates[j]
				})
				if len(candidates) > 0 {
					result = append(result, TimestampSelection{Replicate: replicate, Instrument: instrument, Year: year, Date: candidates[0]})
				}
			}
		}
	}
	return result
}

func mustJSON(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func BenchmarkTimestampPlaceboReference(b *testing.B) {
	dates := benchmarkDates()
	evaluator := benchmarkEvaluator
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		referenceTimestampSelections(dates, "2024-01-01", "2025-12-31", "benchmark-manifest", 100, evaluator)
	}
}

func BenchmarkTimestampPlaceboOptimized(b *testing.B) {
	prepared, err := PrepareTimestampPools(benchmarkDates(), "2024-01-01", "2025-12-31", false, benchmarkEvaluator)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prepared.Select("benchmark-manifest", 100)
	}
}

func benchmarkDates() map[string][]string {
	dates := map[string][]string{}
	for _, instrument := range []string{"AAA", "BBB", "CCC", "DDD"} {
		values := make([]string, 0, 500)
		for year := 2024; year <= 2025; year++ {
			for day := 1; day <= 250; day++ {
				values = append(values, benchmarkDate(year, day))
			}
		}
		dates[instrument] = values
	}
	return dates
}

func benchmarkDate(year, day int) string {
	month := (day-1)/28 + 1
	dayOfMonth := (day-1)%28 + 1
	return fmtDate(year, month, dayOfMonth)
}

func fmtDate(year, month, day int) string {
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func benchmarkEvaluator(_ string, date string, _ int) TimestampEligibility {
	return TimestampEligibility{RecoveryDateEligible: true, StructuralEligible: date != "2024-02-05"}
}
