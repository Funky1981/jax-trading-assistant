package aishadow

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	minimumValidOutputRate        = 0.98
	maximumFabricatedTickerRate   = 0.02
	minimumDirectMappingAgreement = 0.85
	minimumAbsoluteSeparation     = 0.0005
	minimumRelativeSeparation     = 0.10
	minimumBaselineImprovement    = 0.0005
)

type ArtifactPaths struct {
	Markdown string `json:"markdown"`
	JSON     string `json:"json"`
	CSV      string `json:"csv"`
	Manifest string `json:"manifest"`
}

type Separation struct {
	HighCount          int      `json:"high_count"`
	LowUncertainCount  int      `json:"low_uncertain_count"`
	HighMedian         *float64 `json:"high_median"`
	LowUncertainMedian *float64 `json:"low_uncertain_median"`
	Difference         *float64 `json:"difference"`
}

type Baseline struct {
	WatchCount    int      `json:"watch_count"`
	NoTradeCount  int      `json:"no_trade_count"`
	WatchMedian   *float64 `json:"watch_median"`
	NoTradeMedian *float64 `json:"no_trade_median"`
	Difference    *float64 `json:"difference"`
}

type ReviewItem struct {
	EventID                 string `json:"event_id"`
	DeterministicMapping    string `json:"deterministic_mapping"`
	DeterministicAsset      string `json:"deterministic_asset,omitempty"`
	ModelMapping            string `json:"model_mapping"`
	ModelDirectIssuer       string `json:"model_direct_issuer,omitempty"`
	ModelProxyExposure      string `json:"model_proxy_exposure"`
	JaxResolutionStatus     string `json:"jax_resolution_status"`
	JaxNormalizedIssuer     string `json:"jax_normalized_issuer,omitempty"`
	JaxCanonicalIssuer      string `json:"jax_canonical_issuer,omitempty"`
	JaxMatchedAlias         string `json:"jax_matched_alias,omitempty"`
	JaxResolvedTicker       string `json:"jax_resolved_ticker,omitempty"`
	ResolutionPolicyVersion string `json:"resolution_policy_version"`
	MatchedRule             string `json:"matched_rule,omitempty"`
}

type Example struct {
	EventID                 string           `json:"event_id"`
	ModelOutput             StructuredResult `json:"model_output"`
	DeterministicResolution PolicyResolution `json:"deterministic_resolution"`
}

type FrozenReference struct {
	Decision string `json:"decision"`
	Mapping  string `json:"mapping"`
	Asset    string `json:"asset,omitempty"`
}

type Evaluation struct {
	EventID                 string            `json:"event_id"`
	ValidationStatus        string            `json:"validation_status"`
	ModelClassification     *StructuredResult `json:"model_issuer_exposure_classification,omitempty"`
	DeterministicResolution *PolicyResolution `json:"jax_deterministic_resolution,omitempty"`
	FrozenReference         FrozenReference   `json:"frozen_reference"`
}

type Report struct {
	RunID                    string           `json:"run_id"`
	ManifestVersion          string           `json:"manifest_version"`
	ManifestFingerprint      string           `json:"manifest_fingerprint"`
	Provider                 string           `json:"provider"`
	Model                    string           `json:"model"`
	PromptVersion            string           `json:"prompt_version"`
	SchemaVersion            string           `json:"schema_version"`
	Seed                     int64            `json:"seed"`
	Temperature              float64          `json:"temperature"`
	EventsSelected           int              `json:"events_selected"`
	EventsAttempted          int              `json:"events_attempted"`
	Accepted                 int              `json:"accepted_outputs"`
	Rejected                 int              `json:"rejected_outputs"`
	RetryCount               int              `json:"retry_count"`
	MedianLatencyMS          float64          `json:"median_latency_ms"`
	P95LatencyMS             float64          `json:"p95_latency_ms"`
	AssetResolutionCoverage  float64          `json:"asset_resolution_coverage"`
	ExactMappingAgreement    float64          `json:"exact_mapping_agreement"`
	DirectMappingAgreement   float64          `json:"direct_mapping_agreement"`
	ProxyMappingAgreement    float64          `json:"proxy_mapping_agreement"`
	UnresolvedAgreement      float64          `json:"unresolved_agreement"`
	FabricatedInvalidTickers int              `json:"fabricated_or_invalid_ticker_count"`
	RelevanceDistribution    map[string]int   `json:"relevance_distribution"`
	ConfidenceDistribution   map[string]int   `json:"confidence_distribution"`
	ConsistencyViolations    int              `json:"consistency_violations"`
	AI1H                     Separation       `json:"ai_relevance_1h"`
	AI1D                     Separation       `json:"ai_relevance_1d"`
	Baseline1H               Baseline         `json:"deterministic_baseline_1h"`
	Baseline1D               Baseline         `json:"deterministic_baseline_1d"`
	Examples                 []Example        `json:"model_output_examples"`
	Evaluations              []Evaluation     `json:"evaluations"`
	Disagreements            []ReviewItem     `json:"disagreement_review_list"`
	SafetyBefore             SafetyCounts     `json:"safety_before"`
	SafetyAfter              SafetyCounts     `json:"safety_after"`
	Verdict                  string           `json:"verdict"`
	Limitations              []string         `json:"limitations"`
	Results                  []EventResult    `json:"-"`
	Events                   []BenchmarkEvent `json:"-"`
}

func BuildReport(run RunRecord, manifest Manifest, events []BenchmarkEvent, results []EventResult, before, after SafetyCounts) Report {
	report := Report{
		RunID: run.ID, ManifestVersion: manifest.Version, ManifestFingerprint: manifest.Fingerprint,
		Provider: run.Provider, Model: run.Model, PromptVersion: run.PromptVersion, SchemaVersion: run.SchemaVersion,
		Seed: run.Seed, Temperature: run.Temperature, EventsSelected: len(events), EventsAttempted: len(results),
		RelevanceDistribution:  map[string]int{"HIGH": 0, "MEDIUM": 0, "LOW": 0, "UNCERTAIN": 0},
		ConfidenceDistribution: map[string]int{"HIGH": 0, "MEDIUM": 0, "LOW": 0},
		SafetyBefore:           before, SafetyAfter: after, Results: results, Events: events,
	}
	byID := map[string]BenchmarkEvent{}
	for _, event := range events {
		byID[event.ID] = event
	}
	latencies := []float64{}
	exact, directOK, proxyOK, unresolvedOK := 0, 0, 0, 0
	directTotal, proxyTotal, unresolvedTotal, covered := 0, 0, 0, 0
	for _, result := range results {
		latencies = append(latencies, float64(result.Duration.Milliseconds()))
		report.RetryCount += result.RetryCount
		event := byID[result.EventID]
		detType, detAsset := deterministicMapping(event)
		report.Evaluations = append(report.Evaluations, Evaluation{
			EventID: result.EventID, ValidationStatus: result.ValidationStatus,
			ModelClassification: result.Parsed, DeterministicResolution: result.Resolution,
			FrozenReference: FrozenReference{Decision: event.Decision, Mapping: detType, Asset: detAsset},
		})
		switch detType {
		case "DIRECT":
			directTotal++
		case "PROXY":
			proxyTotal++
		default:
			unresolvedTotal++
		}
		if result.ValidationStatus != "accepted" || result.Parsed == nil || result.Resolution == nil {
			report.Rejected++
			for _, validationError := range result.ValidationErrors {
				if strings.Contains(validationError, "ticker") {
					report.FabricatedInvalidTickers++
				}
				if strings.Contains(validationError, "mapping") || strings.Contains(validationError, "confidence") || strings.Contains(validationError, "ticker") {
					report.ConsistencyViolations++
				}
			}
			continue
		}
		report.Accepted++
		output := *result.Parsed
		report.RelevanceDistribution[output.MarketRelevance]++
		report.ConfidenceDistribution[output.MappingConfidence]++
		resolvedAsset := result.Resolution.ResolvedTicker
		if resolvedAsset != "" {
			covered++
		}
		agree := output.MappingStatus == detType && resolvedAsset == detAsset
		if agree {
			exact++
		}
		switch detType {
		case "DIRECT":
			if agree {
				directOK++
			}
		case "PROXY":
			if agree {
				proxyOK++
			}
		default:
			if agree {
				unresolvedOK++
			}
		}
		if !agree {
			report.Disagreements = append(report.Disagreements, ReviewItem{
				EventID: event.ID, DeterministicMapping: detType, DeterministicAsset: detAsset,
				ModelMapping: output.MappingStatus, ModelDirectIssuer: output.DirectIssuer,
				ModelProxyExposure: output.ProxyExposure, JaxResolvedTicker: resolvedAsset,
				JaxResolutionStatus: result.Resolution.Status, JaxNormalizedIssuer: result.Resolution.NormalizedIssuer,
				JaxCanonicalIssuer: result.Resolution.CanonicalIssuer, JaxMatchedAlias: result.Resolution.MatchedAlias,
				ResolutionPolicyVersion: result.Resolution.PolicyVersion, MatchedRule: result.Resolution.MatchedRule,
			})
		}
		if len(report.Examples) < 5 {
			report.Examples = append(report.Examples, Example{EventID: event.ID, ModelOutput: output, DeterministicResolution: *result.Resolution})
		}
	}
	report.MedianLatencyMS = percentile(latencies, 0.5)
	report.P95LatencyMS = percentile(latencies, 0.95)
	report.AssetResolutionCoverage = rate(covered, len(events))
	report.ExactMappingAgreement = rate(exact, len(events))
	report.DirectMappingAgreement = rate(directOK, directTotal)
	report.ProxyMappingAgreement = rate(proxyOK, proxyTotal)
	report.UnresolvedAgreement = rate(unresolvedOK, unresolvedTotal)
	report.AI1H = aiSeparation(events, results, true)
	report.AI1D = aiSeparation(events, results, false)
	report.Baseline1H = deterministicBaseline(events, true)
	report.Baseline1D = deterministicBaseline(events, false)
	report.Verdict, report.Limitations = verdict(report)
	return report
}

func verdict(report Report) (string, []string) {
	limitations := []string{}
	if report.EventsSelected < 60 {
		limitations = append(limitations, "bounded smoke sample; not eligible for a final value verdict")
	}
	validRate := rate(report.Accepted, report.EventsAttempted)
	fabricatedRate := rate(report.FabricatedInvalidTickers, report.EventsAttempted)
	strong1H := materiallyStronger(report.AI1H)
	improves := difference(report.AI1H) >= differenceBaseline(report.Baseline1H)+minimumBaselineImprovement || difference(report.AI1D) >= differenceBaseline(report.Baseline1D)+minimumBaselineImprovement
	if report.EventsSelected < 60 {
		return "MIXED", limitations
	}
	if report.SafetyBefore != report.SafetyAfter {
		return "INVALID", append(limitations, "prohibited safety counts changed")
	}
	if validRate >= minimumValidOutputRate && fabricatedRate <= maximumFabricatedTickerRate && report.DirectMappingAgreement >= minimumDirectMappingAgreement && strong1H && improves {
		return "AI VALUE DEMONSTRATED", limitations
	}
	if validRate >= minimumValidOutputRate && fabricatedRate <= maximumFabricatedTickerRate && report.DirectMappingAgreement >= minimumDirectMappingAgreement {
		return "MIXED", append(limitations, "operational validity passed but outcome improvements were inconsistent or too small")
	}
	return "NOT DEMONSTRATED", append(limitations, "useful improvement was absent or mapping/validity errors exceeded predeclared limits")
}

func materiallyStronger(value Separation) bool {
	if value.HighMedian == nil || value.LowUncertainMedian == nil {
		return false
	}
	return *value.HighMedian-*value.LowUncertainMedian >= minimumAbsoluteSeparation && *value.HighMedian >= *value.LowUncertainMedian*(1+minimumRelativeSeparation)
}

func aiSeparation(events []BenchmarkEvent, results []EventResult, oneHour bool) Separation {
	byID := map[string]BenchmarkEvent{}
	for _, event := range events {
		byID[event.ID] = event
	}
	high, low := []float64{}, []float64{}
	for _, result := range results {
		if result.Parsed == nil || result.ValidationStatus != "accepted" {
			continue
		}
		event := byID[result.EventID]
		value := event.Outcome1D
		if oneHour {
			value = event.Outcome1H
		}
		switch result.Parsed.MarketRelevance {
		case "HIGH":
			high = append(high, value)
		case "LOW", "UNCERTAIN":
			low = append(low, value)
		}
	}
	return newSeparation(high, low)
}

func deterministicBaseline(events []BenchmarkEvent, oneHour bool) Baseline {
	watch, noTrade := []float64{}, []float64{}
	for _, event := range events {
		value := event.Outcome1D
		if oneHour {
			value = event.Outcome1H
		}
		if event.Decision == "WATCH" {
			watch = append(watch, value)
		} else if event.Decision == "NO_TRADE" {
			noTrade = append(noTrade, value)
		}
	}
	result := Baseline{WatchCount: len(watch), NoTradeCount: len(noTrade), WatchMedian: medianPointer(watch), NoTradeMedian: medianPointer(noTrade)}
	if result.WatchMedian != nil && result.NoTradeMedian != nil {
		value := *result.WatchMedian - *result.NoTradeMedian
		result.Difference = &value
	}
	return result
}

func newSeparation(high, low []float64) Separation {
	result := Separation{HighCount: len(high), LowUncertainCount: len(low), HighMedian: medianPointer(high), LowUncertainMedian: medianPointer(low)}
	if result.HighMedian != nil && result.LowUncertainMedian != nil {
		v := *result.HighMedian - *result.LowUncertainMedian
		result.Difference = &v
	}
	return result
}
func deterministicMapping(event BenchmarkEvent) (string, string) {
	if !event.Mapping.Mapped {
		return "UNRESOLVED", ""
	}
	if event.Mapping.Direct {
		return "DIRECT", event.Mapping.Symbol
	}
	return "PROXY", event.Mapping.Symbol
}
func rate(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}
func medianPointer(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	v := percentile(values, .5)
	return &v
}
func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64{}, values...)
	sort.Float64s(sorted)
	index := int(math.Ceil(p*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	return sorted[index]
}
func difference(value Separation) float64 {
	if value.Difference == nil {
		return math.Inf(-1)
	}
	return *value.Difference
}
func differenceBaseline(value Baseline) float64 {
	if value.Difference == nil {
		return math.Inf(1)
	}
	return *value.Difference
}

func WriteArtifacts(dir string, report Report, manifest Manifest) (ArtifactPaths, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ArtifactPaths{}, fmt.Errorf("create AI shadow report directory: %w", err)
	}
	paths := ArtifactPaths{Markdown: filepath.Join(dir, "report.md"), JSON: filepath.Join(dir, "report.json"), CSV: filepath.Join(dir, "results.csv"), Manifest: filepath.Join(dir, "manifest.json")}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return ArtifactPaths{}, err
	}
	if err = os.WriteFile(paths.JSON, append(raw, '\n'), 0o644); err != nil {
		return ArtifactPaths{}, err
	}
	if err = WriteManifest(paths.Manifest, manifest); err != nil {
		return ArtifactPaths{}, err
	}
	if err = os.WriteFile(paths.Markdown, []byte(markdown(report)), 0o644); err != nil {
		return ArtifactPaths{}, err
	}
	if err = writeCSV(paths.CSV, report); err != nil {
		return ArtifactPaths{}, err
	}
	return paths, nil
}

func writeCSV(path string, report Report) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	_ = writer.Write([]string{"event_id", "validation_status", "retry_count", "latency_ms", "market_relevance", "model_mapping_status", "model_direct_issuer", "model_proxy_exposure", "model_mapping_confidence", "jax_resolution_status", "jax_raw_direct_issuer", "jax_normalized_issuer", "jax_canonical_issuer", "jax_matched_alias", "jax_resolved_ticker", "jax_mapping_type", "jax_relationship", "resolution_policy_version", "matched_rule", "jax_deterministic_reason", "frozen_reference_decision", "frozen_reference_mapping", "frozen_reference_asset", "absolute_1h_move", "absolute_1d_move"})
	byID := map[string]BenchmarkEvent{}
	for _, e := range report.Events {
		byID[e.ID] = e
	}
	for _, r := range report.Results {
		e := byID[r.EventID]
		relevance, directIssuer, proxyExposure, mapping, confidence := "", "", "", "", ""
		if r.Parsed != nil {
			relevance = r.Parsed.MarketRelevance
			mapping = r.Parsed.MappingStatus
			confidence = r.Parsed.MappingConfidence
			directIssuer = r.Parsed.DirectIssuer
			proxyExposure = r.Parsed.ProxyExposure
		}
		resolvedTicker, policyVersion, matchedRule := "", "", ""
		resolutionStatus, rawIssuer, normalizedIssuer, canonicalIssuer, matchedAlias := "", "", "", "", ""
		mappingType, relationship, deterministicReason := "", "", ""
		if r.Resolution != nil {
			resolvedTicker, policyVersion, matchedRule = r.Resolution.ResolvedTicker, r.Resolution.PolicyVersion, r.Resolution.MatchedRule
			resolutionStatus, rawIssuer, normalizedIssuer = r.Resolution.Status, r.Resolution.RawDirectIssuer, r.Resolution.NormalizedIssuer
			canonicalIssuer, matchedAlias = r.Resolution.CanonicalIssuer, r.Resolution.MatchedAlias
			mappingType, relationship, deterministicReason = r.Resolution.MappingType, r.Resolution.Relationship, r.Resolution.Reason
		}
		dt, da := deterministicMapping(e)
		_ = writer.Write([]string{r.EventID, r.ValidationStatus, strconv.Itoa(r.RetryCount), strconv.FormatInt(r.Duration.Milliseconds(), 10), relevance, mapping, directIssuer, proxyExposure, confidence, resolutionStatus, rawIssuer, normalizedIssuer, canonicalIssuer, matchedAlias, resolvedTicker, mappingType, relationship, policyVersion, matchedRule, deterministicReason, e.Decision, dt, da, strconv.FormatFloat(e.Outcome1H, 'f', 8, 64), strconv.FormatFloat(e.Outcome1D, 'f', 8, 64)})
	}
	return writer.Error()
}

func markdown(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# AI Shadow Benchmark\n\nRun `%s` used manifest `%s` with `%s` / `%s`.\n\n", r.RunID, r.ManifestFingerprint, r.Provider, r.Model)
	fmt.Fprintf(&b, "## Verdict\n\n**%s**\n\n", r.Verdict)
	fmt.Fprintf(&b, "## Operations\n\n| Metric | Value |\n|---|---:|\n| Events selected | %d |\n| Events attempted | %d |\n| Accepted | %d |\n| Rejected | %d |\n| Corrective retries | %d |\n| Median latency (ms) | %.0f |\n| p95 latency (ms) | %.0f |\n\n", r.EventsSelected, r.EventsAttempted, r.Accepted, r.Rejected, r.RetryCount, r.MedianLatencyMS, r.P95LatencyMS)
	fmt.Fprintf(&b, "## Mapping\n\n| Metric | Value |\n|---|---:|\n| Coverage | %.2f%% |\n| Exact agreement | %.2f%% |\n| Direct agreement | %.2f%% |\n| Proxy agreement | %.2f%% |\n| Unresolved agreement | %.2f%% |\n| Fabricated or invalid tickers | %d |\n| Consistency violations | %d |\n\n", 100*r.AssetResolutionCoverage, 100*r.ExactMappingAgreement, 100*r.DirectMappingAgreement, 100*r.ProxyMappingAgreement, 100*r.UnresolvedAgreement, r.FabricatedInvalidTickers, r.ConsistencyViolations)
	fmt.Fprintf(&b, "## Outcome separation\n\n| Classifier | Horizon | High/Watch n | Low/Uncertain/No trade n | First median | Second median | Difference |\n|---|---|---:|---:|---:|---:|---:|\n")
	fmt.Fprintf(&b, "| AI | 1h | %d | %d | %s | %s | %s |\n", r.AI1H.HighCount, r.AI1H.LowUncertainCount, percent(r.AI1H.HighMedian), percent(r.AI1H.LowUncertainMedian), percent(r.AI1H.Difference))
	fmt.Fprintf(&b, "| AI | 1d | %d | %d | %s | %s | %s |\n", r.AI1D.HighCount, r.AI1D.LowUncertainCount, percent(r.AI1D.HighMedian), percent(r.AI1D.LowUncertainMedian), percent(r.AI1D.Difference))
	fmt.Fprintf(&b, "| Deterministic | 1h | %d | %d | %s | %s | %s |\n", r.Baseline1H.WatchCount, r.Baseline1H.NoTradeCount, percent(r.Baseline1H.WatchMedian), percent(r.Baseline1H.NoTradeMedian), percent(r.Baseline1H.Difference))
	fmt.Fprintf(&b, "| Deterministic | 1d | %d | %d | %s | %s | %s |\n\n", r.Baseline1D.WatchCount, r.Baseline1D.NoTradeCount, percent(r.Baseline1D.WatchMedian), percent(r.Baseline1D.NoTradeMedian), percent(r.Baseline1D.Difference))
	fmt.Fprintf(&b, "Likely direction and expected horizon are exploratory metrics, not proof of predictive power.\n\n## Output distributions\n\nRelevance: `%s`\n\nConfidence: `%s`\n\n", compactJSON(r.RelevanceDistribution), compactJSON(r.ConfidenceDistribution))
	b.WriteString("## Model issuer/exposure classifications and Jax deterministic resolutions\n\n")
	for _, example := range r.Examples {
		fmt.Fprintf(&b, "- `%s`: model `%s`; Jax deterministic `%s`\n", example.EventID, compactJSON(example.ModelOutput), compactJSON(example.DeterministicResolution))
	}
	b.WriteString("\n## Final comparison against frozen reference\n\n")
	for _, item := range r.Disagreements {
		fmt.Fprintf(&b, "- `%s`: frozen reference %s `%s`; model classification %s with issuer `%s` / exposure `%s`; Jax deterministic status `%s`, policy `%s`, alias `%s`, rule `%s`, resolved ticker `%s`\n", item.EventID, item.DeterministicMapping, item.DeterministicAsset, item.ModelMapping, item.ModelDirectIssuer, item.ModelProxyExposure, item.JaxResolutionStatus, item.ResolutionPolicyVersion, item.JaxMatchedAlias, item.MatchedRule, item.JaxResolvedTicker)
	}
	if len(r.Disagreements) == 0 {
		b.WriteString("None.\n")
	}
	fmt.Fprintf(&b, "\n## Safety state\n\nBefore: `%s`\n\nAfter: `%s`\n\n", compactJSON(r.SafetyBefore), compactJSON(r.SafetyAfter))
	if len(r.Limitations) > 0 {
		b.WriteString("\n## Limitations\n\n")
		for _, v := range r.Limitations {
			fmt.Fprintf(&b, "- %s\n", v)
		}
	}
	return b.String()
}

func percent(value *float64) string {
	if value == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.4f%%", *value*100)
}

func compactJSON(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

var _ = time.Time{}
