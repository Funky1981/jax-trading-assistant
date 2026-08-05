package aishadow

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
	"jax-trading-assistant/internal/modules/evidencequality"
)

func validEnvironment() map[string]string {
	return map[string]string{
		"JAX_AI_SHADOW_ENABLED": "true", "JAX_AI_PROVIDER": "ollama",
		"JAX_AI_MODEL": "qwen3.5:9b", "JAX_AI_BASE_URL": "http://localhost:11434",
		"JAX_AI_TIMEOUT_SECONDS": "120", "JAX_AI_TEMPERATURE": "0",
		"JAX_AI_SEED": "20260803", "JAX_AI_MAX_EVENTS": "3",
	}
}

func lookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
}

func testAssetResolver(t *testing.T) assetresolution.Resolver {
	t.Helper()
	rules, err := assetresolution.LoadRuleset("../../../config/event-asset-resolution-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	return assetresolution.Resolver{Rules: rules}
}

func testInput(title, category string) EventInput {
	return EventInput{
		Title: title, Summary: "Receipt-time summary", Source: "Wire",
		PublicationTimestamp: time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC),
		ReceiptTimestamp:     time.Date(2026, 7, 29, 12, 1, 0, 0, time.UTC),
		EventCategory:        category, Entities: []string{}, ReceiptEvidence: []string{},
	}
}

func unresolvedJSON() string {
	return `{"market_relevance":"LOW","mapping_status":"UNRESOLVED","direct_ticker":"","proxy_exposure":"NONE","mapping_confidence":"LOW","expected_horizon":"UNCLEAR","likely_direction":"UNCLEAR","catalyst_type":"unclear","reason":"The available record does not support a bounded asset mapping.","missing_evidence":[]}`
}

func proxyJSON(exposure string) string {
	return `{"market_relevance":"HIGH","mapping_status":"PROXY","direct_ticker":"","proxy_exposure":"` + exposure + `","mapping_confidence":"HIGH","expected_horizon":"ONE_DAY","likely_direction":"POSITIVE","catalyst_type":"energy supply","reason":"The event has a clear principal energy-market exposure.","missing_evidence":[]}`
}

func directJSON(ticker string) string {
	return `{"market_relevance":"HIGH","mapping_status":"DIRECT","direct_ticker":"` + ticker + `","proxy_exposure":"NONE","mapping_confidence":"HIGH","expected_horizon":"ONE_DAY","likely_direction":"POSITIVE","catalyst_type":"earnings","reason":"The named listed issuer is directly affected by the catalyst.","missing_evidence":[]}`
}

func TestLoadConfigFailsClosed(t *testing.T) {
	for key := range validEnvironment() {
		values := validEnvironment()
		delete(values, key)
		if _, err := LoadConfig(lookup(values)); err == nil {
			t.Fatalf("missing %s should fail", key)
		}
	}
	unsafe := validEnvironment()
	unsafe["JAX_AI_BASE_URL"] = "http://public.example:11434"
	if _, err := LoadConfig(lookup(unsafe)); err == nil {
		t.Fatal("public Ollama URL should fail")
	}
	disabled := validEnvironment()
	disabled["JAX_AI_SHADOW_ENABLED"] = "false"
	if _, err := LoadConfig(lookup(disabled)); err == nil {
		t.Fatal("disabled shadow configuration should fail")
	}
}

func TestInitialPromptContainsOnlyReceiptTimeInputAndNoTickerMapping(t *testing.T) {
	resolver := testAssetResolver(t)
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		t.Fatal(err)
	}
	input := testInput("Headline", "macro")
	input.Entities = []string{"Acme"}
	input.ReceiptEvidence = []string{"official release"}
	request, err := InitialRequest(input, exposures)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(request.User), &decoded); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"title": true, "summary": true, "source": true, "publication_timestamp": true, "receipt_timestamp": true, "event_category": true, "entities": true, "receipt_evidence": true}
	for key := range decoded {
		if !allowed[key] {
			t.Fatalf("leaked model-input field %s", key)
		}
	}
	for _, forbidden := range []string{"decision", "watch", "no_trade", "outcome", "return", "resolved_asset", "candidate", "subject"} {
		if _, ok := decoded[forbidden]; ok {
			t.Fatalf("leaked forbidden field %s", forbidden)
		}
	}
	promptAndSchema := systemPrompt + compactJSON(request.Schema)
	for _, forbidden := range []string{"XLE", "TLT", "SPY", "QQQ", "SOXX", "GLD", "realised return", "realized return", "future candle", "deterministic answer", "Ryanair", "Abu Dhabi"} {
		if strings.Contains(promptAndSchema, forbidden) {
			t.Fatalf("prompt or schema contains ticker mapping or answer leakage %q", forbidden)
		}
	}
	if regexp.MustCompile(`(?i)"direct_ticker"\s*:\s*"[A-Z0-9.-]+"`).MatchString(systemPrompt) {
		t.Fatal("system prompt contains a fixed ticker example")
	}
	if strings.Count(systemPrompt, "\n1. ")+strings.Count(systemPrompt, "\n2. ")+strings.Count(systemPrompt, "\n3. ") != 3 || strings.Contains(systemPrompt, "\n4. ") {
		t.Fatal("system prompt must contain exactly three generic examples")
	}
}

func TestParseAndValidateDirectTickerUsesReceiptTimePolicy(t *testing.T) {
	resolver := testAssetResolver(t)
	input := testInput("Apple reports quarterly earnings", "company")
	result, resolution, errs := ParseAndValidate(directJSON("AAPL"), input, resolver)
	if len(errs) > 0 || result == nil || resolution == nil {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if resolution.ResolvedTicker != "AAPL" || resolution.MatchedRule != "Apple Inc." || resolution.PolicyVersion != "event-asset-resolution-v1" {
		t.Fatalf("direct resolution provenance=%+v", resolution)
	}
	_, _, errs = ParseAndValidate(directJSON("MSFT"), input, resolver)
	if !hasError(errs, "not independently verified") {
		t.Fatalf("mismatched direct ticker was accepted: %v", errs)
	}
	_, _, errs = ParseAndValidate(strings.TrimSuffix(directJSON("AAPL"), "}")+`,"extra":true}`, input, resolver)
	if !hasError(errs, "unknown field") {
		t.Fatalf("expected unknown field error: %v", errs)
	}
}

func TestProxyExposureResolvesThroughVersionedPolicy(t *testing.T) {
	resolver := testAssetResolver(t)
	result, resolution, errs := ParseAndValidate(proxyJSON("OIL_CATEGORY"), testInput("Oil supply disruption", "energy_oil"), resolver)
	if len(errs) != 0 || result == nil || resolution == nil {
		t.Fatalf("unexpected validation errors: %v", errs)
	}
	if result.DirectTicker != "" || resolution.ResolvedTicker != "XLE" || resolution.MatchedRule != "oil_category" || resolution.PolicyVersion != "event-asset-resolution-v1" {
		t.Fatalf("proxy result=%+v resolution=%+v", result, resolution)
	}
	_, _, errs = ParseAndValidate(proxyJSON("UNSUPPORTED"), testInput("Broad event", "unknown"), resolver)
	if !hasError(errs, "not allowlisted") {
		t.Fatalf("unsupported exposure was accepted: %v", errs)
	}
}

func TestUnknownAssetRemainsValidAsUnresolved(t *testing.T) {
	result, resolution, errs := ParseAndValidate(unresolvedJSON(), testInput("Unknown local event", "unknown"), testAssetResolver(t))
	if len(errs) != 0 || result == nil || resolution == nil {
		t.Fatalf("unknown asset should remain a valid unresolved result: %v", errs)
	}
	if result.MappingStatus != "UNRESOLVED" || result.DirectTicker != "" || result.ProxyExposure != "NONE" || resolution.Status != "unresolved" || resolution.ResolvedTicker != "" || resolution.PolicyVersion != "event-asset-resolution-v1" {
		t.Fatalf("unexpected unresolved model/policy result: model=%+v policy=%+v", result, resolution)
	}
}

func TestSemanticCrossFieldValidation(t *testing.T) {
	resolver := testAssetResolver(t)
	input := testInput("Apple reports quarterly earnings", "company")
	cases := []struct{ name, raw, want string }{
		{"unresolved direct ticker", strings.Replace(unresolvedJSON(), `"direct_ticker":""`, `"direct_ticker":"AAPL"`, 1), "requires an empty direct_ticker"},
		{"unresolved exposure", strings.Replace(unresolvedJSON(), `"proxy_exposure":"NONE"`, `"proxy_exposure":"OIL_CATEGORY"`, 1), "requires proxy_exposure NONE"},
		{"unresolved confidence", strings.Replace(unresolvedJSON(), `"mapping_confidence":"LOW"`, `"mapping_confidence":"HIGH"`, 1), "requires LOW"},
		{"direct empty ticker", strings.Replace(directJSON("AAPL"), `"direct_ticker":"AAPL"`, `"direct_ticker":""`, 1), "requires a non-empty direct_ticker"},
		{"direct proxy exposure", strings.Replace(directJSON("AAPL"), `"proxy_exposure":"NONE"`, `"proxy_exposure":"OIL_CATEGORY"`, 1), "requires proxy_exposure NONE"},
		{"direct low confidence", strings.Replace(directJSON("AAPL"), `"mapping_confidence":"HIGH"`, `"mapping_confidence":"LOW"`, 1), "requires HIGH or MEDIUM"},
		{"proxy direct ticker", strings.Replace(proxyJSON("OIL_CATEGORY"), `"direct_ticker":""`, `"direct_ticker":"AAPL"`, 1), "requires an empty direct_ticker"},
		{"proxy none", strings.Replace(proxyJSON("OIL_CATEGORY"), `"proxy_exposure":"OIL_CATEGORY"`, `"proxy_exposure":"NONE"`, 1), "requires a bounded proxy_exposure"},
		{"ticker format", strings.Replace(directJSON("AAPL"), `"direct_ticker":"AAPL"`, `"direct_ticker":"bad ticker"`, 1), "invalid format"},
		{"null ticker", strings.Replace(unresolvedJSON(), `"direct_ticker":""`, `"direct_ticker":null`, 1), "not null"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, _, errs := ParseAndValidate(tt.raw, input, resolver)
			if !hasError(errs, tt.want) {
				t.Fatalf("errors %v do not contain %q", errs, tt.want)
			}
		})
	}
}

func TestOutputSchemaIsBoundedAndHasNoProxyTicker(t *testing.T) {
	resolver := testAssetResolver(t)
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		t.Fatal(err)
	}
	schema := OutputSchema(exposures)
	properties := schema["properties"].(map[string]any)
	if properties["direct_ticker"].(map[string]any)["type"] != "string" {
		t.Fatal("direct_ticker must have only string type")
	}
	gotExposures := properties["proxy_exposure"].(map[string]any)["enum"].([]string)
	if !reflect.DeepEqual(gotExposures, append([]string{"NONE"}, exposures...)) {
		t.Fatalf("proxy exposure enum=%v", gotExposures)
	}
	for _, removed := range []string{"ticker", "resolved_asset", "asset_mapping_type", "confidence"} {
		if _, ok := properties[removed]; ok {
			t.Fatalf("historic property %s remains in v3 schema", removed)
		}
	}
}

func TestPersistedDecoderKeepsV1V2AndV3Distinct(t *testing.T) {
	legacyRaw := []byte(`{"market_relevance":"HIGH","resolved_asset":"NVDA","asset_mapping_type":"direct","expected_horizon":"1d","likely_direction":"positive","confidence":85,"catalyst_type":"earnings","reason":"A concrete company catalyst may move the named equity.","missing_evidence":[]}`)
	legacy, err := DecodePersistedResult(LegacySchemaVersion, legacyRaw)
	if err != nil || legacy.Legacy == nil || legacy.V2 != nil || legacy.Current != nil || legacy.Legacy.Confidence != 85 {
		t.Fatalf("legacy numeric confidence was not preserved: %#v err=%v", legacy, err)
	}
	v2Raw := []byte(`{"market_relevance":"HIGH","mapping_status":"PROXY","ticker":"XLE","mapping_confidence":"HIGH","expected_horizon":"ONE_DAY","likely_direction":"POSITIVE","catalyst_type":"energy","reason":"A prior free-form proxy ticker result remains historical.","missing_evidence":[]}`)
	v2, err := DecodePersistedResult(V2SchemaVersion, v2Raw)
	if err != nil || v2.V2 == nil || v2.Current != nil || v2.V2.Ticker != "XLE" {
		t.Fatalf("v2 free-form result was reinterpreted: %#v err=%v", v2, err)
	}
	v3Raw := []byte(`{"model_output":` + proxyJSON("OIL_CATEGORY") + `,"deterministic_resolution":{"status":"resolved","policy_version":"event-asset-resolution-v1","matched_rule":"oil_category","resolved_ticker":"XLE","mapping_type":"event_category_proxy","relationship":"proxy","reason":"policy mapping"}}`)
	v3, err := DecodePersistedResult(SchemaVersion, v3Raw)
	if err != nil || v3.Current == nil || v3.V2 != nil || v3.Current.DeterministicResolution.MatchedRule != "oil_category" {
		t.Fatalf("v3 envelope was not decoded: %#v err=%v", v3, err)
	}
	if _, err := DecodePersistedResult(SchemaVersion, v2Raw); err == nil {
		t.Fatal("v2 flat result must not be interpreted as v3")
	}
	if _, err := DecodePersistedResult(V2SchemaVersion, v3Raw); err == nil {
		t.Fatal("v3 envelope must not be interpreted as v2")
	}
}

func TestPersistedV3PayloadIncludesSeparatePolicyProvenance(t *testing.T) {
	model := &StructuredResult{MarketRelevance: "HIGH", MappingStatus: "PROXY", ProxyExposure: "OIL_CATEGORY", MappingConfidence: "HIGH", ExpectedHorizon: "ONE_DAY", LikelyDirection: "POSITIVE", CatalystType: "energy", Reason: "The event has one bounded energy exposure.", MissingEvidence: []string{}}
	resolution := &PolicyResolution{Status: "resolved", PolicyVersion: "event-asset-resolution-v1", MatchedRule: "oil_category", ResolvedTicker: "XLE", MappingType: "event_category_proxy", Relationship: "proxy", Reason: "policy mapping"}
	raw, err := persistedResultJSON(EventResult{Attempt: Attempt{SchemaVersion: SchemaVersion}, Parsed: model, Resolution: resolution})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodePersistedResult(SchemaVersion, []byte(raw.(string)))
	if err != nil || decoded.Current == nil {
		t.Fatalf("decode persisted payload: %v %#v", err, decoded)
	}
	if decoded.Current.ModelOutput.DirectTicker != "" || decoded.Current.DeterministicResolution.ResolvedTicker != "XLE" {
		t.Fatalf("model and policy fields were not separated: %#v", decoded.Current)
	}
}

func TestManifestDeterminismAndFingerprintValidation(t *testing.T) {
	events := []BenchmarkEvent{
		{ID: "00000000-0000-0000-0000-000000000001", InputFingerprint: "one"},
		{ID: "00000000-0000-0000-0000-000000000002", InputFingerprint: "two"},
	}
	one, err := NewManifest(events, 2)
	if err != nil {
		t.Fatal(err)
	}
	two, err := NewManifest(events, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(one, two) {
		t.Fatal("manifest is not deterministic")
	}
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := WriteManifest(path, one); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded.Events[0].InputFingerprint = "changed"
	if _, err := ResolveManifest(loaded, events, 2); err == nil {
		t.Fatal("changed input fingerprint should fail")
	}
}

func TestRunnerAllowsAtMostOneCorrectiveRetryAndPreservesSafety(t *testing.T) {
	provider := &sequenceProvider{responses: []ProviderResponse{{Content: `{"bad":true}`}, {Content: proxyJSON("OIL_CATEGORY")}}}
	repo := &memoryRepository{counts: SafetyCounts{Approvals: 2, PaperTickets: 1}}
	event := BenchmarkEvent{ID: "00000000-0000-0000-0000-000000000001", Input: testInput("Oil supply disruption", "energy_oil"), InputFingerprint: "fingerprint", Decision: "WATCH", Mapping: evidencequality.Mapping{Mapped: true, Symbol: "XLE"}, Outcome1H: .01, Outcome1D: .02}
	manifest, err := NewManifest([]BenchmarkEvent{event}, 1)
	if err != nil {
		t.Fatal(err)
	}
	runner := Runner{Config: Config{Provider: "ollama", Model: "test", MaxEvents: 1}, Provider: provider, Repository: repo, OutputRoot: t.TempDir(), AssetResolver: testAssetResolver(t)}
	report, paths, err := runner.Run(manifest, []BenchmarkEvent{event})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 || len(repo.attempts) != 2 || report.RetryCount != 1 {
		t.Fatalf("retry was not bounded to one: calls=%d attempts=%d", provider.calls, len(repo.attempts))
	}
	if len(repo.results) != 1 || repo.results[0].Resolution == nil || repo.results[0].Resolution.ResolvedTicker != "XLE" {
		t.Fatalf("deterministic resolution was not captured: %#v", repo.results)
	}
	if report.ExactMappingAgreement != 1 || report.FabricatedInvalidTickers != 0 {
		t.Fatalf("report did not compare deterministic resolution correctly: %+v", report)
	}
	var correction map[string]json.RawMessage
	if err := json.Unmarshal([]byte(provider.requests[1].User), &correction); err != nil {
		t.Fatal(err)
	}
	if len(correction) != 2 || correction["validation_errors"] == nil || correction["previous_structured_response"] == nil {
		t.Fatalf("correction payload contains more than previous response and concrete errors: %s", provider.requests[1].User)
	}
	if repo.before != repo.after {
		t.Fatal("safety counts changed")
	}
	for _, path := range []string{paths.Markdown, paths.JSON, paths.CSV, paths.Manifest} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing report artifact %s: %v", path, err)
		}
	}
	jsonReport, err := os.ReadFile(paths.JSON)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonReport), `"model_output":`) || !strings.Contains(string(jsonReport), `"proxy_exposure": "OIL_CATEGORY"`) || !strings.Contains(string(jsonReport), `"deterministic_resolution":`) || !strings.Contains(string(jsonReport), `"policy_version": "event-asset-resolution-v1"`) {
		t.Fatalf("JSON report does not separate model exposure and policy provenance: %s", jsonReport)
	}
	csvReport, err := os.ReadFile(paths.CSV)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(csvReport), "proxy_exposure") || !strings.Contains(string(csvReport), "jax_resolved_ticker") || !strings.Contains(string(csvReport), "matched_rule") {
		t.Fatalf("CSV report is missing v3 provenance columns: %s", csvReport)
	}
	markdownReport, err := os.ReadFile(paths.Markdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdownReport), "model `") || !strings.Contains(string(markdownReport), "Jax deterministic `") {
		t.Fatalf("markdown report does not distinguish model and Jax behavior: %s", markdownReport)
	}
}

func TestRunnerDoesNotLoopAfterSecondRejection(t *testing.T) {
	provider := &sequenceProvider{responses: []ProviderResponse{{Content: `{}`}, {Content: `{}`}, {Content: `{}`}}}
	repo := &memoryRepository{}
	event := BenchmarkEvent{ID: "00000000-0000-0000-0000-000000000001", Input: testInput("Unknown event", "unknown"), InputFingerprint: "fp", Decision: "NO_TRADE", Mapping: evidencequality.Mapping{}}
	manifest, _ := NewManifest([]BenchmarkEvent{event}, 1)
	_, _, err := Runner{Config: Config{Provider: "ollama", Model: "test", MaxEvents: 1}, Provider: provider, Repository: repo, OutputRoot: t.TempDir(), AssetResolver: testAssetResolver(t)}.Run(manifest, []BenchmarkEvent{event})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 {
		t.Fatalf("calls=%d, want 2", provider.calls)
	}
}

func TestRunnerFailsBeforeProviderWhenPolicyIsMissing(t *testing.T) {
	provider := &sequenceProvider{responses: []ProviderResponse{{Content: unresolvedJSON()}}}
	event := BenchmarkEvent{ID: "00000000-0000-0000-0000-000000000001", Input: testInput("Unknown", "unknown"), InputFingerprint: "fp"}
	manifest, _ := NewManifest([]BenchmarkEvent{event}, 1)
	_, _, err := Runner{Config: Config{Provider: "ollama", Model: "test", MaxEvents: 1}, Provider: provider, Repository: &memoryRepository{}, OutputRoot: t.TempDir()}.Run(manifest, []BenchmarkEvent{event})
	if err == nil || provider.calls != 0 {
		t.Fatalf("missing policy should fail before inference: err=%v calls=%d", err, provider.calls)
	}
}

func TestPersistenceNormalizesNilValidationErrors(t *testing.T) {
	values := nonNilStrings(nil)
	if values == nil || len(values) != 0 {
		t.Fatalf("nil validation errors were not normalized: %#v", values)
	}
}

func hasError(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

type sequenceProvider struct {
	responses []ProviderResponse
	requests  []ProviderRequest
	calls     int
}

func (p *sequenceProvider) Complete(request ProviderRequest) (ProviderResponse, error) {
	if p.calls >= len(p.responses) {
		return ProviderResponse{}, errors.New("unexpected call")
	}
	p.requests = append(p.requests, request)
	response := p.responses[p.calls]
	p.calls++
	return response, nil
}

type memoryRepository struct {
	counts, before, after SafetyCounts
	attempts              []Attempt
	results               []EventResult
}

func (r *memoryRepository) SafetyCounts() (SafetyCounts, error) {
	if r.before == (SafetyCounts{}) {
		r.before = r.counts
	}
	r.after = r.counts
	return r.counts, nil
}
func (r *memoryRepository) StartRun(RunRecord) error { return nil }
func (r *memoryRepository) SaveAttempt(value Attempt) error {
	r.attempts = append(r.attempts, value)
	return nil
}
func (r *memoryRepository) SaveResult(value EventResult) error {
	r.results = append(r.results, value)
	return nil
}
func (r *memoryRepository) FinishRun(FinishRecord) error { return nil }
