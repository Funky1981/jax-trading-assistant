package evidencequality

import (
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

func metricRows(events []EvaluatedEvent, rules Ruleset) []MetricRow {
	decisions := []string{DecisionWatch, DecisionNoTrade, DecisionCandidate}
	anchors := []string{"publication", "collection", "receipt", "decision"}
	horizons := []string{"1h", "1d", "1w"}
	rows := []MetricRow{}
	for _, decision := range decisions {
		for _, anchor := range anchors {
			for _, horizon := range horizons {
				outcomes := selectOutcomes(events, decision, anchor, horizon)
				row := MetricRow{Decision: decision, Anchor: anchor, Horizon: horizon, Count: len(outcomes)}
				if len(outcomes) > 0 {
					absolute := make([]float64, len(outcomes))
					abnormal := []float64{}
					mfe := make([]float64, len(outcomes))
					mae := make([]float64, len(outcomes))
					ranges := make([]float64, len(outcomes))
					aboveHalf, aboveOne, aboveTwo := 0, 0, 0
					for i, item := range outcomes {
						absolute[i] = item.AbsoluteRawReturn
						mfe[i] = item.MaximumFavourableExcursion
						mae[i] = item.MaximumAdverseExcursion
						ranges[i] = item.RealisedRange
						if item.AbsoluteAbnormalReturn != nil {
							abnormal = append(abnormal, *item.AbsoluteAbnormalReturn)
						}
						if item.AbsoluteRawReturn >= 0.005 {
							aboveHalf++
						}
						if item.AbsoluteRawReturn >= 0.01 {
							aboveOne++
						}
						if item.AbsoluteRawReturn >= 0.02 {
							aboveTwo++
						}
					}
					row.MedianAbsoluteReturn = pointer(median(absolute))
					row.MeanAbsoluteReturn = pointer(mean(absolute))
					row.ExceedPointFivePercent = pointer(float64(aboveHalf) / float64(len(outcomes)))
					row.ExceedOnePercent = pointer(float64(aboveOne) / float64(len(outcomes)))
					row.ExceedTwoPercent = pointer(float64(aboveTwo) / float64(len(outcomes)))
					row.MeanMaximumFavourableExcursion = pointer(mean(mfe))
					row.MeanMaximumAdverseExcursion = pointer(mean(mae))
					row.MeanRealisedRange = pointer(mean(ranges))
					if len(abnormal) > 0 {
						row.MedianAbsoluteAbnormalReturn = pointer(median(abnormal))
						row.MeanAbsoluteAbnormalReturn = pointer(mean(abnormal))
					}
					if len(absolute) >= rules.MinimumComparisonGroupSize {
						low, high := bootstrapMedianCI(absolute, rules.BootstrapIterations)
						row.MedianBootstrap95Low = pointer(low)
						row.MedianBootstrap95High = pointer(high)
					}
				}
				rows = append(rows, row)
			}
		}
	}
	return rows
}

func comparisons(events []EvaluatedEvent, rules Ruleset) []Comparison {
	result := []Comparison{}
	for _, anchor := range []string{"publication", "collection", "receipt", "decision"} {
		for _, horizon := range []string{"1h", "1d", "1w"} {
			watch := absoluteReturns(selectOutcomes(events, DecisionWatch, anchor, horizon))
			noTrade := absoluteReturns(selectOutcomes(events, DecisionNoTrade, anchor, horizon))
			row := Comparison{Anchor: anchor, Horizon: horizon, WatchCount: len(watch), NoTradeCount: len(noTrade)}
			if len(watch) >= rules.MinimumComparisonGroupSize && len(noTrade) >= rules.MinimumComparisonGroupSize {
				u := mannWhitneyU(watch, noTrade)
				p := permutationP(watch, noTrade, rules.PermutationIterations)
				delta := cliffsDelta(watch, noTrade)
				row.Available = true
				row.MannWhitneyU = &u
				row.PermutationP = &p
				row.CliffsDelta = &delta
			} else {
				row.Limitation = "both decision groups require at least " + strconv.Itoa(rules.MinimumComparisonGroupSize) + " observations"
			}
			result = append(result, row)
		}
	}
	return result
}

func latencyRows(events []EvaluatedEvent, index marketIndex, rules Ruleset) []LatencySummary {
	result := []LatencySummary{}
	for _, decision := range []string{DecisionWatch, DecisionNoTrade, DecisionCandidate} {
		pubCollection, collectionReceipt, receiptDecision, before, after := []float64{}, []float64{}, []float64{}, []float64{}, []float64{}
		count := 0
		for _, event := range events {
			if event.Decision != decision {
				continue
			}
			count++
			if event.CollectionAt != nil {
				pubCollection = append(pubCollection, event.CollectionAt.Sub(event.PublicationAt).Seconds())
				collectionReceipt = append(collectionReceipt, event.ReceiptAt.Sub(*event.CollectionAt).Seconds())
			}
			receiptDecision = append(receiptDecision, event.DecisionAt.Sub(event.ReceiptAt).Seconds())
			if event.Mapping.Mapped {
				maxDelay := time.Duration(rules.MaximumIntradayAnchorDelayMinutes) * time.Minute
				_, pubPrice, pubOK := index.intradayAnchorPrice(event.Mapping.Symbol, event.PublicationAt, maxDelay)
				_, receiptPrice, receiptOK := index.intradayAnchorPrice(event.Mapping.Symbol, event.ReceiptAt, maxDelay)
				if pubOK && receiptOK && pubPrice > 0 {
					before = append(before, math.Abs(receiptPrice/pubPrice-1))
				}
				if outcome, ok := index.outcome(event.Mapping.Symbol, event.ReceiptAt, "1h", maxDelay); ok {
					after = append(after, math.Abs(outcome.RawReturn))
				}
			}
		}
		row := LatencySummary{Decision: decision, Count: count, PublicationCollectionMedian: medianPointer(pubCollection), CollectionReceiptMedian: medianPointer(collectionReceipt), ReceiptDecisionMedian: medianPointer(receiptDecision), MoveBeforeReceiptMedian: medianPointer(before), MoveAfterReceiptMedian: medianPointer(after)}
		result = append(result, row)
	}
	return result
}

func breakdownRows(events []EvaluatedEvent) []BreakdownRow {
	type key struct{ dimension, value, decision string }
	groups := map[key][]EvaluatedEvent{}
	add := func(event EvaluatedEvent, dimension, value string) {
		if strings.TrimSpace(value) == "" {
			value = "unknown"
		}
		k := key{dimension, value, event.Decision}
		groups[k] = append(groups[k], event)
	}
	for _, event := range events {
		add(event, "event_category", event.EventType)
		add(event, "source", event.SourceName)
		add(event, "subject_type", event.SubjectType)
		if event.Mapping.Mapped {
			add(event, "resolved_asset_state", "mapped")
		} else {
			add(event, "resolved_asset_state", "unknown")
		}
		if event.PrimarySources > 0 {
			add(event, "source_role", "primary")
		} else {
			add(event, "source_role", "secondary_or_unknown")
		}
		add(event, "independent_source_group_count", strconv.Itoa(event.IndependentSources))
		add(event, "publication_to_receipt_latency", latencyBucket(event.ReceiptAt.Sub(event.PublicationAt)))
		for _, reason := range event.Event.Reasons {
			add(event, "decision_reason", reason)
		}
		for _, missing := range event.MissingEvidence {
			add(event, "missing_evidence", missing)
		}
	}
	result := make([]BreakdownRow, 0, len(groups))
	for k, items := range groups {
		result = append(result, makeBreakdown(k.dimension, k.value, k.decision, items))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Dimension != result[j].Dimension {
			return result[i].Dimension < result[j].Dimension
		}
		if result[i].Value != result[j].Value {
			return result[i].Value < result[j].Value
		}
		return result[i].Decision < result[j].Decision
	})
	return result
}

func accumulationRows(events []EvaluatedEvent) []BreakdownRow {
	type key struct{ dimension, value, decision string }
	groups := map[key][]EvaluatedEvent{}
	add := func(event EvaluatedEvent, d, v string) {
		groups[key{d, v, event.Decision}] = append(groups[key{d, v, event.Decision}], event)
	}
	for _, event := range events {
		if event.SubjectEventCount > 1 {
			add(event, "subject_event_count", "multi_event")
		} else {
			add(event, "subject_event_count", "single_event")
		}
		if event.IndependentSources > 0 {
			add(event, "source_independence", "independent_or_primary")
		} else if event.RepeatedSources > 0 {
			add(event, "source_independence", "repeated_same_group")
		} else {
			add(event, "source_independence", "unknown")
		}
	}
	result := []BreakdownRow{}
	for k, items := range groups {
		result = append(result, makeBreakdown(k.dimension, k.value, k.decision, items))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Dimension != result[j].Dimension {
			return result[i].Dimension < result[j].Dimension
		}
		if result[i].Value != result[j].Value {
			return result[i].Value < result[j].Value
		}
		return result[i].Decision < result[j].Decision
	})
	return result
}

func makeBreakdown(dimension, value, decision string, events []EvaluatedEvent) BreakdownRow {
	row := BreakdownRow{Dimension: dimension, Value: value, Decision: decision, Count: len(events)}
	moves := []float64{}
	for _, event := range events {
		if event.Mapping.Mapped {
			row.Mapped++
		}
		for _, outcome := range event.Outcomes {
			if outcome.Anchor == "receipt" && outcome.Horizon == "1d" {
				row.OutcomeCount++
				moves = append(moves, outcome.AbsoluteRawReturn)
			}
		}
	}
	row.MedianAbsoluteReturn = medianPointer(moves)
	return row
}

func missRows(events []EvaluatedEvent, decision string, weak bool) []MissRow {
	rows := []MissRow{}
	for _, event := range events {
		if event.Decision != decision {
			continue
		}
		for _, outcome := range event.Outcomes {
			if outcome.Anchor != "receipt" || outcome.Horizon != "1d" {
				continue
			}
			cause := "outcome available for manual rule review"
			if outcome.AnchorDelaySeconds > 12*3600 {
				cause = "market data begins materially after receipt"
			}
			if weak {
				cause = "WATCH produced a weak mapped one-day move"
			}
			reason := ""
			if len(event.Event.Reasons) > 0 {
				reason = event.Event.Reasons[0]
			}
			rows = append(rows, MissRow{SourceEventIdentity: event.SourceEventIdentity, Decision: event.Decision, Headline: event.Headline, Reason: reason, MissingEvidence: event.MissingEvidence, Symbol: event.Mapping.Symbol, Horizon: "1d", AbsoluteMove: outcome.AbsoluteRawReturn, ProbableCause: cause})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if weak {
			return rows[i].AbsoluteMove < rows[j].AbsoluteMove
		}
		return rows[i].AbsoluteMove > rows[j].AbsoluteMove
	})
	if len(rows) > 10 {
		return rows[:10]
	}
	return rows
}

func selectOutcomes(events []EvaluatedEvent, decision, anchor, horizon string) []Outcome {
	result := []Outcome{}
	for _, event := range events {
		if event.Decision != decision {
			continue
		}
		for _, outcome := range event.Outcomes {
			if outcome.Anchor == anchor && outcome.Horizon == horizon {
				result = append(result, outcome)
			}
		}
	}
	return result
}
func absoluteReturns(values []Outcome) []float64 {
	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = v.AbsoluteRawReturn
	}
	return result
}
func pointer(value float64) *float64 { return &value }
func medianPointer(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	return pointer(median(values))
}
func mean(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total / float64(len(values))
}
func median(values []float64) float64 {
	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	middle := len(copyValues) / 2
	if len(copyValues)%2 == 1 {
		return copyValues[middle]
	}
	return (copyValues[middle-1] + copyValues[middle]) / 2
}
func latencyBucket(value time.Duration) string {
	switch {
	case value < 15*time.Minute:
		return "under_15m"
	case value < time.Hour:
		return "15m_to_1h"
	case value < 6*time.Hour:
		return "1h_to_6h"
	case value < 24*time.Hour:
		return "6h_to_24h"
	default:
		return "over_24h"
	}
}

func bootstrapMedianCI(values []float64, iterations int) (float64, float64) {
	rng := rand.New(rand.NewSource(20260731))
	samples := make([]float64, iterations)
	draw := make([]float64, len(values))
	for i := 0; i < iterations; i++ {
		for j := range draw {
			draw[j] = values[rng.Intn(len(values))]
		}
		samples[i] = median(draw)
	}
	sort.Float64s(samples)
	return samples[int(float64(iterations-1)*0.025)], samples[int(float64(iterations-1)*0.975)]
}
func mannWhitneyU(a, b []float64) float64 {
	u := 0.0
	for _, x := range a {
		for _, y := range b {
			if x > y {
				u++
			} else if x == y {
				u += 0.5
			}
		}
	}
	return u
}
func cliffsDelta(a, b []float64) float64 {
	greater, less := 0, 0
	for _, x := range a {
		for _, y := range b {
			if x > y {
				greater++
			} else if x < y {
				less++
			}
		}
	}
	return float64(greater-less) / float64(len(a)*len(b))
}
func permutationP(a, b []float64, iterations int) float64 {
	observed := math.Abs(median(a) - median(b))
	combined := append(append([]float64{}, a...), b...)
	rng := rand.New(rand.NewSource(20260731))
	extreme := 0
	for i := 0; i < iterations; i++ {
		rng.Shuffle(len(combined), func(x, y int) { combined[x], combined[y] = combined[y], combined[x] })
		difference := math.Abs(median(combined[:len(a)]) - median(combined[len(a):]))
		if difference >= observed {
			extreme++
		}
	}
	return float64(extreme+1) / float64(iterations+1)
}
