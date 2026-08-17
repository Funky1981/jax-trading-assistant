package aishadow

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestV5SchemaIsClosedResolverDerivedAndComplete(t *testing.T) {
	exposures := mustProxyExposures(t)
	schema := V5OutputSchema(exposures)
	if schema["additionalProperties"] != false || !reflect.DeepEqual(schema["required"], requiredV5ResultFields) {
		t.Fatalf("unexpected v5 root shape: %#v", schema)
	}
	properties := schema["properties"].(map[string]any)
	if len(properties) != 12 {
		t.Fatalf("v5 must retain ten fields and add exactly two: %d", len(properties))
	}
	attributions := properties["issuer_attributions"].(map[string]any)
	if attributions["maxItems"] != 10 {
		t.Fatalf("issuer attribution max changed: %#v", attributions)
	}
	item := attributions["items"].(map[string]any)
	if item["additionalProperties"] != false || !reflect.DeepEqual(item["required"], []string{"issuer", "causal_role"}) {
		t.Fatalf("issuer attribution item is not closed: %#v", item)
	}
	candidates := properties["principal_proxy_candidates"].(map[string]any)
	if candidates["maxItems"] != len(exposures) || !reflect.DeepEqual(candidates["items"].(map[string]any)["enum"], exposures) {
		t.Fatalf("proxy vocabulary is not resolver-derived: %#v", candidates)
	}
	if contains(exposures, NoProxyExposure) {
		t.Fatal("resolver-derived principal proxy enum must exclude NONE")
	}
	if err := validateOpenAIStructuredOutputSchema(schema); err != nil {
		t.Fatalf("v5 schema is incompatible with strict Structured Outputs subset: %v", err)
	}
}

func TestV5OpenAIWireUsesCanonicalStrictSchemaWithoutToolsOrLabels(t *testing.T) {
	exposures := mustProxyExposures(t)
	request, err := V5InitialRequest(v5TestInput(), exposures)
	if err != nil {
		t.Fatal(err)
	}
	config := OpenAIDiagnosticConfig{
		Runtime: Config{Model: OpenAIDiagnosticLunaModel}, ExperimentID: OpenAIC1E3GeneralizationExperimentID,
		ReasoningEffort: OpenAIDiagnosticReasoningEffort, MaxOutputTokens: 256,
		OutputContractMode: OpenAIOutputContractStrictJSONSchema,
		PromptVersion:      V5PromptVersion, OutputContract: V5SchemaVersion, CausalPolicy: CausalAttributionPolicyVersion,
	}
	raw, err := marshalOpenAIDiagnosticRequest(config, request)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["store"] != false || wire["service_tier"] != OpenAIStructuredOutputsServiceTier || wire["max_output_tokens"] != float64(256) {
		t.Fatalf("wire safety/defaults changed: %s", raw)
	}
	reasoning := wire["reasoning"].(map[string]any)
	if reasoning["effort"] != OpenAIDiagnosticReasoningEffort {
		t.Fatalf("reasoning effort changed: %s", raw)
	}
	format := wire["text"].(map[string]any)["format"].(map[string]any)
	if format["type"] != "json_schema" || format["name"] != V5OpenAISchemaName || format["strict"] != true {
		t.Fatalf("v5 strict schema binding missing: %s", raw)
	}
	wireSchema := format["schema"].(map[string]any)
	canonicalRaw, _ := json.Marshal(request.Schema)
	wireSchemaRaw, _ := json.Marshal(wireSchema)
	if string(canonicalRaw) != string(wireSchemaRaw) {
		t.Fatalf("wire schema diverged from canonical ProviderRequest.Schema")
	}
	for _, forbidden := range []string{"tools", "web_search", "retrieval", "expected_mapping", "expected_issuer", "adjudicated_label"} {
		if strings.Contains(string(raw), `"`+forbidden+`"`) {
			t.Fatalf("wire body leaked forbidden field %q: %s", forbidden, raw)
		}
	}
}

func TestV5ProviderSchemaHashFailsClosed(t *testing.T) {
	request, err := V5InitialRequest(v5TestInput(), mustProxyExposures(t))
	if err != nil {
		t.Fatal(err)
	}
	request.SchemaSHA256 = strings.Repeat("0", 64)
	config := OpenAIDiagnosticConfig{
		Runtime: Config{Model: OpenAIDiagnosticLunaModel}, ExperimentID: OpenAIC1E3BoundaryExperimentID,
		ReasoningEffort: OpenAIDiagnosticReasoningEffort, MaxOutputTokens: 256,
		OutputContractMode: OpenAIOutputContractStrictJSONSchema,
		PromptVersion:      V5PromptVersion, OutputContract: V5SchemaVersion, CausalPolicy: CausalAttributionPolicyVersion,
	}
	if _, err := marshalOpenAIDiagnosticRequest(config, request); err == nil || !strings.Contains(err.Error(), "schema hash mismatch") {
		t.Fatalf("wrong v5 schema hash did not fail closed: %v", err)
	}
}

func TestContractVersionRoutingFailsClosed(t *testing.T) {
	tests := []struct{ prompt, contract, policy string }{
		{V5PromptVersion, SchemaVersion, CausalConsistencyPolicyVersion},
		{PromptVersion, V5SchemaVersion, CausalAttributionPolicyVersion},
		{V5PromptVersion, V5SchemaVersion, CausalConsistencyPolicyVersion},
		{V5PromptVersion, V5SchemaVersion, "unknown-policy"},
		{"wrong-prompt", V5SchemaVersion, CausalAttributionPolicyVersion},
		{V5PromptVersion, "unknown-contract", CausalAttributionPolicyVersion},
	}
	for _, test := range tests {
		if err := ValidateContractRoute(test.prompt, test.contract, test.policy); err == nil {
			t.Fatalf("mixed/unknown route accepted: %+v", test)
		}
	}
	if err := ValidateContractRoute(PromptVersion, SchemaVersion, CausalConsistencyPolicyVersion); err != nil {
		t.Fatalf("historical v4 route no longer works: %v", err)
	}
	if err := ValidateContractRoute(V5PromptVersion, V5SchemaVersion, CausalAttributionPolicyVersion); err != nil {
		t.Fatalf("v5 route rejected: %v", err)
	}
}

func TestC1E3ProfilesAreRegisteredButExecutionIsLabelGated(t *testing.T) {
	for _, identity := range []string{DiagnosticProfileGeneralizationV2, DiagnosticProfileBoundaryV2} {
		profile, err := LoadDiagnosticEvaluationProfile(identity)
		if err != nil {
			t.Fatal(err)
		}
		prompt, output, policy := profile.executionVersions()
		if prompt != V5PromptVersion || output != V5SchemaVersion || policy != CausalAttributionPolicyVersion ||
			profile.RequiredModel != OpenAIDiagnosticLunaModel || profile.DefaultRepetitions != 1 || !profile.RequiresTypedAttributionLabels {
			t.Fatalf("C1E3 profile is not correctly frozen: %+v", profile)
		}
		_, err = PrepareDiagnostic(DiagnosticPaths{EvaluationProfileID: identity}, Config{Provider: OpenAIDiagnosticProvider, MaxEvents: profile.CaseCount}, DiagnosticSafetyState{RuntimeMode: "paper", MaximumLeverage: 1})
		if err == nil || !strings.Contains(err.Error(), "typed-attribution label sidecar") {
			t.Fatalf("C1E3 profile did not fail before holdout loading: %v", err)
		}
	}
}

func TestC1E3HostedCellsBindLunaV5Strictly(t *testing.T) {
	for _, identity := range []string{DiagnosticProfileGeneralizationV2, DiagnosticProfileBoundaryV2} {
		profile, err := LoadDiagnosticEvaluationProfile(identity)
		if err != nil {
			t.Fatal(err)
		}
		values := hostedConfigValues()
		delete(values, OpenAIDiagnosticAPIKeyEnv)
		values["JAX_AI_MODEL"] = OpenAIDiagnosticLunaModel
		values["JAX_AI_EXPERIMENT_ID"] = profile.RequiredExperimentID
		if profile.CaseCount == 48 {
			values["JAX_AI_MAX_EVENTS"] = "48"
		} else {
			values["JAX_AI_MAX_EVENTS"] = "32"
		}
		values[OpenAIDiagnosticContractModeEnv] = OpenAIStructuredOutputsMode
		values["JAX_AI_EXPERIMENT_BUDGET_USD"] = formatUSDMicros(profile.MaximumBudgetMicros)
		config, err := LoadOpenAIDiagnosticConfigForProfile(mapLookup(values), profile, false)
		if err != nil {
			t.Fatal(err)
		}
		if config.PromptVersion != V5PromptVersion || config.OutputContract != V5SchemaVersion || config.CausalPolicy != CausalAttributionPolicyVersion ||
			config.StructuredOutputSchemaName() != V5OpenAISchemaName || config.ServiceTier() != OpenAIStructuredOutputsServiceTier || config.InferenceExplicitlyAuthorized {
			t.Fatalf("C1E3 hosted cell binding is incomplete: %+v", config)
		}

		bad := hostedConfigValues()
		delete(bad, OpenAIDiagnosticAPIKeyEnv)
		bad["JAX_AI_MODEL"] = OpenAIDiagnosticSolModel
		bad["JAX_AI_EXPERIMENT_ID"] = profile.RequiredExperimentID
		bad["JAX_AI_MAX_EVENTS"] = values["JAX_AI_MAX_EVENTS"]
		bad[OpenAIDiagnosticContractModeEnv] = OpenAIStructuredOutputsMode
		bad["JAX_AI_EXPERIMENT_BUDGET_USD"] = values["JAX_AI_EXPERIMENT_BUDGET_USD"]
		if _, err := LoadOpenAIDiagnosticConfigForProfile(mapLookup(bad), profile, false); err == nil {
			t.Fatal("C1E3 profile accepted the wrong model")
		}
	}
	if _, err := LoadDiagnosticEvaluationProfile("wrong-v2-profile"); err == nil {
		t.Fatal("unknown C1E3 dataset/profile did not fail closed")
	}
}
