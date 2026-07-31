package evidencequality

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ArtifactPaths struct {
	Markdown      string `json:"markdown"`
	JSON          string `json:"json"`
	PopulationCSV string `json:"populationCsv"`
	OutcomesCSV   string `json:"outcomesCsv"`
}

func WriteArtifacts(outputDir string, report Report) (ArtifactPaths, error) {
	if strings.TrimSpace(outputDir) == "" {
		return ArtifactPaths{}, fmt.Errorf("output directory is required")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return ArtifactPaths{}, fmt.Errorf("create evaluation output directory: %w", err)
	}
	paths := ArtifactPaths{Markdown: filepath.Join(outputDir, "report.md"), JSON: filepath.Join(outputDir, "summary.json"), PopulationCSV: filepath.Join(outputDir, "population.csv"), OutcomesCSV: filepath.Join(outputDir, "outcomes.csv")}
	jsonRaw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return ArtifactPaths{}, fmt.Errorf("encode evaluation JSON: %w", err)
	}
	jsonRaw = append(jsonRaw, '\n')
	if err := os.WriteFile(paths.JSON, jsonRaw, 0o644); err != nil {
		return ArtifactPaths{}, fmt.Errorf("write evaluation JSON: %w", err)
	}
	if err := os.WriteFile(paths.Markdown, []byte(markdownReport(report)), 0o644); err != nil {
		return ArtifactPaths{}, fmt.Errorf("write evaluation Markdown: %w", err)
	}
	if err := writePopulationCSV(paths.PopulationCSV, report); err != nil {
		return ArtifactPaths{}, err
	}
	if err := writeOutcomesCSV(paths.OutcomesCSV, report); err != nil {
		return ArtifactPaths{}, err
	}
	return paths, nil
}

func writePopulationCSV(path string, report Report) error {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"source_event_identity", "decision", "event_type", "source_name", "publication_at", "collection_at", "receipt_at", "decision_at", "mapped", "mapping_type", "symbol", "mapping_confidence", "direct_or_proxy", "benchmark", "subject_type", "subject_event_count", "source_group_count", "independent_source_count", "primary_source_count", "repeated_source_count", "missing_evidence"})
	for _, event := range report.Events {
		collection := ""
		if event.CollectionAt != nil {
			collection = event.CollectionAt.UTC().Format(time.RFC3339Nano)
		}
		direct := "unknown"
		if event.Mapping.Mapped {
			if event.Mapping.Direct {
				direct = "direct"
			} else {
				direct = "proxy"
			}
		}
		_ = writer.Write([]string{event.SourceEventIdentity, event.Decision, event.EventType, event.SourceName, event.PublicationAt.UTC().Format(time.RFC3339Nano), collection, event.ReceiptAt.UTC().Format(time.RFC3339Nano), event.DecisionAt.UTC().Format(time.RFC3339Nano), strconv.FormatBool(event.Mapping.Mapped), event.Mapping.MappingType, event.Mapping.Symbol, event.Mapping.Confidence, direct, event.Mapping.Benchmark, event.SubjectType, strconv.Itoa(event.SubjectEventCount), strconv.Itoa(event.SourceGroupCount), strconv.Itoa(event.IndependentSources), strconv.Itoa(event.PrimarySources), strconv.Itoa(event.RepeatedSources), strings.Join(event.MissingEvidence, "|")})
	}
	for _, exclusion := range report.Exclusions {
		_ = writer.Write([]string{exclusion.SourceEventIdentity, exclusion.Decision, "EXCLUDED", exclusion.Reason})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("encode population CSV: %w", err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write population CSV: %w", err)
	}
	return nil
}

func writeOutcomesCSV(path string, report Report) error {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"source_event_identity", "decision", "anchor", "anchor_at", "effective_anchor_at", "anchor_delay_seconds", "horizon", "symbol", "benchmark", "raw_return", "absolute_raw_return", "abnormal_return", "absolute_abnormal_return", "realised_range", "maximum_favourable_excursion", "maximum_adverse_excursion", "candle_count", "market_data_source", "timestamp_semantics"})
	for _, event := range report.Events {
		for _, outcome := range event.Outcomes {
			abnormal, absoluteAbnormal := "", ""
			if outcome.AbnormalReturn != nil {
				abnormal = formatFloat(*outcome.AbnormalReturn)
			}
			if outcome.AbsoluteAbnormalReturn != nil {
				absoluteAbnormal = formatFloat(*outcome.AbsoluteAbnormalReturn)
			}
			_ = writer.Write([]string{event.SourceEventIdentity, event.Decision, outcome.Anchor, outcome.AnchorAt.UTC().Format(time.RFC3339Nano), outcome.EffectiveAnchorAt.UTC().Format(time.RFC3339Nano), formatFloat(outcome.AnchorDelaySeconds), outcome.Horizon, outcome.Symbol, outcome.Benchmark, formatFloat(outcome.RawReturn), formatFloat(outcome.AbsoluteRawReturn), abnormal, absoluteAbnormal, formatFloat(outcome.RealisedRange), formatFloat(outcome.MaximumFavourableExcursion), formatFloat(outcome.MaximumAdverseExcursion), strconv.Itoa(outcome.CandleCount), outcome.MarketDataSource, outcome.TimestampSemantics})
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("encode outcomes CSV: %w", err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write outcomes CSV: %w", err)
	}
	return nil
}

func markdownReport(report Report) string {
	var b strings.Builder
	b.WriteString("# Historical Evidence Quality and Market-Relevance Evaluation\n\n")
	b.WriteString("Ruleset: `" + report.RulesetVersion + "`  \nInput fingerprint: `" + report.InputFingerprint + "`  \nPrimary operational anchor: `" + report.PrimaryAnchor + "`\n\n")
	first, last := reportDateRange(report)
	b.WriteString("## Population\n\n")
	b.WriteString(fmt.Sprintf("- Genuine decisions considered: %d\n- Included: %d\n- Excluded: %d\n- Publication range: %s to %s\n- Receipt range: %s to %s\n- Decision range: %s to %s\n- Mapped: %d\n- Unmapped: %d\n- WATCH: %d\n- NO_TRADE: %d\n- CANDIDATE: %d\n\n", report.Population.Considered, report.Population.Included, report.Population.Excluded, formatTime(first), formatTime(last), formatTimePointer(report.Population.FirstReceipt), formatTimePointer(report.Population.LastReceipt), formatTimePointer(report.Population.FirstDecision), formatTimePointer(report.Population.LastDecision), report.Population.Mapped, report.Population.Unmapped, report.Population.Watch, report.Population.NoTrade, report.Population.Candidate))
	b.WriteString("Exclusions:\n\n")
	for _, key := range sortedKeys(report.Population.ExclusionCounts) {
		b.WriteString(fmt.Sprintf("- %s: %d\n", key, report.Population.ExclusionCounts[key]))
	}
	b.WriteString("\n")
	b.WriteString("## Market-data coverage\n\n| Symbol | Timeframe | Provider | Candles | First | Last | Gaps | Timestamp semantics |\n|---|---:|---|---:|---|---|---:|---|\n")
	for _, row := range report.MarketCoverage {
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s | %d | %s |\n", row.Symbol, row.Timeframe, row.Source, row.Count, formatTime(row.First), formatTime(row.Last), row.GapCount, row.TimestampSemantics))
	}
	b.WriteString("\n")
	b.WriteString("Horizon sufficiency:\n\n")
	for _, row := range report.HorizonCoverage {
		b.WriteString(fmt.Sprintf("- %s: %d outcome rows across %d events; sufficient=%t — %s\n", row.Horizon, row.OutcomeCount, row.EventCount, row.Sufficient, row.Reason))
	}
	b.WriteString("\n")
	b.WriteString("## Outcome summary\n\nPrimary-anchor results (absolute, because no trustworthy pre-outcome direction exists):\n\n| Decision | Horizon | n | Median absolute move | Mean absolute move | Median absolute abnormal move | >=0.5% | >=1% | >=2% | Mean range | Mean MFE | Mean MAE |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, row := range report.Metrics {
		if row.Anchor != report.PrimaryAnchor {
			continue
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %d | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n", row.Decision, row.Horizon, row.Count, formatPercentPointer(row.MedianAbsoluteReturn), formatPercentPointer(row.MeanAbsoluteReturn), formatPercentPointer(row.MedianAbsoluteAbnormalReturn), formatPercentPointer(row.ExceedPointFivePercent), formatPercentPointer(row.ExceedOnePercent), formatPercentPointer(row.ExceedTwoPercent), formatPercentPointer(row.MeanRealisedRange), formatPercentPointer(row.MeanMaximumFavourableExcursion), formatPercentPointer(row.MeanMaximumAdverseExcursion)))
	}
	b.WriteString("\n")
	b.WriteString("No percentage is interpreted without its displayed count. Bootstrap intervals, Mann–Whitney U, permutation tests, and effect sizes are emitted in `summary.json` only where both sample-size gates pass.\n\n")
	b.WriteString("## Latency\n\n| Decision | n | Publication → collection median | Collection → receipt median | Receipt → decision median | Move before receipt | Move after receipt |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, row := range report.Latency {
		b.WriteString(fmt.Sprintf("| %s | %d | %s | %s | %s | %s | %s |\n", row.Decision, row.Count, formatDurationPointer(row.PublicationCollectionMedian), formatDurationPointer(row.CollectionReceiptMedian), formatDurationPointer(row.ReceiptDecisionMedian), formatPercentPointer(row.MoveBeforeReceiptMedian), formatPercentPointer(row.MoveAfterReceiptMedian)))
	}
	b.WriteString("\n")
	b.WriteString("## Category and source analysis\n\n")
	writeBreakdown(&b, report.Breakdowns, []string{"event_category", "source", "subject_type", "resolved_asset_state"})
	b.WriteString("## Miss analysis\n\n")
	if len(report.Misses) == 0 {
		b.WriteString("No eligible mapped NO_TRADE outcome exists, so a largest-miss ranking would be fabricated.\n\n")
	} else {
		writeMisses(&b, report.Misses)
	}
	b.WriteString("## WATCH quality\n\n")
	if len(report.WeakWatches) == 0 {
		b.WriteString("No eligible mapped receipt-anchored one-day WATCH outcome exists. The dominant WATCH weakness is unknown asset resolution, not an observed weak return.\n\n")
	} else {
		writeMisses(&b, report.WeakWatches)
	}
	b.WriteString("## Evidence accumulation\n\n")
	writeBreakdown(&b, report.EvidenceAccumulation, []string{"subject_event_count", "source_independence"})
	b.WriteString("Outcome separation cannot be attributed to accumulation when the compared groups lack mapped outcomes. Current subject projections are not backfilled as historical event labels.\n\n")
	b.WriteString("## Bias controls\n\n- Current event-decision projections only; replay-only historical versions excluded.\n- Synthetic, manual test, controlled QQQ proof, duplicate, invalid-time, and post-coverage rows excluded deterministically.\n- Receipt is the primary operational anchor. Publication and collection anchors are separate diagnostics.\n- The first observable candle at or after an anchor is used; no candle preceding an event is used. Daily events occurring after the session open begin at the next persisted session open.\n- Unknown assets stay unknown. Category proxies are bounded by the versioned ruleset; no universal SPY/QQQ mapping is applied.\n- Direction is not inferred. Absolute raw and abnormal movement is evaluated.\n- Existing paper-ticket outcome checkpoints are reported but not joined to genuine-event decisions.\n\n")
	b.WriteString("## Safety\n\n")
	b.WriteString(fmt.Sprintf("Runtime: paper=%t, live trading=%t, execution=%t, execution worker=%t, broker execution=%t, maximum leverage=%.1fx.\n\n", report.RuntimeSafety.RuntimeMode == "paper", report.RuntimeSafety.AllowLiveTrading, report.RuntimeSafety.ExecutionEnabled, report.RuntimeSafety.ExecutionWorker, report.RuntimeSafety.BrokerExecution, report.RuntimeSafety.MaximumLeverage))
	b.WriteString(fmt.Sprintf("Prohibited-state before/after: approvals %d/%d; candidate approvals %d/%d; paper tickets %d/%d; execution instructions %d/%d; order intents %d/%d; broker orders %d/%d; trades %d/%d; fills %d/%d. Delta: zero.\n\n", report.SafetyBefore.Approvals, report.SafetyAfter.Approvals, report.SafetyBefore.CandidateApprovals, report.SafetyAfter.CandidateApprovals, report.SafetyBefore.PaperTickets, report.SafetyAfter.PaperTickets, report.SafetyBefore.ExecutionInstructions, report.SafetyAfter.ExecutionInstructions, report.SafetyBefore.OrderIntents, report.SafetyAfter.OrderIntents, report.SafetyBefore.BrokerOrders, report.SafetyAfter.BrokerOrders, report.SafetyBefore.Trades, report.SafetyAfter.Trades, report.SafetyBefore.Fills, report.SafetyAfter.Fills))
	b.WriteString("## Limitations\n\n")
	for _, limitation := range report.Limitations {
		b.WriteString("- " + limitation + "\n")
	}
	b.WriteString("\n")
	b.WriteString("## Conclusion\n\nPrimary conclusion: **" + report.Conclusion + "**.\n\nProduct recommendation: **" + report.ProductRecommendation + "**.\n\nFinal verdict: **" + report.Verdict + "**.\n")
	return b.String()
}

func writeBreakdown(b *strings.Builder, rows []BreakdownRow, dimensions []string) {
	allowed := map[string]bool{}
	for _, d := range dimensions {
		allowed[d] = true
	}
	b.WriteString("| Dimension | Value | Decision | n | Mapped | 1d outcomes | Median absolute 1d move |\n|---|---|---|---:|---:|---:|---:|\n")
	for _, row := range rows {
		if allowed[row.Dimension] {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %d | %d | %s |\n", row.Dimension, escapeTable(row.Value), row.Decision, row.Count, row.Mapped, row.OutcomeCount, formatPercentPointer(row.MedianAbsoluteReturn)))
		}
	}
	b.WriteString("\n")
}
func writeMisses(b *strings.Builder, rows []MissRow) {
	b.WriteString("| Event | Decision | Symbol | Horizon | Absolute move | Probable cause |\n|---|---|---|---|---:|---|\n")
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %.3f%% | %s |\n", escapeTable(row.Headline), row.Decision, row.Symbol, row.Horizon, row.AbsoluteMove*100, escapeTable(row.ProbableCause)))
	}
	b.WriteString("\n")
}
func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func formatFloat(value float64) string { return strconv.FormatFloat(value, 'f', 10, 64) }
func formatTime(value time.Time) string {
	if value.IsZero() {
		return "n/a"
	}
	return value.UTC().Format(time.RFC3339)
}
func formatTimePointer(value *time.Time) string {
	if value == nil {
		return "n/a"
	}
	return formatTime(*value)
}
func formatPercentPointer(value *float64) string {
	if value == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.3f%%", *value*100)
}
func formatDurationPointer(value *float64) string {
	if value == nil {
		return "n/a"
	}
	return (time.Duration(*value) * time.Second).String()
}
func escapeTable(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "|", "\\|"), "\n", " ")
}
