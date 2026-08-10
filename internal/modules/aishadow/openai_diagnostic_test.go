package aishadow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

type queuedHTTPDoer struct {
	responses []*http.Response
	errors    []error
	calls     int
	requests  []*http.Request
}

func (d *queuedHTTPDoer) Do(request *http.Request) (*http.Response, error) {
	d.requests = append(d.requests, request)
	index := d.calls
	d.calls++
	if index < len(d.errors) && d.errors[index] != nil {
		return nil, d.errors[index]
	}
	return d.responses[index], nil
}

func openAITestConfig() OpenAIDiagnosticConfig {
	return OpenAIDiagnosticConfig{
		Runtime: Config{Enabled: true, Provider: OpenAIDiagnosticProvider, Model: OpenAIDiagnosticModel, BaseURL: "https://api.openai.com", Timeout: time.Second, MaxEvents: 48},
		APIKey:  APISecret{value: "sk-test-only-do-not-use"}, ExperimentID: OpenAIDiagnosticExperimentID,
		ReasoningEffort: OpenAIDiagnosticReasoningEffort, MaxOutputTokens: 256,
		BudgetCeilingMicros: 1_000_000, InputPriceMicrosPerMillion: 5_000_000,
		CachedInputPriceMicrosPerMillion: 500_000, CacheWritePriceMicrosPerMillion: 6_250_000,
		OutputPriceMicrosPerMillion: 30_000_000,
	}
}

func openAILunaTestConfig() OpenAIDiagnosticConfig {
	config := openAITestConfig()
	config.Runtime.Model = OpenAIDiagnosticLunaModel
	config.BudgetCeilingMicros = OpenAIDiagnosticLunaMaximumBudgetMicros
	config.InputPriceMicrosPerMillion = 200_000
	config.CachedInputPriceMicrosPerMillion = 20_000
	config.CacheWritePriceMicrosPerMillion = 250_000
	config.OutputPriceMicrosPerMillion = 1_200_000
	return config
}

func openAITestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status, Body: io.NopCloser(strings.NewReader(body)),
		Header: http.Header{"X-Request-Id": []string{"req-test"}},
	}
}

func completedOpenAIResponse(content string, inputTokens, outputTokens int) string {
	return completedOpenAIResponseForModel(content, OpenAIDiagnosticSolModel+"-2026-08-01", "fp_openai_test", inputTokens, outputTokens)
}

func completedOpenAIResponseForModel(content, model, fingerprint string, inputTokens, outputTokens int) string {
	return `{
  "id":"resp_test",
  "model":` + strconvQuote(model) + `,
  "system_fingerprint":` + strconvQuote(fingerprint) + `,
  "status":"completed",
  "output":[{"type":"message","content":[{"type":"output_text","text":` + strconvQuote(content) + `}]}],
  "usage":{"input_tokens":` + itoa(inputTokens) + `,"input_tokens_details":{"cached_tokens":11,"cache_write_tokens":7},"output_tokens":` + itoa(outputTokens) + `,"output_tokens_details":{"reasoning_tokens":13},"total_tokens":` + itoa(inputTokens+outputTokens) + `}
}`
}

func TestLoadOpenAIDiagnosticConfigFailsClosed(t *testing.T) {
	values := hostedConfigValues()
	delete(values, OpenAIDiagnosticAPIKeyEnv)
	if _, err := LoadOpenAIDiagnosticConfig(mapLookup(values)); err == nil || !strings.Contains(err.Error(), OpenAIDiagnosticAPIKeyEnv) {
		t.Fatalf("missing API key was not rejected: %v", err)
	}

	values = hostedConfigValues()
	values["JAX_AI_PROVIDER"] = "ollama"
	if _, err := LoadOpenAIDiagnosticConfig(mapLookup(values)); err == nil || !strings.Contains(err.Error(), "JAX_AI_PROVIDER=openai") {
		t.Fatalf("wrong provider was not rejected: %v", err)
	}

	values = hostedConfigValues()
	values["JAX_AI_MODEL"] = "gpt-5.6"
	if _, err := LoadOpenAIDiagnosticConfig(mapLookup(values)); err == nil || !strings.Contains(err.Error(), OpenAIDiagnosticModel) {
		t.Fatalf("wrong model was not rejected: %v", err)
	}
}

func TestLoadOpenAIDiagnosticConfigAcceptsOnlySolAndLuna(t *testing.T) {
	for _, model := range []string{OpenAIDiagnosticSolModel, OpenAIDiagnosticLunaModel} {
		t.Run(model, func(t *testing.T) {
			values := hostedConfigValues()
			values["JAX_AI_MODEL"] = model
			if model == OpenAIDiagnosticLunaModel {
				values["JAX_AI_EXPERIMENT_BUDGET_USD"] = "0.12"
			}
			config, err := LoadOpenAIDiagnosticConfig(mapLookup(values))
			if err != nil {
				t.Fatal(err)
			}
			if config.Runtime.Model != model {
				t.Fatalf("selected model=%s, want %s", config.Runtime.Model, model)
			}
		})
	}
	values := hostedConfigValues()
	values["JAX_AI_MODEL"] = "gpt-5.6-terra"
	if _, err := LoadOpenAIDiagnosticConfig(mapLookup(values)); err == nil {
		t.Fatal("arbitrary OpenAI model was accepted")
	}
}

func TestLoadOpenAIDiagnosticConfigAppliesLunaBudgetCap(t *testing.T) {
	values := hostedConfigValues()
	values["JAX_AI_MODEL"] = OpenAIDiagnosticLunaModel
	values["JAX_AI_EXPERIMENT_BUDGET_USD"] = "0.120001"
	if _, err := LoadOpenAIDiagnosticConfig(mapLookup(values)); err == nil || !strings.Contains(err.Error(), "0.12") {
		t.Fatalf("Luna budget cap was not enforced: %v", err)
	}
}

func TestOpenAIDiagnosticConfigCannotSerializeOrFormatCredential(t *testing.T) {
	config := openAITestConfig()
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, rendered := range []string{string(raw), fmt.Sprintf("%+v", config), fmt.Sprintf("%#v", config), config.APIKey.String()} {
		if strings.Contains(rendered, "sk-test-only-do-not-use") {
			t.Fatalf("credential leaked from configuration rendering: %s", rendered)
		}
	}
}

func TestOpenAIDiagnosticClientParsesResponseAndUsageWithoutStructuredOutputs(t *testing.T) {
	doer := &queuedHTTPDoer{responses: []*http.Response{openAITestResponse(http.StatusOK, completedOpenAIResponse(unresolvedJSON(), 1000, 100))}}
	client := NewOpenAIDiagnosticClient(openAITestConfig(), doer)
	response, err := client.Complete(ProviderRequest{System: "frozen-system", User: "frozen-user", EventID: "case-1", AttemptNumber: 1, RequestKind: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Content != unresolvedJSON() || response.ModelIdentifier != "gpt-5.6-sol-2026-08-01" || response.SystemFingerprint != "fp_openai_test" || response.RequestID != "req-test" || response.ResponseID != "resp_test" {
		t.Fatalf("unexpected response metadata: %+v", response)
	}
	if response.Usage.InputTokens != 1000 || response.Usage.CacheMissTokens != 982 || response.Usage.OutputTokens != 100 || response.Usage.ReasoningTokens != 13 || response.Usage.CachedTokens != 11 || response.Usage.CacheWriteTokens != 7 {
		t.Fatalf("unexpected usage: %+v", response.Usage)
	}
	if doer.calls != 1 {
		t.Fatalf("HTTP calls=%d", doer.calls)
	}
	request := doer.requests[0]
	if got := request.Header.Get("Authorization"); got != "Bearer sk-test-only-do-not-use" {
		t.Fatalf("authorization header was not applied")
	}
	raw, _ := io.ReadAll(request.Body)
	body := string(raw)
	for _, want := range []string{`"model":"gpt-5.6-sol"`, `"effort":"none"`, `"max_output_tokens":256`, `"store":false`, `"role":"system"`, `"role":"user"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body missing %s: %s", want, body)
		}
	}
	if strings.Contains(body, `"text"`) || strings.Contains(body, "json_schema") {
		t.Fatalf("A1 request silently enabled provider structured outputs: %s", body)
	}
	snapshot := client.ExperimentSnapshot()
	if snapshot.RequestCount != 1 || snapshot.ActualCalculableCostUSD != "0.00796" || snapshot.AccountedCostUSD != "0.00796" || snapshot.RemainingBudgetUSD != "0.99204" ||
		snapshot.CostByCategory.TotalUSD != "0.00796" || len(snapshot.ReturnedModels) != 1 || len(snapshot.SystemFingerprints) != 1 {
		t.Fatalf("unexpected experiment snapshot: %+v", snapshot)
	}
}

func TestOpenAIDiagnosticClientParsesLunaResponseAndRequiresExactIdentity(t *testing.T) {
	config := openAILunaTestConfig()
	body := completedOpenAIResponseForModel(unresolvedJSON(), OpenAIDiagnosticLunaModel, "fp_luna_test", 1000, 100)
	doer := &queuedHTTPDoer{responses: []*http.Response{openAITestResponse(http.StatusOK, body)}}
	client := NewOpenAIDiagnosticClient(config, doer)
	response, err := client.Complete(ProviderRequest{System: "frozen-system", User: "frozen-user", RequestKind: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	if response.ModelIdentifier != OpenAIDiagnosticLunaModel || response.SystemFingerprint != "fp_luna_test" || response.Usage.CacheMissTokens != 982 {
		t.Fatalf("unexpected Luna response evidence: %+v", response)
	}
	raw, _ := io.ReadAll(doer.requests[0].Body)
	for _, want := range []string{`"model":"gpt-5.6-luna"`, `"effort":"none"`, `"store":false`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("Luna request missing %s: %s", want, raw)
		}
	}
	for _, forbidden := range []string{`"tools"`, "web_search", "file_search", "json_schema"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("Luna request enabled forbidden feature %q: %s", forbidden, raw)
		}
	}
	snapshot := client.ExperimentSnapshot()
	if snapshot.RequestedModel != OpenAIDiagnosticLunaModel || snapshot.ReasoningEffort != "none" || snapshot.ActualCalculableCostUSD != "0.00032" || snapshot.RemainingBudgetUSD != "0.11968" {
		t.Fatalf("unexpected Luna experiment snapshot: %+v", snapshot)
	}

	mismatch := completedOpenAIResponseForModel(unresolvedJSON(), OpenAIDiagnosticLunaModel+"-2026-08-01", "fp_luna_test", 100, 20)
	doer = &queuedHTTPDoer{responses: []*http.Response{openAITestResponse(http.StatusOK, mismatch)}}
	_, err = NewOpenAIDiagnosticClient(config, doer).Complete(ProviderRequest{System: "system", User: "user"})
	if err == nil || !strings.Contains(err.Error(), "kind=model_identity") {
		t.Fatalf("non-published Luna snapshot identity was not rejected: %v", err)
	}
}

func TestOpenAIDiagnosticClientRejectsUnsupportedConfiguredModelBeforeHTTP(t *testing.T) {
	config := openAITestConfig()
	config.Runtime.Model = "gpt-5.6-terra"
	doer := &queuedHTTPDoer{}
	_, err := NewOpenAIDiagnosticClient(config, doer).Complete(ProviderRequest{System: "system", User: "user"})
	if err == nil || !strings.Contains(err.Error(), "kind=configured_model") || doer.calls != 0 {
		t.Fatalf("unsupported configured model did not fail before HTTP: err=%v calls=%d", err, doer.calls)
	}
}

func TestOpenAIDiagnosticClientSanitizesHTTPError(t *testing.T) {
	doer := &queuedHTTPDoer{responses: []*http.Response{openAITestResponse(http.StatusBadRequest, `{"error":{"message":"bad sk-test-only-do-not-use","type":"invalid_request_error","code":"sk-test-only-do-not-use"}}`)}}
	client := NewOpenAIDiagnosticClient(openAITestConfig(), doer)
	_, err := client.Complete(ProviderRequest{System: "system", User: "user"})
	if err == nil || !strings.Contains(err.Error(), "http_status=400") || !strings.Contains(err.Error(), "provider_code=<redacted>") {
		t.Fatalf("unexpected HTTP error: %v", err)
	}
	if strings.Contains(err.Error(), "sk-test-only-do-not-use") || strings.Contains(err.Error(), "bad sk") {
		t.Fatalf("credential or provider message leaked through error: %v", err)
	}
	snapshot := client.ExperimentSnapshot()
	if snapshot.ProviderErrorCount != 1 || snapshot.AmbiguousLiabilityUSD != "0.00" {
		t.Fatalf("unexpected error accounting: %+v", snapshot)
	}
}

func TestOpenAIDiagnosticClientAccountsTimeoutAsAmbiguous(t *testing.T) {
	doer := &queuedHTTPDoer{errors: []error{context.DeadlineExceeded}}
	client := NewOpenAIDiagnosticClient(openAITestConfig(), doer)
	_, err := client.Complete(ProviderRequest{System: "system", User: "user", EventID: "case-1", AttemptNumber: 1})
	if err == nil || !strings.Contains(err.Error(), "timeout=true") || strings.Contains(err.Error(), "sk-test-only-do-not-use") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
	snapshot := client.ExperimentSnapshot()
	if snapshot.TimeoutCount != 1 || snapshot.AmbiguousLiabilityUSD == "0.00" || !snapshot.Failures[0].AmbiguousSpend {
		t.Fatalf("timeout was not conservatively accounted: %+v", snapshot)
	}
}

func TestOpenAIDiagnosticClientFailsClosedWhenUsageIsMissing(t *testing.T) {
	body := `{"id":"resp_test","model":"gpt-5.6-sol","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{}"}]}]}`
	doer := &queuedHTTPDoer{responses: []*http.Response{openAITestResponse(http.StatusOK, body)}}
	client := NewOpenAIDiagnosticClient(openAITestConfig(), doer)
	_, err := client.Complete(ProviderRequest{System: "system", User: "user"})
	if err == nil || !strings.Contains(err.Error(), "kind=usage") {
		t.Fatalf("missing usage was not rejected: %v", err)
	}
	if snapshot := client.ExperimentSnapshot(); snapshot.AmbiguousLiabilityUSD == "0.00" || snapshot.ProviderErrorCount != 1 {
		t.Fatalf("missing usage was not conservatively accounted: %+v", snapshot)
	}
}

func TestOpenAIDiagnosticBudgetRejectsUnaffordableRequestBeforeHTTP(t *testing.T) {
	config := openAITestConfig()
	config.BudgetCeilingMicros = 1
	doer := &queuedHTTPDoer{}
	client := NewOpenAIDiagnosticClient(config, doer)
	_, err := client.Complete(ProviderRequest{System: "system", User: "user"})
	var budgetErr BudgetGuardError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("expected budget error, got %v", err)
	}
	if doer.calls != 0 {
		t.Fatalf("unaffordable request reached HTTP transport %d times", doer.calls)
	}
	if snapshot := client.ExperimentSnapshot(); snapshot.BudgetRejectionCount != 1 || snapshot.StopReason != "budget_guard_rejected_request" {
		t.Fatalf("unexpected budget rejection snapshot: %+v", snapshot)
	}
}

func TestOpenAIDiagnosticBudgetExhaustionStopsNextRequest(t *testing.T) {
	config := openAITestConfig()
	config.BudgetCeilingMicros = 10_000
	config.InputPriceMicrosPerMillion = 1_000_000
	config.OutputPriceMicrosPerMillion = 1_000_000
	config.MaxOutputTokens = 1
	doer := &queuedHTTPDoer{responses: []*http.Response{openAITestResponse(http.StatusOK, completedOpenAIResponse(unresolvedJSON(), 9000, 1))}}
	client := NewOpenAIDiagnosticClient(config, doer)
	if _, err := client.Complete(ProviderRequest{System: "s", User: "u"}); err != nil {
		t.Fatal(err)
	}
	_, err := client.Complete(ProviderRequest{System: "s", User: "u"})
	var budgetErr BudgetGuardError
	if !errors.As(err, &budgetErr) || doer.calls != 1 {
		t.Fatalf("budget exhaustion did not stop before second HTTP call: err=%v calls=%d", err, doer.calls)
	}
}

func TestOpenAIDiagnosticCorrectiveRetryAccountingUsesExistingPipeline(t *testing.T) {
	config := openAITestConfig()
	doer := &queuedHTTPDoer{responses: []*http.Response{
		openAITestResponse(http.StatusOK, completedOpenAIResponse(`{"mapping_status":"PROXY"}`, 100, 20)),
		openAITestResponse(http.StatusOK, completedOpenAIResponse(unresolvedJSON(), 110, 30)),
	}}
	provider := NewOpenAIDiagnosticClient(config, doer)
	resolver := testAssetResolver(t)
	input := testInput("Unknown local event", "unknown")
	_, attempts, _, err := analyseEvent(config.Runtime, provider, resolver, "run", DiagnosticManifestVersion, "case-1", "fingerprint", input, []string{"OIL_CATEGORY"})
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 2 || provider.ExperimentSnapshot().RetryCount != 1 || doer.calls != 2 {
		t.Fatalf("corrective retry semantics changed: attempts=%d snapshot=%+v calls=%d", len(attempts), provider.ExperimentSnapshot(), doer.calls)
	}
}

func hostedConfigValues() map[string]string {
	return map[string]string{
		"JAX_AI_SHADOW_ENABLED": "true", "JAX_AI_PROVIDER": "openai", "JAX_AI_MODEL": OpenAIDiagnosticModel,
		"JAX_AI_TIMEOUT_SECONDS": "120", "JAX_AI_MAX_EVENTS": "48", OpenAIDiagnosticAPIKeyEnv: "sk-test-only-do-not-use",
		"JAX_AI_EXPERIMENT_ID": "A1", "JAX_AI_REASONING_EFFORT": "none", "JAX_AI_MAX_OUTPUT_TOKENS": "256",
		"JAX_AI_EXPERIMENT_BUDGET_USD": "1.00", "JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS": "5.00",
		"JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS": "0.50", "JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS": "6.25",
		"JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS": "30.00", OpenAIDiagnosticInferenceAuthEnv: "false",
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
}

func strconvQuote(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func itoa(value int) string { return strconv.Itoa(value) }
