package evidencequality

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

func Evaluate(snapshot Snapshot, rules Ruleset, runtime RuntimeSafety) (Report, error) {
	if runtime.RuntimeMode != "paper" || runtime.AllowLiveTrading || runtime.ExecutionEnabled || runtime.ExecutionWorker || runtime.BrokerExecution || runtime.MaximumLeverage > 1 {
		return Report{}, fmt.Errorf("unsafe runtime state for historical evaluation")
	}
	if snapshot.SafetyBefore != snapshot.SafetyAfter {
		return Report{}, fmt.Errorf("prohibited record counts changed during evaluation")
	}
	index := newMarketIndex(snapshot.Candles)
	population, exclusions := BuildPopulation(snapshot.Events, index.end, rules)
	evaluated := make([]EvaluatedEvent, 0, len(population))
	for _, event := range population {
		mapping := MapEvent(event, rules)
		evaluated = append(evaluated, EvaluatedEvent{
			Event: event, SourceEventIdentity: event.SourceEventIdentity, Decision: event.Decision, DecisionOrigin: event.DecisionOrigin, DecisionContext: event.DecisionContext,
			EventType: emptyAsUnknown(event.EventType), SourceName: emptyAsUnknown(event.SourceName), Headline: event.Headline,
			PublicationAt: event.PublicationAt, CollectionAt: event.CollectionAt, ReceiptAt: event.ReceiptAt,
			DecisionAt: event.DecisionAt, Mapping: mapping, SubjectType: emptyAsUnknown(event.SubjectType),
			SubjectEventCount: event.SubjectEventCount, SourceGroupCount: event.SourceGroupCount,
			IndependentSources: event.IndependentSourceCount, PrimarySources: event.PrimarySourceCount,
			RepeatedSources: event.RepeatedSourceCount, MissingEvidence: event.MissingEvidence,
			Outcomes: calculateOutcomes(event, mapping, index, rules),
		})
	}
	populationSummary := summarizePopulation(snapshot.Events, evaluated, exclusions)
	metrics := metricRows(evaluated, rules)
	comparisons := comparisons(evaluated, rules)
	coverage := coverageRows(snapshot.Candles)
	report := Report{
		RulesetVersion: rules.Version, PrimaryAnchor: rules.PrimaryAnchor, Population: populationSummary,
		Exclusions: exclusions, MarketCoverage: coverage, HorizonCoverage: summarizeHorizonCoverage(evaluated, rules),
		Metrics: metrics, Comparisons: comparisons, Latency: latencyRows(evaluated, index, rules),
		Breakdowns: breakdownRows(evaluated), Misses: missRows(evaluated, DecisionNoTrade, false),
		WeakWatches: missRows(evaluated, DecisionWatch, true), EvidenceAccumulation: accumulationRows(evaluated),
		ExistingOutcomes: snapshot.ExistingOutcomes, SafetyBefore: snapshot.SafetyBefore, SafetyAfter: snapshot.SafetyAfter,
		RuntimeSafety: runtime, Events: evaluated,
	}
	report.Limitations = reportLimitations(report)
	report.Conclusion, report.ProductRecommendation, report.Verdict = reportConclusion(report, rules)
	fingerprintInput := struct {
		Ruleset Ruleset          `json:"ruleset"`
		Events  []EvaluatedEvent `json:"events"`
		Candles []CoverageRow    `json:"marketCoverage"`
		Safety  SafetyCounts     `json:"safety"`
	}{rules, evaluated, coverage, snapshot.SafetyBefore}
	raw, err := json.Marshal(fingerprintInput)
	if err != nil {
		return Report{}, fmt.Errorf("fingerprint evaluation input: %w", err)
	}
	digest := sha256.Sum256(raw)
	report.InputFingerprint = hex.EncodeToString(digest[:])
	return report, nil
}

func summarizePopulation(considered []Event, events []EvaluatedEvent, exclusions []Exclusion) PopulationSummary {
	result := PopulationSummary{Considered: len(considered), Included: len(events), Excluded: len(exclusions), DecisionCounts: map[string]int{}, CategoryCounts: map[string]int{}, SourceCounts: map[string]int{}, ExclusionCounts: map[string]int{}, OriginCounts: map[string]int{}}
	for _, event := range events {
		result.DecisionCounts[event.Decision]++
		result.CategoryCounts[event.EventType]++
		result.SourceCounts[event.SourceName]++
		result.OriginCounts[event.DecisionOrigin]++
		if event.Mapping.Mapped {
			result.Mapped++
		} else {
			result.Unmapped++
		}
		switch event.Decision {
		case DecisionWatch:
			result.Watch++
		case DecisionNoTrade:
			result.NoTrade++
		case DecisionCandidate:
			result.Candidate++
		}
		publication := event.PublicationAt
		if result.FirstPublication == nil || publication.Before(*result.FirstPublication) {
			value := publication
			result.FirstPublication = &value
		}
		if result.LastPublication == nil || publication.After(*result.LastPublication) {
			value := publication
			result.LastPublication = &value
		}
		receipt := event.ReceiptAt
		if result.FirstReceipt == nil || receipt.Before(*result.FirstReceipt) {
			value := receipt
			result.FirstReceipt = &value
		}
		if result.LastReceipt == nil || receipt.After(*result.LastReceipt) {
			value := receipt
			result.LastReceipt = &value
		}
		decisionAt := event.DecisionAt
		if result.FirstDecision == nil || decisionAt.Before(*result.FirstDecision) {
			value := decisionAt
			result.FirstDecision = &value
		}
		if result.LastDecision == nil || decisionAt.After(*result.LastDecision) {
			value := decisionAt
			result.LastDecision = &value
		}
	}
	for _, exclusion := range exclusions {
		result.ExclusionCounts[exclusion.Reason]++
	}
	return result
}

func summarizeHorizonCoverage(events []EvaluatedEvent, rules Ruleset) []HorizonCoverage {
	result := []HorizonCoverage{}
	for _, horizon := range []string{"1h", "1d", "1w"} {
		eventIDs := map[string]bool{}
		count := 0
		watch, noTrade := 0, 0
		for _, event := range events {
			found := false
			for _, outcome := range event.Outcomes {
				if outcome.Anchor == rules.PrimaryAnchor && outcome.Horizon == horizon {
					count++
					found = true
				}
			}
			if found {
				eventIDs[event.SourceEventIdentity] = true
				if event.Decision == DecisionWatch {
					watch++
				}
				if event.Decision == DecisionNoTrade {
					noTrade++
				}
			}
		}
		sufficient := watch >= rules.MinimumComparisonGroupSize && noTrade >= rules.MinimumComparisonGroupSize
		reason := "both WATCH and NO_TRADE have sufficient mapped outcomes"
		if !sufficient {
			reason = "insufficient mapped outcomes in one or both genuine decision groups"
		}
		result = append(result, HorizonCoverage{Horizon: horizon, OutcomeCount: count, EventCount: len(eventIDs), Sufficient: sufficient, Reason: reason})
	}
	return result
}

func reportLimitations(report Report) []string {
	limitations := []string{}
	if report.Population.NoTrade == 0 {
		limitations = append(limitations, "no genuine event-level NO_TRADE decisions remain after required test/proof exclusions")
	}
	if report.Population.Watch == 0 {
		limitations = append(limitations, "no genuine event-level WATCH decisions remain after exclusions")
	}
	if report.Population.Included > 0 && float64(report.Population.Unmapped)/float64(report.Population.Included) > 0.5 {
		limitations = append(limitations, "most included events have no conservative deterministic asset mapping")
	}
	for _, coverage := range report.HorizonCoverage {
		if !coverage.Sufficient {
			limitations = append(limitations, coverage.Horizon+" coverage cannot support a WATCH versus NO_TRADE comparison")
		}
	}
	if report.ExistingOutcomes.RecordCount > 0 {
		limitations = append(limitations, "existing outcome checkpoints belong to paper tickets and are not joined to this evaluation population")
	}
	limitations = append(limitations, "current subject NO_TRADE projections are not used as historical labels because staleness re-evaluation occurred after the outcome windows")
	if report.Population.OriginCounts["historical_backfill"] > 0 {
		limitations = append(limitations, "historical backfilled labels are reported separately and were not literally emitted at the event time")
	}
	return uniqueStrings(limitations)
}

func reportConclusion(report Report, rules Ruleset) (string, string, string) {
	comparisonAvailable := false
	for _, row := range report.Comparisons {
		if row.Anchor == rules.PrimaryAnchor && row.Available {
			comparisonAvailable = true
		}
	}
	if !comparisonAvailable {
		if report.Population.Included > 0 && report.Population.Unmapped*2 > report.Population.Included {
			return "timing/data quality prevents conclusion", "SOLVE ASSET RESOLUTION FURTHER", "PASS WITH LIMITATIONS"
		}
		return "insufficient data", "INSUFFICIENT DATA — COLLECT MORE", "PASS WITH LIMITATIONS"
	}
	return "weak evidence of separation", "REFINE SPECIFIC RULES", "PASS WITH LIMITATIONS"
}

func emptyAsUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return strings.TrimSpace(value)
}
func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func reportDateRange(report Report) (time.Time, time.Time) {
	if report.Population.FirstPublication == nil || report.Population.LastPublication == nil {
		return time.Time{}, time.Time{}
	}
	return *report.Population.FirstPublication, *report.Population.LastPublication
}
