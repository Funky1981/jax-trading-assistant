package aishadow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func deepSeekTestConfig() DeepSeekDiagnosticConfig {
	return DeepSeekDiagnosticConfig{
		Runtime: Config{Enabled: true, Provider: DeepSeekDiagnosticProvider, Model: DeepSeekDiagnosticModel, BaseURL: "https://api.deepseek.com", Timeout: time.Second, MaxEvents: 48},
		APIKey:  APISecret{value: "ds-test-only-do-not-use"}, ExperimentID: DeepSeekDiagnosticExperimentID,
		ReasoningEffort: DeepSeekDiagnosticReasoningEffort, ThinkingMode: DeepSeekDiagnosticThinkingMode, MaxOutputTokens: 256,
		BudgetCeilingMicros: 100_000, CacheMissPriceMicrosPerMillion: 435_000,
		CacheHitPriceMicrosPerMillion: 3_625, OutputPriceMicrosPerMillion: 870_000,
	}
}

func deepSeekTestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status, Body: io.NopCloser(strings.NewReader(body)),
		Header: http.Header{"X-Request-Id": []string{"req-deepseek-test"}},
	}
}

func completedDeepSeekResponse(content, model, fingerprint string, prompt, hit, miss, completion, reasoning int) string {
	return fmt.Sprintf(`{
  "id":"chatcmpl_test","object":"chat.completion","model":%s,"system_fingerprint":%s,
  "choices":[{"index":0,"message":{"role":"assistant","content":%s},"finish_reason":"stop"}],
  "usage":{"prompt_tokens":%d,"prompt_cache_hit_tokens":%d,"prompt_cache_miss_tokens":%d,"completion_tokens":%d,"completion_tokens_details":{"reasoning_tokens":%d},"total_tokens":%d}
}`,
		strconvQuote(model), strconvQuote(fingerprint), strconvQuote(content), prompt, hit, miss, completion, reasoning, prompt+completion)
}

func TestLoadDeepSeekDiagnosticConfigFailsClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string]string)
		want   string
	}{
		{name: "missing API key", mutate: func(values map[string]string) { delete(values, DeepSeekDiagnosticAPIKeyEnv) }, want: DeepSeekDiagnosticAPIKeyEnv},
		{name: "wrong provider", mutate: func(values map[string]string) { values["JAX_AI_PROVIDER"] = "openai" }, want: "JAX_AI_PROVIDER=deepseek"},
		{name: "wrong model", mutate: func(values map[string]string) { values["JAX_AI_MODEL"] = "deepseek-chat" }, want: DeepSeekDiagnosticModel},
		{name: "thinking enabled", mutate: func(values map[string]string) { values[DeepSeekDiagnosticThinkingModeEnv] = "enabled" }, want: "JAX_AI_THINKING_MODE=disabled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := deepSeekConfigValues()
			tt.mutate(values)
			if _, err := LoadDeepSeekDiagnosticConfig(mapLookup(values)); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("configuration was not rejected: %v", err)
			}
		})
	}
}

func TestDeepSeekDiagnosticConfigCannotSerializeOrFormatCredential(t *testing.T) {
	config := deepSeekTestConfig()
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, rendered := range []string{string(raw), fmt.Sprintf("%+v", config), fmt.Sprintf("%#v", config), config.APIKey.String()} {
		if strings.Contains(rendered, "ds-test-only-do-not-use") {
			t.Fatalf("credential leaked from configuration rendering: %s", rendered)
		}
	}
}

func TestDeepSeekDiagnosticClientParsesChatCompletionAndEvidence(t *testing.T) {
	body := completedDeepSeekResponse(unresolvedJSON(), DeepSeekDiagnosticModel, "fp_deepseek_test", 1000, 100, 900, 100, 0)
	doer := &queuedHTTPDoer{responses: []*http.Response{deepSeekTestResponse(http.StatusOK, body)}}
	client := NewDeepSeekDiagnosticClient(deepSeekTestConfig(), doer)
	response, err := client.Complete(ProviderRequest{System: "frozen-system", User: "frozen-user", EventID: "case-1", AttemptNumber: 1, RequestKind: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Content != unresolvedJSON() || response.ModelIdentifier != DeepSeekDiagnosticModel || response.SystemFingerprint != "fp_deepseek_test" ||
		response.FinishReason != "stop" || response.RequestID != "req-deepseek-test" || response.ResponseID != "chatcmpl_test" {
		t.Fatalf("unexpected response evidence: %+v", response)
	}
	if response.Usage.InputTokens != 1000 || response.Usage.CachedTokens != 100 || response.Usage.CacheMissTokens != 900 ||
		response.Usage.OutputTokens != 100 || response.Usage.ReasoningTokens != 0 || response.Usage.TotalTokens != 1100 {
		t.Fatalf("unexpected usage: %+v", response.Usage)
	}
	if doer.calls != 1 || doer.requests[0].URL.String() != DeepSeekDiagnosticEndpoint {
		t.Fatalf("unexpected HTTP calls or endpoint: calls=%d requests=%+v", doer.calls, doer.requests)
	}
	if got := doer.requests[0].Header.Get("Authorization"); got != "Bearer ds-test-only-do-not-use" {
		t.Fatal("authorization header was not applied")
	}
	raw, _ := io.ReadAll(doer.requests[0].Body)
	requestBody := string(raw)
	for _, want := range []string{`"model":"deepseek-v4-pro"`, `"thinking":{"type":"disabled"}`, `"max_tokens":256`, `"role":"system"`, `"role":"user"`} {
		if !strings.Contains(requestBody, want) {
			t.Fatalf("request body missing %s: %s", want, requestBody)
		}
	}
	for _, forbidden := range []string{"response_format", "json_schema", `"tools"`, "reasoning_effort", "temperature"} {
		if strings.Contains(requestBody, forbidden) {
			t.Fatalf("DeepSeek A1 request enabled forbidden feature %q: %s", forbidden, requestBody)
		}
	}
	snapshot := client.ExperimentSnapshot()
	if snapshot.RequestCount != 1 || snapshot.RetryCount != 0 || snapshot.ThinkingMode != "disabled" ||
		len(snapshot.ReturnedModels) != 1 || len(snapshot.SystemFingerprints) != 1 || len(snapshot.FinishReasons) != 1 {
		t.Fatalf("unexpected experiment snapshot: %+v", snapshot)
	}
}

func TestDeepSeekDiagnosticClientFailsClosedOnIdentityAndThinkingDiscrepancies(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		want       string
		stopReason string
	}{
		{name: "missing fingerprint", body: completedDeepSeekResponse(unresolvedJSON(), DeepSeekDiagnosticModel, "", 100, 0, 100, 20, 0), want: "kind=system_fingerprint"},
		{name: "model mismatch", body: completedDeepSeekResponse(unresolvedJSON(), "deepseek-v4-pro-2026-04-24", "fp_test", 100, 0, 100, 20, 0), want: "kind=model_identity"},
		{name: "reasoning tokens", body: completedDeepSeekResponse(unresolvedJSON(), DeepSeekDiagnosticModel, "fp_test", 100, 0, 100, 20, 1), want: "kind=reasoning_tokens", stopReason: "non_thinking_reasoning_tokens_nonzero"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doer := &queuedHTTPDoer{responses: []*http.Response{deepSeekTestResponse(http.StatusOK, tt.body)}}
			client := NewDeepSeekDiagnosticClient(deepSeekTestConfig(), doer)
			_, err := client.Complete(ProviderRequest{System: "system", User: "user"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("discrepancy was not rejected: %v", err)
			}
			snapshot := client.ExperimentSnapshot()
			if snapshot.ProviderErrorCount != 1 || snapshot.StopReason != tt.stopReason {
				t.Fatalf("discrepancy evidence is incomplete: %+v", snapshot)
			}
		})
	}
}

func TestDeepSeekDiagnosticClientSanitizesProviderErrorAndHeaderEvidence(t *testing.T) {
	doer := &queuedHTTPDoer{responses: []*http.Response{deepSeekTestResponse(http.StatusBadRequest, `{"error":{"message":"bad ds-test-only-do-not-use","type":"invalid_request_error","code":"ds-test-only-do-not-use"}}`)}}
	client := NewDeepSeekDiagnosticClient(deepSeekTestConfig(), doer)
	_, err := client.Complete(ProviderRequest{System: "system", User: "user"})
	if err == nil || !strings.Contains(err.Error(), "http_status=400") || !strings.Contains(err.Error(), "provider_code=<redacted>") {
		t.Fatalf("unexpected provider error: %v", err)
	}
	rendered := err.Error() + fmt.Sprintf("%+v", client.ExperimentSnapshot())
	if strings.Contains(rendered, "ds-test-only-do-not-use") || strings.Contains(rendered, "Authorization") {
		t.Fatalf("secret or header leaked into evidence: %s", rendered)
	}
}

func TestDeepSeekDiagnosticClientAccountsTimeoutAsAmbiguous(t *testing.T) {
	doer := &queuedHTTPDoer{errors: []error{context.DeadlineExceeded}}
	client := NewDeepSeekDiagnosticClient(deepSeekTestConfig(), doer)
	_, err := client.Complete(ProviderRequest{System: "system", User: "user", EventID: "case-1", AttemptNumber: 1})
	if err == nil || !strings.Contains(err.Error(), "timeout=true") {
		t.Fatalf("unexpected timeout result: %v", err)
	}
	snapshot := client.ExperimentSnapshot()
	if snapshot.TimeoutCount != 1 || snapshot.AmbiguousLiabilityUSD == "0.00" || !snapshot.Failures[0].AmbiguousSpend {
		t.Fatalf("timeout liability was not retained: %+v", snapshot)
	}
}

func TestDeepSeekDiagnosticClientRejectsMalformedAndInconsistentUsage(t *testing.T) {
	tests := []struct {
		name string
		body string
		kind string
	}{
		{name: "malformed response", body: `{`, kind: "kind=decode"},
		{name: "missing usage", body: `{"id":"x","object":"chat.completion","model":"deepseek-v4-pro","system_fingerprint":"fp","choices":[]}`, kind: "kind=usage"},
		{name: "cache split inconsistent", body: completedDeepSeekResponse(unresolvedJSON(), DeepSeekDiagnosticModel, "fp", 100, 10, 80, 20, 0), kind: "kind=usage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doer := &queuedHTTPDoer{responses: []*http.Response{deepSeekTestResponse(http.StatusOK, tt.body)}}
			client := NewDeepSeekDiagnosticClient(deepSeekTestConfig(), doer)
			_, err := client.Complete(ProviderRequest{System: "system", User: "user"})
			if err == nil || !strings.Contains(err.Error(), tt.kind) {
				t.Fatalf("response was not rejected: %v", err)
			}
			if snapshot := client.ExperimentSnapshot(); snapshot.AmbiguousLiabilityUSD == "0.00" || snapshot.ProviderErrorCount != 1 {
				t.Fatalf("ambiguous usage liability missing: %+v", snapshot)
			}
		})
	}
}

func TestDeepSeekDiagnosticBudgetRejectsBeforeHTTPAndStopsAfterExhaustion(t *testing.T) {
	config := deepSeekTestConfig()
	config.BudgetCeilingMicros = 1
	doer := &queuedHTTPDoer{}
	client := NewDeepSeekDiagnosticClient(config, doer)
	_, err := client.Complete(ProviderRequest{System: "system", User: "user"})
	var budgetErr BudgetGuardError
	if !errors.As(err, &budgetErr) || doer.calls != 0 {
		t.Fatalf("unaffordable request reached HTTP: err=%v calls=%d", err, doer.calls)
	}

	config = deepSeekTestConfig()
	config.BudgetCeilingMicros = 10_000
	config.CacheMissPriceMicrosPerMillion = 1_000_000
	config.CacheHitPriceMicrosPerMillion = 1_000_000
	config.OutputPriceMicrosPerMillion = 1_000_000
	config.MaxOutputTokens = 1
	body := completedDeepSeekResponse(unresolvedJSON(), DeepSeekDiagnosticModel, "fp", 9000, 0, 9000, 1, 0)
	doer = &queuedHTTPDoer{responses: []*http.Response{deepSeekTestResponse(http.StatusOK, body)}}
	client = NewDeepSeekDiagnosticClient(config, doer)
	if _, err := client.Complete(ProviderRequest{System: "s", User: "u"}); err != nil {
		t.Fatal(err)
	}
	_, err = client.Complete(ProviderRequest{System: "s", User: "u"})
	if !errors.As(err, &budgetErr) || doer.calls != 1 || client.ExperimentSnapshot().BudgetRejectionCount != 1 {
		t.Fatalf("budget exhaustion did not stop before second HTTP call: err=%v calls=%d snapshot=%+v", err, doer.calls, client.ExperimentSnapshot())
	}
}

func TestDeepSeekDiagnosticCorrectiveRetryAccountingUsesExistingPipeline(t *testing.T) {
	config := deepSeekTestConfig()
	doer := &queuedHTTPDoer{responses: []*http.Response{
		deepSeekTestResponse(http.StatusOK, completedDeepSeekResponse(`{"mapping_status":"PROXY"}`, DeepSeekDiagnosticModel, "fp_a", 100, 0, 100, 20, 0)),
		deepSeekTestResponse(http.StatusOK, completedDeepSeekResponse(unresolvedJSON(), DeepSeekDiagnosticModel, "fp_a", 110, 10, 100, 30, 0)),
	}}
	provider := NewDeepSeekDiagnosticClient(config, doer)
	resolver := testAssetResolver(t)
	input := testInput("Unknown local event", "unknown")
	_, attempts, _, err := analyseEvent(config.Runtime, provider, resolver, "run", DiagnosticManifestVersion, "case-1", "fingerprint", input, []string{"OIL_CATEGORY"})
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 2 || provider.ExperimentSnapshot().RetryCount != 1 || doer.calls != 2 {
		t.Fatalf("corrective retry contract changed: attempts=%d snapshot=%+v calls=%d", len(attempts), provider.ExperimentSnapshot(), doer.calls)
	}
}

func deepSeekConfigValues() map[string]string {
	return map[string]string{
		"JAX_AI_SHADOW_ENABLED": "true", "JAX_AI_PROVIDER": DeepSeekDiagnosticProvider, "JAX_AI_MODEL": DeepSeekDiagnosticModel,
		"JAX_AI_TIMEOUT_SECONDS": "120", "JAX_AI_MAX_EVENTS": "48", DeepSeekDiagnosticAPIKeyEnv: "ds-test-only-do-not-use",
		"JAX_AI_EXPERIMENT_ID": "A1", "JAX_AI_REASONING_EFFORT": "none", DeepSeekDiagnosticThinkingModeEnv: "disabled",
		"JAX_AI_MAX_OUTPUT_TOKENS": "256", "JAX_AI_EXPERIMENT_BUDGET_USD": "0.10",
		"JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS": "0.435", "JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS": "0.003625",
		"JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS": "0.87", OpenAIDiagnosticInferenceAuthEnv: "false",
	}
}
