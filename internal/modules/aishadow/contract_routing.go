package aishadow

import (
	"fmt"
	"reflect"
)

const (
	DiagnosticValidatorV4 = "parse-validate-and-guard-v4"
	DiagnosticValidatorV5 = "parse-validate-and-apply-v5"
)

type diagnosticExecutionRoute string

const (
	diagnosticRouteHistoricalV4       diagnosticExecutionRoute = "historical-v4"
	diagnosticRouteHistoricalC1EV5    diagnosticExecutionRoute = "historical-c1e-v5"
	diagnosticRouteC1F3               diagnosticExecutionRoute = "c1f-c1f3"
	diagnosticRouteC1FRepeatabilityR2 diagnosticExecutionRoute = "c1f-repeatability-r2"
	diagnosticRouteC1FRepeatabilityR3 diagnosticExecutionRoute = "c1f-repeatability-r3"
)

type diagnosticExecutionContract struct {
	Route     diagnosticExecutionRoute
	Prompt    string
	Output    string
	Validator string
	Policy    string
	Scoring   string
}

func diagnosticExecutionContractForProfile(profile DiagnosticEvaluationProfile) (diagnosticExecutionContract, error) {
	var expected diagnosticExecutionContract
	switch profile.Identity {
	case DiagnosticProfileOriginal, DiagnosticProfileGeneralization, DiagnosticProfileBoundary:
		expected = diagnosticExecutionContract{Route: diagnosticRouteHistoricalV4, Prompt: PromptVersion, Output: SchemaVersion, Validator: DiagnosticValidatorV4, Policy: CausalConsistencyPolicyVersion}
	case DiagnosticProfileGeneralizationV2, DiagnosticProfileBoundaryV2:
		expected = diagnosticExecutionContract{Route: diagnosticRouteHistoricalC1EV5, Prompt: V5PromptVersion, Output: V5SchemaVersion, Validator: DiagnosticValidatorV5, Policy: CausalAttributionPolicyVersion, Scoring: CausalAttributionScoringVersion}
	case C1F3ProfileGeneralization, C1F3ProfileBoundary:
		frozen, err := LoadC1F3EvaluationProfile(profile.Identity)
		if err != nil {
			return diagnosticExecutionContract{}, err
		}
		expected = diagnosticExecutionContract{Route: diagnosticRouteC1F3, Prompt: frozen.Prompt, Output: frozen.OutputContract, Validator: frozen.Validator, Policy: frozen.AttributionPolicy, Scoring: frozen.Scoring}
		if frozen.Validator != C1FValidatorVersion {
			return diagnosticExecutionContract{}, fmt.Errorf("frozen C1F3 profile %s validator identity changed", profile.Identity)
		}
	case C1F3RepeatabilityProfileIdentity:
		frozen, err := FrozenC1F3RepeatabilityProfile()
		if err != nil {
			return diagnosticExecutionContract{}, err
		}
		if frozen.SourceProfileIdentity != C1F3ProfileGeneralization || frozen.FrozenSemanticStack.Validator != C1FValidatorVersion {
			return diagnosticExecutionContract{}, fmt.Errorf("frozen C1F repeatability profile %s contract identity changed", profile.Identity)
		}
		expected = diagnosticExecutionContract{Route: diagnosticRouteC1FRepeatabilityR2, Prompt: frozen.FrozenSemanticStack.Prompt, Output: frozen.FrozenSemanticStack.OutputContract, Validator: frozen.FrozenSemanticStack.Validator, Policy: frozen.FrozenSemanticStack.AttributionPolicy, Scoring: frozen.FrozenSemanticStack.Scoring}
	case C1F3RepeatabilityR3ProfileIdentity:
		frozen, err := FrozenC1F3RepeatabilityR3Profile()
		if err != nil {
			return diagnosticExecutionContract{}, err
		}
		if frozen.SourceProfileIdentity != C1F3ProfileGeneralization || frozen.FrozenSemanticStack.Validator != C1FValidatorVersion {
			return diagnosticExecutionContract{}, fmt.Errorf("frozen C1F repeatability profile %s contract identity changed", profile.Identity)
		}
		expected = diagnosticExecutionContract{Route: diagnosticRouteC1FRepeatabilityR3, Prompt: frozen.FrozenSemanticStack.Prompt, Output: frozen.FrozenSemanticStack.OutputContract, Validator: frozen.FrozenSemanticStack.Validator, Policy: frozen.FrozenSemanticStack.AttributionPolicy, Scoring: frozen.FrozenSemanticStack.Scoring}
	default:
		return diagnosticExecutionContract{}, fmt.Errorf("unknown diagnostic execution contract for profile %q", profile.Identity)
	}
	prompt, output, policy := profile.executionVersions()
	if prompt != expected.Prompt {
		return diagnosticExecutionContract{}, fmt.Errorf("profile %s route %s requires prompt %q", profile.Identity, expected.Route, expected.Prompt)
	}
	if output != expected.Output {
		return diagnosticExecutionContract{}, fmt.Errorf("profile %s route %s requires output contract %q", profile.Identity, expected.Route, expected.Output)
	}
	if profile.ExecutionValidatorVersion != expected.Validator {
		return diagnosticExecutionContract{}, fmt.Errorf("profile %s route %s requires validator %q", profile.Identity, expected.Route, expected.Validator)
	}
	if policy != expected.Policy {
		return diagnosticExecutionContract{}, fmt.Errorf("profile %s route %s requires policy %q", profile.Identity, expected.Route, expected.Policy)
	}
	if profile.ScoringVersion != expected.Scoring {
		return diagnosticExecutionContract{}, fmt.Errorf("profile %s route %s requires scorer %q", profile.Identity, expected.Route, expected.Scoring)
	}
	return expected, nil
}

func (r diagnosticExecutionRoute) usesC1F() bool {
	return r == diagnosticRouteC1F3 || r == diagnosticRouteC1FRepeatabilityR2 || r == diagnosticRouteC1FRepeatabilityR3
}

// ValidateContractRoute rejects mixed or unknown prompt/contract/policy cells
// before provider construction. Historical v4 and typed v5 routes are the
// only executable current-generation combinations.
func ValidateContractRoute(promptVersion, outputContract, causalPolicy string) error {
	switch outputContract {
	case SchemaVersion:
		if promptVersion != PromptVersion {
			return fmt.Errorf("v4 output contract requires prompt %q", PromptVersion)
		}
		if causalPolicy != CausalConsistencyPolicyVersion {
			return fmt.Errorf("v4 output contract requires historical policy %q", CausalConsistencyPolicyVersion)
		}
		return nil
	case V5SchemaVersion:
		if promptVersion != V5PromptVersion {
			return fmt.Errorf("v5 output contract requires prompt %q", V5PromptVersion)
		}
		if causalPolicy != CausalAttributionPolicyVersion {
			return fmt.Errorf("v5 output contract requires typed policy %q", CausalAttributionPolicyVersion)
		}
		return nil
	default:
		return fmt.Errorf("unsupported AI shadow output contract %q", outputContract)
	}
}

func ValidateV5ProviderRequestSchema(request ProviderRequest) error {
	if request.SchemaContract != V5SchemaVersion {
		return fmt.Errorf("v5 provider request requires schema contract %q", V5SchemaVersion)
	}
	properties, ok := request.Schema["properties"].(map[string]any)
	if !ok {
		return fmt.Errorf("v5 provider request schema has no properties")
	}
	candidates, ok := properties["principal_proxy_candidates"].(map[string]any)
	if !ok {
		return fmt.Errorf("v5 provider request schema has no principal_proxy_candidates")
	}
	items, ok := candidates["items"].(map[string]any)
	if !ok {
		return fmt.Errorf("v5 provider request proxy candidates have no items schema")
	}
	proxyExposures, ok := items["enum"].([]string)
	if !ok {
		return fmt.Errorf("v5 provider request proxy candidates have no bounded enum")
	}
	want := V5OutputSchema(proxyExposures)
	if !reflect.DeepEqual(request.Schema, want) {
		return fmt.Errorf("v5 provider request schema does not equal the canonical contract")
	}
	hash, err := fingerprint(request.Schema)
	if err != nil {
		return err
	}
	if request.SchemaSHA256 == "" || request.SchemaSHA256 != hash {
		return fmt.Errorf("v5 provider request schema hash mismatch")
	}
	return nil
}
