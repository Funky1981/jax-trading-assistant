package aishadow

import (
	"fmt"
	"sort"
	"strings"

	"jax-trading-assistant/internal/modules/assetresolution"
)

var AcceptedC1BRunIDs = []string{
	"7fa0d4b9-b0f9-4f53-8e30-07e251dd14f2",
	"434beae6-165a-4f26-ac17-370e4371f44c",
	"8f8dfdf7-3e2a-4074-a519-b41e7adfddfb",
}

type CausalReplayRate struct {
	Numerator   int     `json:"numerator"`
	Denominator int     `json:"denominator"`
	Value       float64 `json:"value"`
}

type CausalReplayMetrics struct {
	FinalValidity         CausalReplayRate `json:"final_validity"`
	SemanticAccuracy      CausalReplayRate `json:"semantic_accuracy"`
	EndToEndAccuracy      CausalReplayRate `json:"end_to_end_accuracy"`
	DirectRecall          CausalReplayRate `json:"direct_recall"`
	DirectPrecision       CausalReplayRate `json:"direct_precision"`
	ProxyAccuracy         CausalReplayRate `json:"proxy_accuracy"`
	UnresolvedAccuracy    CausalReplayRate `json:"unresolved_accuracy"`
	FalseDirect           int              `json:"false_direct"`
	FalseProxy            int              `json:"false_proxy"`
	SemanticRepeatability CausalReplayRate `json:"semantic_repeatability"`
	ResolverFailures      int              `json:"deterministic_resolver_failures"`
}

type CausalReplayAbstention struct {
	RunID            string       `json:"run_id"`
	CaseID           string       `json:"case_id"`
	RawMapping       AssetMapping `json:"raw_mapping"`
	EffectiveMapping AssetMapping `json:"effective_mapping"`
	ReasonCodes      []string     `json:"reason_codes"`
}

type CausalReplayReport struct {
	Version                        string                   `json:"version"`
	PolicyVersion                  string                   `json:"policy_version"`
	SourceManifestFingerprint      string                   `json:"source_manifest_fingerprint"`
	SourceRunIDs                   []string                 `json:"source_run_ids"`
	EvaluationCount                int                      `json:"evaluation_count"`
	Raw                            CausalReplayMetrics      `json:"raw"`
	Guarded                        CausalReplayMetrics      `json:"guarded"`
	TotalAbstentions               int                      `json:"total_abstentions"`
	DirectToUnresolved             int                      `json:"direct_to_unresolved_abstentions"`
	ProxyToUnresolved              int                      `json:"proxy_to_unresolved_abstentions"`
	TruePositiveMappingsSuppressed int                      `json:"true_positive_mappings_suppressed"`
	FalsePositiveMappingsRemoved   int                      `json:"false_positive_mappings_removed"`
	NewlyCreatedFalseNegatives     int                      `json:"newly_created_false_negatives"`
	AffectedCases                  []string                 `json:"affected_cases"`
	ReasonCodeDistribution         map[string]int           `json:"reason_code_distribution"`
	Abstentions                    []CausalReplayAbstention `json:"abstentions"`
}

type replayObservation struct {
	caseID     string
	label      DiagnosticLabel
	raw        StructuredResult
	effective  StructuredResult
	rawRes     *PolicyResolution
	guardedRes *PolicyResolution
	valid      bool
	guard      CausalConsistencyDecision
	event      DiagnosticEvent
}

func BuildCausalConsistencyReplay(manifest DiagnosticManifest, reports []DiagnosticRunReport, resolver assetresolution.Resolver) (CausalReplayReport, error) {
	if len(reports) != len(AcceptedC1BRunIDs) {
		return CausalReplayReport{}, fmt.Errorf("causal replay requires exactly %d accepted C1B reports", len(AcceptedC1BRunIDs))
	}
	events := map[string]DiagnosticEvent{}
	for _, event := range manifest.Events {
		events[event.ID] = event
	}
	accepted := map[string]bool{}
	for _, id := range AcceptedC1BRunIDs {
		accepted[id] = true
	}
	seenRuns := map[string]bool{}
	report := CausalReplayReport{
		Version:                   "ai-shadow-causal-consistency-replay-v1",
		PolicyVersion:             CausalConsistencyPolicyVersion,
		SourceManifestFingerprint: manifest.Fingerprint,
		ReasonCodeDistribution:    map[string]int{},
	}
	observations := []replayObservation{}
	affected := map[string]bool{}

	for _, source := range reports {
		if !accepted[source.RunID] || seenRuns[source.RunID] {
			return CausalReplayReport{}, fmt.Errorf("unaccepted or duplicate C1B run %q", source.RunID)
		}
		seenRuns[source.RunID] = true
		if source.ManifestFingerprint != manifest.Fingerprint || source.PromptVersion != PromptVersion || source.OutputContract != SchemaVersion || len(source.Repetitions) != 1 {
			return CausalReplayReport{}, fmt.Errorf("C1B run %s has incompatible frozen identity", source.RunID)
		}
		report.SourceRunIDs = append(report.SourceRunIDs, source.RunID)
		for _, evaluation := range source.Repetitions[0].Cases {
			event, ok := events[evaluation.CaseID]
			if !ok {
				return CausalReplayReport{}, fmt.Errorf("C1B run %s contains unknown case %s", source.RunID, evaluation.CaseID)
			}
			observation := replayObservation{caseID: evaluation.CaseID, label: event.Label, rawRes: evaluation.DeterministicResolution, valid: evaluation.ValidationStatus == "accepted" && evaluation.ModelClassification != nil && evaluation.DeterministicResolution != nil, event: event}
			if evaluation.ModelClassification != nil {
				observation.raw = *evaluation.ModelClassification
				observation.guard = ApplyCausalConsistencyGuard(observation.raw, event.Input, resolver)
				observation.effective = observation.raw
				observation.effective.MappingStatus = observation.guard.EffectiveMapping.MappingStatus
				observation.effective.DirectIssuer = observation.guard.EffectiveMapping.DirectIssuer
				observation.effective.ProxyExposure = observation.guard.EffectiveMapping.ProxyExposure
				guardedResolution := resolveEffectiveMapping(observation.guard, event.Input, resolver)
				observation.guardedRes = &guardedResolution
			}
			observations = append(observations, observation)
			if !observation.guard.Abstained {
				continue
			}
			report.TotalAbstentions++
			affected[evaluation.CaseID] = true
			switch observation.guard.RawMapping.MappingStatus {
			case "DIRECT":
				report.DirectToUnresolved++
			case "PROXY":
				report.ProxyToUnresolved++
			}
			for _, reason := range observation.guard.ReasonCodes {
				report.ReasonCodeDistribution[reason]++
			}
			rawCorrect := replaySemanticCorrect(event, observation.raw, resolver)
			guardedCorrect := replaySemanticCorrect(event, observation.effective, resolver)
			if rawCorrect && !guardedCorrect {
				report.TruePositiveMappingsSuppressed++
				report.NewlyCreatedFalseNegatives++
			}
			if !rawCorrect && guardedCorrect {
				report.FalsePositiveMappingsRemoved++
			}
			report.Abstentions = append(report.Abstentions, CausalReplayAbstention{
				RunID: source.RunID, CaseID: evaluation.CaseID, RawMapping: observation.guard.RawMapping,
				EffectiveMapping: observation.guard.EffectiveMapping, ReasonCodes: append([]string(nil), observation.guard.ReasonCodes...),
			})
		}
	}
	if len(observations) != len(AcceptedC1BRunIDs)*len(manifest.Events) {
		return CausalReplayReport{}, fmt.Errorf("causal replay has %d evaluations, want %d", len(observations), len(AcceptedC1BRunIDs)*len(manifest.Events))
	}
	for id := range affected {
		report.AffectedCases = append(report.AffectedCases, id)
	}
	sort.Strings(report.AffectedCases)
	sort.Strings(report.SourceRunIDs)
	sort.Slice(report.Abstentions, func(i, j int) bool {
		if report.Abstentions[i].CaseID == report.Abstentions[j].CaseID {
			return report.Abstentions[i].RunID < report.Abstentions[j].RunID
		}
		return report.Abstentions[i].CaseID < report.Abstentions[j].CaseID
	})
	report.EvaluationCount = len(observations)
	report.Raw = calculateReplayMetrics(observations, false, resolver, len(manifest.Events))
	report.Guarded = calculateReplayMetrics(observations, true, resolver, len(manifest.Events))
	return report, nil
}

func calculateReplayMetrics(observations []replayObservation, guarded bool, resolver assetresolution.Resolver, caseCount int) CausalReplayMetrics {
	metrics := CausalReplayMetrics{}
	valid, semantic, endToEnd := 0, 0, 0
	directExpected, directPredicted, directCorrect := 0, 0, 0
	proxyExpected, proxyCorrect := 0, 0
	unresolvedExpected, unresolvedCorrect := 0, 0
	byCase := map[string][]string{}
	for _, observation := range observations {
		if observation.label.MappingStatus == "DIRECT" {
			directExpected++
		}
		if observation.label.MappingStatus == "PROXY" {
			proxyExpected++
		}
		if observation.label.MappingStatus == "UNRESOLVED" {
			unresolvedExpected++
		}
		if !observation.valid {
			byCase[observation.caseID] = append(byCase[observation.caseID], "<invalid>")
			continue
		}
		valid++
		output, resolution := observation.raw, observation.rawRes
		if guarded {
			output, resolution = observation.effective, observation.guardedRes
		}
		correct := replaySemanticCorrect(observation.event, output, resolver)
		if correct {
			semantic++
		}
		resolutionCorrect := resolutionMatchesLabel(observation.event, resolution, resolver)
		if correct && resolutionCorrect {
			endToEnd++
		}
		if correct && !resolutionCorrect {
			metrics.ResolverFailures++
		}
		identity := output.MappingStatus + "|" + assetresolution.CanonicalizeIssuerName(output.DirectIssuer) + "|" + output.ProxyExposure
		byCase[observation.caseID] = append(byCase[observation.caseID], identity)
		if output.MappingStatus == "DIRECT" {
			directPredicted++
			if observation.label.MappingStatus == "DIRECT" && issuerMatchesLabel(observation.event, &output, resolver) {
				directCorrect++
			} else {
				metrics.FalseDirect++
			}
		}
		if observation.label.MappingStatus == "PROXY" && output.MappingStatus == "PROXY" && output.ProxyExposure == observation.label.ProxyExposure {
			proxyCorrect++
		}
		if output.MappingStatus == "PROXY" && (observation.label.MappingStatus != "PROXY" || output.ProxyExposure != observation.label.ProxyExposure) {
			metrics.FalseProxy++
		}
		if observation.label.MappingStatus == "UNRESOLVED" && output.MappingStatus == "UNRESOLVED" {
			unresolvedCorrect++
		}
	}
	repeatable := 0
	for _, identities := range byCase {
		if len(identities) == len(AcceptedC1BRunIDs) && allEqual(identities) {
			repeatable++
		}
	}
	metrics.FinalValidity = causalReplayRate(valid, len(observations))
	metrics.SemanticAccuracy = causalReplayRate(semantic, len(observations))
	metrics.EndToEndAccuracy = causalReplayRate(endToEnd, len(observations))
	metrics.DirectRecall = causalReplayRate(directCorrect, directExpected)
	metrics.DirectPrecision = causalReplayRate(directCorrect, directPredicted)
	metrics.ProxyAccuracy = causalReplayRate(proxyCorrect, proxyExpected)
	metrics.UnresolvedAccuracy = causalReplayRate(unresolvedCorrect, unresolvedExpected)
	metrics.SemanticRepeatability = causalReplayRate(repeatable, caseCount)
	return metrics
}

func replaySemanticCorrect(event DiagnosticEvent, output StructuredResult, resolver assetresolution.Resolver) bool {
	if output.MappingStatus != event.Label.MappingStatus {
		return false
	}
	switch output.MappingStatus {
	case "DIRECT":
		return issuerMatchesLabel(event, &output, resolver)
	case "PROXY":
		return output.ProxyExposure == event.Label.ProxyExposure
	case "UNRESOLVED":
		return true
	default:
		return false
	}
}

func causalReplayRate(numerator, denominator int) CausalReplayRate {
	result := CausalReplayRate{Numerator: numerator, Denominator: denominator}
	if denominator > 0 {
		result.Value = float64(numerator) / float64(denominator)
	}
	return result
}

func allEqual(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values[1:] {
		if !strings.EqualFold(value, values[0]) {
			return false
		}
	}
	return true
}
