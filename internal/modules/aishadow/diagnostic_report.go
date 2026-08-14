package aishadow

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
)

type DiagnosticRate struct {
	Numerator   int      `json:"numerator"`
	Denominator int      `json:"denominator"`
	Value       *float64 `json:"value"`
}

type DiagnosticContractMetrics struct {
	TotalEvents              int `json:"total_events"`
	ValidFirstPassOutputs    int `json:"valid_first_pass_outputs"`
	CorrectiveRetries        int `json:"corrective_retries"`
	FinalValidOutputs        int `json:"final_valid_outputs"`
	InvalidOutputs           int `json:"invalid_outputs"`
	SchemaFailures           int `json:"schema_failures"`
	SemanticFailures         int `json:"semantic_failures"`
	TickerEmissionViolations int `json:"ticker_emission_violations"`
}

type DiagnosticIssuerMetrics struct {
	TruePositives               int            `json:"true_positives"`
	FalsePositives              int            `json:"false_positives"`
	TrueNegatives               int            `json:"true_negatives"`
	FalseNegatives              int            `json:"false_negatives"`
	Precision                   DiagnosticRate `json:"precision"`
	Recall                      DiagnosticRate `json:"recall"`
	ClearSingleIssuerRecall     DiagnosticRate `json:"clear_single_issuer_recall"`
	FalseDirectRate             DiagnosticRate `json:"false_direct_rate"`
	UnresolvedRate              DiagnosticRate `json:"unresolved_rate"`
	AmbiguousHandledCorrectly   int            `json:"ambiguous_cases_handled_correctly"`
	MultiIssuerHandledCorrectly int            `json:"multi_issuer_cases_handled_correctly"`
	UnsupportedHandledCorrectly int            `json:"unsupported_unknown_cases_handled_correctly"`
}

type DiagnosticResolutionMetrics struct {
	CorrectIssuerResolved       int `json:"correctly_recognized_issuer_resolved_successfully"`
	CorrectIssuerUnresolved     int `json:"correctly_recognized_issuer_remained_unresolved"`
	AliasMatches                int `json:"alias_matches"`
	CanonicalMatches            int `json:"canonical_matches"`
	AmbiguityRejections         int `json:"ambiguity_rejections"`
	UnsupportedIssuerRejections int `json:"unsupported_issuer_rejections"`
	ExpectedTickerAgreements    int `json:"resolved_ticker_agreements"`
	// IncorrectDeterministicResolutions is the v1 end-to-end resolved-ticker
	// mismatch metric. The v2 attribution fields below identify whether a
	// mismatch followed correct model semantics or a model semantic failure.
	IncorrectDeterministicResolutions  int `json:"incorrect_deterministic_ticker_resolutions"`
	ResolverFailuresAfterSemanticMatch int `json:"resolver_failures_after_semantically_correct_classification"`
	ModelInducedResolutionMismatches   int `json:"model_induced_resolution_mismatches"`
}

type DiagnosticCategoryMetrics struct {
	Total   int `json:"total"`
	Valid   int `json:"valid"`
	Correct int `json:"correct"`
}

type DiagnosticCaseEvaluation struct {
	CaseID                  string                     `json:"case_id"`
	Category                string                     `json:"category"`
	ValidationStatus        string                     `json:"validation_status"`
	FirstPassValid          bool                       `json:"first_pass_valid"`
	RetryCount              int                        `json:"retry_count"`
	ModelClassification     *StructuredResult          `json:"model_classification,omitempty"`
	CausalConsistencyGuard  *CausalConsistencyDecision `json:"causal_consistency_guard,omitempty"`
	DeterministicResolution *PolicyResolution          `json:"deterministic_resolution,omitempty"`
	AdjudicatedLabel        DiagnosticLabel            `json:"adjudicated_label"`
	IssuerCorrect           bool                       `json:"issuer_correct"`
	SemanticCorrect         bool                       `json:"semantic_correct"`
	ResolutionCorrect       bool                       `json:"resolution_correct"`
	FailureClass            string                     `json:"failure_class,omitempty"`
	TickerEmission          []string                   `json:"ticker_emission,omitempty"`
}

type DiagnosticRepetitionReport struct {
	Repetition int                                  `json:"repetition"`
	Contract   DiagnosticContractMetrics            `json:"contract_reliability"`
	Issuer     DiagnosticIssuerMetrics              `json:"issuer_recognition"`
	Resolution DiagnosticResolutionMetrics          `json:"deterministic_resolution"`
	Categories map[string]DiagnosticCategoryMetrics `json:"categories"`
	Cases      []DiagnosticCaseEvaluation           `json:"cases"`
}

type DiagnosticVariation struct {
	CaseID         string                      `json:"case_id"`
	Classification []string                    `json:"variation_classification"`
	Repetitions    []DiagnosticVariationOutput `json:"repetitions"`
}

type DiagnosticVariationOutput struct {
	Repetition              int               `json:"repetition"`
	ValidationStatus        string            `json:"validation_status"`
	ModelClassification     *StructuredResult `json:"model_classification,omitempty"`
	DeterministicResolution *PolicyResolution `json:"deterministic_resolution,omitempty"`
	Correct                 bool              `json:"correct"`
}

type DiagnosticRepeatability struct {
	EventsIdenticalAllRuns          int                   `json:"events_identical_across_all_three_runs"`
	EventsDifferingAcrossRuns       int                   `json:"events_differing_across_runs"`
	MappingStatusStable             int                   `json:"mapping_status_stable"`
	IssuerNameStable                int                   `json:"issuer_name_stable"`
	ConfidenceStable                int                   `json:"confidence_stable"`
	ExposureStable                  int                   `json:"exposure_stable"`
	DeterministicTickerStable       int                   `json:"deterministic_ticker_stable"`
	SemanticClassificationStable    int                   `json:"semantic_classification_stable"`
	SemanticClassificationStability DiagnosticRate        `json:"semantic_classification_stability"`
	CorrectnessChangingEvents       int                   `json:"correctness_changing_events"`
	Variations                      []DiagnosticVariation `json:"variations"`
}

type DiagnosticRunReport struct {
	Version             string                       `json:"version"`
	RunID               string                       `json:"run_id"`
	CompletedAt         time.Time                    `json:"completed_at"`
	ManifestFingerprint string                       `json:"manifest_fingerprint"`
	PromptVersion       string                       `json:"prompt_version"`
	OutputContract      string                       `json:"output_contract"`
	PolicyVersion       string                       `json:"policy_version"`
	ModelIdentity       DiagnosticModelIdentity      `json:"model_identity"`
	ModelConfiguration  DiagnosticModelConfiguration `json:"model_configuration"`
	Repetitions         []DiagnosticRepetitionReport `json:"repetitions"`
	Repeatability       DiagnosticRepeatability      `json:"repeatability"`
	HostedExperiment    *HostedExperimentSnapshot    `json:"hosted_experiment,omitempty"`
}

func EvaluateDiagnosticRepetition(repetition int, manifest DiagnosticManifest, runs []DiagnosticCaseRun, resolver assetresolution.Resolver) DiagnosticRepetitionReport {
	report := DiagnosticRepetitionReport{
		Repetition: repetition,
		Contract:   DiagnosticContractMetrics{TotalEvents: len(manifest.Events)},
		Categories: map[string]DiagnosticCategoryMetrics{},
	}
	byID := map[string]DiagnosticCaseRun{}
	for _, run := range runs {
		byID[run.CaseID] = run
	}
	falseDirectDenominator := 0
	clearPositiveDenominator := 0
	unresolvedPredictions := 0
	for _, event := range manifest.Events {
		run := byID[event.ID]
		evaluation := evaluateDiagnosticCase(event, run, resolver)
		report.Cases = append(report.Cases, evaluation)
		category := report.Categories[event.Category]
		category.Total++
		if evaluation.ValidationStatus == "accepted" {
			category.Valid++
		}
		if evaluation.SemanticCorrect {
			category.Correct++
		}
		report.Categories[event.Category] = category

		if evaluation.FirstPassValid {
			report.Contract.ValidFirstPassOutputs++
		}
		report.Contract.CorrectiveRetries += evaluation.RetryCount
		if evaluation.ValidationStatus == "accepted" {
			report.Contract.FinalValidOutputs++
		} else {
			report.Contract.InvalidOutputs++
			if hasSchemaFailure(run.Result.ValidationErrors) {
				report.Contract.SchemaFailures++
			}
			if hasSemanticFailure(run.Result.ValidationErrors) {
				report.Contract.SemanticFailures++
			}
		}
		if len(evaluation.TickerEmission) > 0 {
			report.Contract.TickerEmissionViolations++
		}

		expectedDirect := event.Label.MappingStatus == "DIRECT"
		predictedDirect := evaluation.ModelClassification != nil && evaluation.ModelClassification.MappingStatus == "DIRECT"
		switch {
		case evaluation.ModelClassification == nil && expectedDirect:
			report.Issuer.FalseNegatives++
		case evaluation.ModelClassification == nil:
			// Invalid outputs are contract failures, not true negatives.
		case expectedDirect && predictedDirect && evaluation.IssuerCorrect:
			report.Issuer.TruePositives++
		case expectedDirect && predictedDirect && !evaluation.IssuerCorrect:
			report.Issuer.FalsePositives++
			report.Issuer.FalseNegatives++
		case expectedDirect:
			report.Issuer.FalseNegatives++
		case predictedDirect:
			report.Issuer.FalsePositives++
		default:
			report.Issuer.TrueNegatives++
		}
		if event.Category == "clear_single_issuer_positive" {
			clearPositiveDenominator++
		}
		if isFalseDirectControl(event.Category) {
			falseDirectDenominator++
		}
		if evaluation.ModelClassification != nil && evaluation.ModelClassification.MappingStatus == "UNRESOLVED" {
			unresolvedPredictions++
		}
		if event.Category == "ambiguous_company_reference" && evaluation.SemanticCorrect {
			report.Issuer.AmbiguousHandledCorrectly++
		}
		if event.Category == "multi_issuer_event" && evaluation.SemanticCorrect {
			report.Issuer.MultiIssuerHandledCorrectly++
		}
		if event.Category == "unsupported_unknown_issuer" && evaluation.SemanticCorrect && evaluation.ResolutionCorrect {
			report.Issuer.UnsupportedHandledCorrectly++
		}
		accumulateResolution(&report.Resolution, event, evaluation)
	}
	report.Issuer.Precision = newDiagnosticRate(report.Issuer.TruePositives, report.Issuer.TruePositives+report.Issuer.FalsePositives)
	report.Issuer.Recall = newDiagnosticRate(report.Issuer.TruePositives, report.Issuer.TruePositives+report.Issuer.FalseNegatives)
	clearCorrect := report.Categories["clear_single_issuer_positive"].Correct
	report.Issuer.ClearSingleIssuerRecall = newDiagnosticRate(clearCorrect, clearPositiveDenominator)
	falseDirect := 0
	for _, evaluation := range report.Cases {
		if isFalseDirectControl(evaluation.Category) && evaluation.ModelClassification != nil && evaluation.ModelClassification.MappingStatus == "DIRECT" {
			falseDirect++
		}
	}
	report.Issuer.FalseDirectRate = newDiagnosticRate(falseDirect, falseDirectDenominator)
	report.Issuer.UnresolvedRate = newDiagnosticRate(unresolvedPredictions, report.Contract.FinalValidOutputs)
	return report
}

func evaluateDiagnosticCase(event DiagnosticEvent, run DiagnosticCaseRun, resolver assetresolution.Resolver) DiagnosticCaseEvaluation {
	evaluation := DiagnosticCaseEvaluation{
		CaseID: event.ID, Category: event.Category, ValidationStatus: run.Result.ValidationStatus,
		RetryCount: run.Result.RetryCount, ModelClassification: run.Result.Parsed,
		CausalConsistencyGuard: run.Result.CausalGuard, DeterministicResolution: run.Result.Resolution, AdjudicatedLabel: event.Label,
	}
	if len(run.Attempts) > 0 {
		evaluation.FirstPassValid = run.Attempts[0].ValidationStatus == "accepted"
	}
	evaluation.TickerEmission = detectTickerEmission(run.Traces, run.Result.Parsed, resolver)
	if run.Result.ValidationStatus != "accepted" || run.Result.Parsed == nil || run.Result.Resolution == nil {
		if hasSchemaFailure(run.Result.ValidationErrors) {
			evaluation.FailureClass = "MODEL_SCHEMA_FAILURE"
		} else {
			evaluation.FailureClass = "MODEL_SEMANTIC_FAILURE"
		}
		return evaluation
	}
	predicted := run.Result.Parsed
	evaluation.IssuerCorrect = issuerMatchesLabel(event, predicted, resolver)
	evaluation.SemanticCorrect = predicted.MappingStatus == event.Label.MappingStatus
	if predicted.MappingStatus == "DIRECT" {
		evaluation.SemanticCorrect = evaluation.SemanticCorrect && evaluation.IssuerCorrect
	} else if predicted.MappingStatus == "PROXY" {
		evaluation.SemanticCorrect = evaluation.SemanticCorrect && predicted.ProxyExposure == event.Label.ProxyExposure
	}
	evaluation.ResolutionCorrect = resolutionMatchesLabel(event, run.Result.Resolution, resolver)
	if !evaluation.SemanticCorrect {
		evaluation.FailureClass = diagnosticFailureClass(event, predicted, evaluation.IssuerCorrect)
	} else if !evaluation.ResolutionCorrect {
		evaluation.FailureClass = diagnosticResolutionFailureClass(event, run.Result.Resolution)
	}
	return evaluation
}

func issuerMatchesLabel(event DiagnosticEvent, predicted *StructuredResult, resolver assetresolution.Resolver) bool {
	if event.Label.MappingStatus != "DIRECT" || predicted == nil || predicted.MappingStatus != "DIRECT" {
		return false
	}
	if assetresolution.CanonicalizeIssuerName(predicted.DirectIssuer) == assetresolution.CanonicalizeIssuerName(event.Label.DirectIssuer) {
		return true
	}
	for _, entity := range event.Input.Entities {
		if assetresolution.CanonicalizeIssuerName(predicted.DirectIssuer) == assetresolution.CanonicalizeIssuerName(entity) {
			return true
		}
	}
	expected := resolver.ResolveIssuer(assetresolution.IssuerInput{IssuerName: event.Label.DirectIssuer, PublicationAt: event.Input.PublicationTimestamp, ReceiptAt: event.Input.ReceiptTimestamp})
	actual := resolver.ResolveIssuer(assetresolution.IssuerInput{IssuerName: predicted.DirectIssuer, PublicationAt: event.Input.PublicationTimestamp, ReceiptAt: event.Input.ReceiptTimestamp})
	return expected.Status == assetresolution.StatusResolved && actual.Status == assetresolution.StatusResolved && expected.Symbol == actual.Symbol && expected.CanonicalEntity == actual.CanonicalEntity
}

func resolutionMatchesLabel(event DiagnosticEvent, actual *PolicyResolution, resolver assetresolution.Resolver) bool {
	if actual == nil || actual.Status != event.Label.ExpectedResolutionStatus {
		return false
	}
	if event.Label.ExpectedResolutionStatus != assetresolution.StatusResolved {
		return actual.ResolvedTicker == ""
	}
	if event.Label.MappingStatus == "DIRECT" {
		expected := resolver.ResolveIssuer(assetresolution.IssuerInput{IssuerName: event.Label.DirectIssuer, PublicationAt: event.Input.PublicationTimestamp, ReceiptAt: event.Input.ReceiptTimestamp})
		return expected.Status == assetresolution.StatusResolved && expected.Symbol == actual.ResolvedTicker
	}
	expected, ok := resolver.ResolveProxyExposure(event.Label.ProxyExposure)
	return ok && expected.Symbol == actual.ResolvedTicker
}

func accumulateResolution(metrics *DiagnosticResolutionMetrics, event DiagnosticEvent, evaluation DiagnosticCaseEvaluation) {
	if evaluation.ModelClassification == nil || evaluation.DeterministicResolution == nil {
		return
	}
	resolution := evaluation.DeterministicResolution
	if !evaluation.ResolutionCorrect {
		if evaluation.SemanticCorrect {
			metrics.ResolverFailuresAfterSemanticMatch++
		} else {
			metrics.ModelInducedResolutionMismatches++
		}
	}
	if evaluation.IssuerCorrect {
		if resolution.Status == assetresolution.StatusResolved {
			metrics.CorrectIssuerResolved++
		} else if resolution.Status == assetresolution.StatusUnresolved {
			metrics.CorrectIssuerUnresolved++
		}
	}
	if resolution.MatchedAlias != "" {
		if assetresolution.CanonicalizeIssuerName(resolution.RawDirectIssuer) == assetresolution.CanonicalizeIssuerName(resolution.CanonicalIssuer) {
			metrics.CanonicalMatches++
		} else {
			metrics.AliasMatches++
		}
	}
	if resolution.Status == assetresolution.StatusAmbiguous {
		metrics.AmbiguityRejections++
	}
	if event.Label.MappingStatus == "DIRECT" && event.Label.ExpectedResolutionStatus == assetresolution.StatusUnresolved && resolution.Status == assetresolution.StatusUnresolved {
		metrics.UnsupportedIssuerRejections++
	}
	if event.Label.ExpectedResolutionStatus == assetresolution.StatusResolved && evaluation.ResolutionCorrect {
		metrics.ExpectedTickerAgreements++
	}
	if event.Label.ExpectedResolutionStatus == assetresolution.StatusResolved && !evaluation.ResolutionCorrect && resolution.Status == assetresolution.StatusResolved {
		metrics.IncorrectDeterministicResolutions++
	}
}

func BuildDiagnosticRunReport(runID string, prepared PreparedDiagnostic, identity DiagnosticModelIdentity, repetitions []DiagnosticRepetitionReport, allRuns [][]DiagnosticCaseRun) DiagnosticRunReport {
	return DiagnosticRunReport{
		Version: DiagnosticReportVersion, RunID: runID, CompletedAt: time.Now().UTC(),
		ManifestFingerprint: prepared.Plan.ManifestFingerprint, PromptVersion: prepared.Plan.PromptVersion,
		OutputContract: prepared.Plan.OutputContract, PolicyVersion: prepared.Plan.PolicyVersion,
		ModelIdentity: identity, ModelConfiguration: prepared.Plan.ModelConfiguration,
		Repetitions: repetitions, Repeatability: evaluateRepeatability(prepared.Manifest, repetitions, allRuns),
	}
}

func evaluateRepeatability(manifest DiagnosticManifest, reports []DiagnosticRepetitionReport, allRuns [][]DiagnosticCaseRun) DiagnosticRepeatability {
	result := DiagnosticRepeatability{}
	if len(allRuns) != diagnosticRepetitionCount || len(reports) != diagnosticRepetitionCount {
		return result
	}
	for index, event := range manifest.Events {
		runs := []DiagnosticCaseRun{allRuns[0][index], allRuns[1][index], allRuns[2][index]}
		outputs := []*StructuredResult{runs[0].Result.Parsed, runs[1].Result.Parsed, runs[2].Result.Parsed}
		resolutions := []*PolicyResolution{runs[0].Result.Resolution, runs[1].Result.Resolution, runs[2].Result.Resolution}
		exact := reflect.DeepEqual(outputs[0], outputs[1]) && reflect.DeepEqual(outputs[1], outputs[2]) &&
			reflect.DeepEqual(resolutions[0], resolutions[1]) && reflect.DeepEqual(resolutions[1], resolutions[2]) &&
			runs[0].Result.ValidationStatus == runs[1].Result.ValidationStatus && runs[1].Result.ValidationStatus == runs[2].Result.ValidationStatus
		mappingStable := equalStrings(outputField(outputs, func(v *StructuredResult) string { return v.MappingStatus }))
		issuerStable := equalStrings(outputField(outputs, func(v *StructuredResult) string { return assetresolution.CanonicalizeIssuerName(v.DirectIssuer) }))
		confidenceStable := equalStrings(outputField(outputs, func(v *StructuredResult) string { return v.MappingConfidence }))
		exposureStable := equalStrings(outputField(outputs, func(v *StructuredResult) string { return v.ProxyExposure }))
		tickerStable := equalStrings(resolutionField(resolutions, func(v *PolicyResolution) string { return v.Status + "|" + v.ResolvedTicker }))
		semanticStable := mappingStable && issuerStable && exposureStable
		if exact {
			result.EventsIdenticalAllRuns++
		} else {
			result.EventsDifferingAcrossRuns++
		}
		if mappingStable {
			result.MappingStatusStable++
		}
		if issuerStable {
			result.IssuerNameStable++
		}
		if confidenceStable {
			result.ConfidenceStable++
		}
		if exposureStable {
			result.ExposureStable++
		}
		if tickerStable {
			result.DeterministicTickerStable++
		}
		if semanticStable {
			result.SemanticClassificationStable++
		}
		if !exact {
			variation := DiagnosticVariation{CaseID: event.ID}
			correctValues := []bool{}
			for repetition := 0; repetition < diagnosticRepetitionCount; repetition++ {
				caseEvaluation := reports[repetition].Cases[index]
				correctValues = append(correctValues, caseEvaluation.SemanticCorrect && caseEvaluation.ResolutionCorrect)
				variation.Repetitions = append(variation.Repetitions, DiagnosticVariationOutput{Repetition: repetition + 1, ValidationStatus: runs[repetition].Result.ValidationStatus, ModelClassification: outputs[repetition], DeterministicResolution: resolutions[repetition], Correct: correctValues[repetition]})
			}
			correctnessChanging := !(correctValues[0] == correctValues[1] && correctValues[1] == correctValues[2])
			if correctnessChanging {
				variation.Classification = append(variation.Classification, "correctness-changing variation")
				result.CorrectnessChangingEvents++
			}
			if !mappingStable {
				variation.Classification = append(variation.Classification, "DIRECT/PROXY/UNRESOLVED variation")
			}
			if !issuerStable {
				variation.Classification = append(variation.Classification, "issuer variation")
			}
			if !exposureStable {
				variation.Classification = append(variation.Classification, "proxy exposure variation")
			}
			if !tickerStable {
				variation.Classification = append(variation.Classification, "deterministic resolution variation")
			}
			if !confidenceStable {
				variation.Classification = append(variation.Classification, "confidence-only variation")
			}
			if len(variation.Classification) == 0 {
				variation.Classification = append(variation.Classification, "harmless wording variation")
			}
			result.Variations = append(result.Variations, variation)
		}
	}
	result.SemanticClassificationStability = newDiagnosticRate(result.SemanticClassificationStable, len(manifest.Events))
	return result
}

func DiagnosticReportMarkdown(report DiagnosticRunReport) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# V4 Issuer Diagnostic\n\nRun `%s` used manifest `%s`, model `%s` (`%s`), prompt `%s`, contract `%s`, and policy `%s`.\n\n", report.RunID, report.ManifestFingerprint, report.ModelIdentity.Name, report.ModelIdentity.Digest, report.PromptVersion, report.OutputContract, report.PolicyVersion)
	if hosted := report.HostedExperiment; hosted != nil {
		fmt.Fprintf(&builder, "## Hosted experiment\n\n- Cell: %s\n- Provider / requested model: %s / %s\n- Returned model identities: %s\n- System fingerprints: %s\n- Reasoning: %s\n- Structured output mode: %s\n- Requests / corrective retries: %d / %d\n- Input / uncached / cached / cache-write tokens: %d / %d / %d / %d\n- Output / reasoning tokens: %d / %d\n- Uncached-input / cached-input / cache-write / output cost USD: %s / %s / %s / %s\n- Calculable / ambiguous / remaining / ceiling USD: %s / %s / %s / %s\n- Stop reason: %s\n\n", hosted.ExperimentID, hosted.Provider, hosted.RequestedModel, strings.Join(hosted.ReturnedModels, ", "), strings.Join(hosted.SystemFingerprints, ", "), hosted.ReasoningEffort, hosted.StructuredOutputMode, hosted.RequestCount, hosted.RetryCount, hosted.Usage.InputTokens, hosted.Usage.CacheMissTokens, hosted.Usage.CachedTokens, hosted.Usage.CacheWriteTokens, hosted.Usage.OutputTokens, hosted.Usage.ReasoningTokens, hosted.CostByCategory.UncachedInputUSD, hosted.CostByCategory.CachedInputUSD, hosted.CostByCategory.CacheWriteUSD, hosted.CostByCategory.OutputUSD, hosted.ActualCalculableCostUSD, hosted.AmbiguousLiabilityUSD, hosted.RemainingBudgetUSD, hosted.BudgetCeilingUSD, hosted.StopReason)
	}
	for _, repetition := range report.Repetitions {
		fmt.Fprintf(&builder, "## Repetition %d\n\n- First-pass valid: %d/%d\n- Final valid: %d/%d\n- Corrective retries: %d\n- Issuer precision: %s\n- Issuer recall: %s\n- False-DIRECT rate: %s\n- Resolver failures after semantically correct classification: %d\n- Model-induced resolution mismatches: %d\n\n", repetition.Repetition, repetition.Contract.ValidFirstPassOutputs, repetition.Contract.TotalEvents, repetition.Contract.FinalValidOutputs, repetition.Contract.TotalEvents, repetition.Contract.CorrectiveRetries, formatDiagnosticRate(repetition.Issuer.Precision), formatDiagnosticRate(repetition.Issuer.Recall), formatDiagnosticRate(repetition.Issuer.FalseDirectRate), repetition.Resolution.ResolverFailuresAfterSemanticMatch, repetition.Resolution.ModelInducedResolutionMismatches)
	}
	fmt.Fprintf(&builder, "## Repeatability\n\n- Identical across all three: %d/%d\n- Semantic stability: %s\n- Correctness-changing events: %d\n", report.Repeatability.EventsIdenticalAllRuns, report.Repeatability.SemanticClassificationStability.Denominator, formatDiagnosticRate(report.Repeatability.SemanticClassificationStability), report.Repeatability.CorrectnessChangingEvents)
	return builder.String()
}

func newDiagnosticRate(numerator, denominator int) DiagnosticRate {
	rate := DiagnosticRate{Numerator: numerator, Denominator: denominator}
	if denominator > 0 {
		value := float64(numerator) / float64(denominator)
		rate.Value = &value
	}
	return rate
}

func formatDiagnosticRate(rate DiagnosticRate) string {
	if rate.Value == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.2f%% (%d/%d)", *rate.Value*100, rate.Numerator, rate.Denominator)
}

func diagnosticFailureClass(event DiagnosticEvent, predicted *StructuredResult, issuerCorrect bool) string {
	if predicted.MappingStatus == "DIRECT" && event.Label.MappingStatus != "DIRECT" {
		return "MODEL_FALSE_ISSUER"
	}
	if event.Category == "ambiguous_company_reference" {
		return "MODEL_AMBIGUITY_FAILURE"
	}
	if event.Category == "multi_issuer_event" {
		return "MODEL_MULTI_ISSUER_FAILURE"
	}
	if event.Label.MappingStatus == "DIRECT" && (predicted.MappingStatus != "DIRECT" || !issuerCorrect) {
		return "MODEL_ISSUER_MISSED"
	}
	return "OTHER"
}

func diagnosticResolutionFailureClass(event DiagnosticEvent, resolution *PolicyResolution) string {
	if resolution == nil {
		return "OTHER"
	}
	if event.Label.ExpectedResolutionStatus == assetresolution.StatusResolved {
		switch resolution.Status {
		case assetresolution.StatusUnresolved:
			return "DETERMINISTIC_ALIAS_MISS"
		case assetresolution.StatusAmbiguous:
			return "DETERMINISTIC_AMBIGUITY_REJECTION"
		case assetresolution.StatusResolved:
			return "DETERMINISTIC_WRONG_RESOLUTION"
		}
	}
	if event.Label.ExpectedResolutionStatus == assetresolution.StatusUnresolved && resolution.Status == assetresolution.StatusUnresolved {
		return "DETERMINISTIC_UNKNOWN_ISSUER"
	}
	return "OTHER"
}

func hasSchemaFailure(errors []string) bool {
	for _, value := range errors {
		if strings.HasPrefix(value, "invalid JSON:") || strings.HasPrefix(value, "missing required field:") || strings.HasPrefix(value, "unknown field:") || strings.HasPrefix(value, "schema decode:") || strings.Contains(value, "trailing data") || strings.Contains(value, "must be a string, not null") {
			return true
		}
	}
	return false
}

func hasSemanticFailure(errors []string) bool {
	for _, value := range errors {
		if !hasSchemaFailure([]string{value}) && value != "provider request failed" {
			return true
		}
	}
	return false
}

func isFalseDirectControl(category string) bool {
	switch category {
	case "clear_exposure_only_negative", "ambiguous_company_reference", "multi_issuer_event", "company_mentioned_not_causal":
		return true
	default:
		return false
	}
}

func detectTickerEmission(traces []ProviderTrace, parsed *StructuredResult, resolver assetresolution.Resolver) []string {
	violations := map[string]bool{}
	for _, trace := range traces {
		var fields map[string]json.RawMessage
		if json.Unmarshal([]byte(trace.Content), &fields) == nil {
			for field := range fields {
				lower := strings.ToLower(field)
				if strings.Contains(lower, "ticker") || strings.Contains(lower, "symbol") {
					violations["prohibited field "+field] = true
				}
			}
			var attempt StructuredResult
			_ = json.Unmarshal(fields["reason"], &attempt.Reason)
			_ = json.Unmarshal(fields["catalyst_type"], &attempt.CatalystType)
			_ = json.Unmarshal(fields["missing_evidence"], &attempt.MissingEvidence)
			detectTickerTokens(attempt, resolver, violations)
		}
	}
	if parsed != nil {
		detectTickerTokens(*parsed, resolver, violations)
	}
	values := make([]string, 0, len(violations))
	for value := range violations {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func detectTickerTokens(output StructuredResult, resolver assetresolution.Resolver, violations map[string]bool) {
	freeText := strings.Join(append([]string{output.Reason, output.CatalystType}, output.MissingEvidence...), " ")
	for _, rule := range resolver.Rules.Aliases {
		symbol := strings.ToUpper(strings.TrimSpace(rule.Symbol))
		if symbol == "" {
			continue
		}
		pattern := regexp.MustCompile(`(^|[^A-Za-z0-9])` + regexp.QuoteMeta(symbol) + `([^A-Za-z0-9]|$)`)
		if pattern.MatchString(freeText) {
			violations["ticker token in free text: "+symbol] = true
		}
	}
}

func outputField(outputs []*StructuredResult, field func(*StructuredResult) string) []string {
	values := make([]string, len(outputs))
	for index, output := range outputs {
		if output == nil {
			values[index] = "<invalid>"
		} else {
			values[index] = field(output)
		}
	}
	return values
}

func resolutionField(outputs []*PolicyResolution, field func(*PolicyResolution) string) []string {
	values := make([]string, len(outputs))
	for index, output := range outputs {
		if output == nil {
			values[index] = "<invalid>"
		} else {
			values[index] = field(output)
		}
	}
	return values
}

func equalStrings(values []string) bool {
	return len(values) == 3 && values[0] == values[1] && values[1] == values[2]
}
