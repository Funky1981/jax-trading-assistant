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

func TestInitialPromptContainsOnlyReceiptTimeInput(t *testing.T) {
	input := EventInput{Title: "Headline", Summary: "Summary", Source: "Wire", PublicationTimestamp: time.Unix(1, 0).UTC(), ReceiptTimestamp: time.Unix(2, 0).UTC(), EventCategory: "macro", Entities: []string{"Acme"}, ReceiptEvidence: []string{"official release"}}
	request, err := InitialRequest(input)
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
	for _, forbidden := range []string{"XLE", "TLT", "SPY", "realised return", "realized return", "future candle", "deterministic mapping", "deterministic answer", "Ryanair", "Abu Dhabi"} {
		if strings.Contains(systemPrompt, forbidden) {
			t.Fatalf("system prompt contains answer or outcome leakage %q", forbidden)
		}
	}
	if regexp.MustCompile(`(?i)"ticker"\s*:\s*"[A-Z0-9.-]+"`).MatchString(systemPrompt) {
		t.Fatal("system prompt contains a fixed ticker example")
	}
	if strings.Count(systemPrompt, "\n1. ")+strings.Count(systemPrompt, "\n2. ")+strings.Count(systemPrompt, "\n3. ") != 3 || strings.Contains(systemPrompt, "\n4. ") {
		t.Fatal("system prompt must contain exactly three generic examples")
	}
}

func TestParseAndValidateStrictSchema(t *testing.T) {
	raw := `{"market_relevance":"HIGH","mapping_status":"DIRECT","ticker":"NVDA","mapping_confidence":"HIGH","expected_horizon":"ONE_DAY","likely_direction":"POSITIVE","catalyst_type":"earnings","reason":"A concrete company catalyst may move the named equity.","missing_evidence":[]}`
	result, errs := ParseAndValidate(raw)
	if len(errs) > 0 || result == nil {
		t.Fatalf("unexpected errors: %v", errs)
	}
	_, errs = ParseAndValidate(strings.TrimSuffix(raw, "}") + `,"extra":true}`)
	if !hasError(errs, "unknown field") {
		t.Fatalf("expected unknown field error: %v", errs)
	}
}

func TestSemanticMappingValidation(t *testing.T) {
	cases := []struct{ name, raw, want string }{
		{"unresolved ticker", `{"market_relevance":"LOW","mapping_status":"UNRESOLVED","ticker":"SPY","mapping_confidence":"LOW","expected_horizon":"UNCLEAR","likely_direction":"UNCLEAR","catalyst_type":"unclear","reason":"The available event record does not identify an asset.","missing_evidence":[]}`, "requires an empty ticker"},
		{"direct empty ticker", `{"market_relevance":"HIGH","mapping_status":"DIRECT","ticker":"","mapping_confidence":"MEDIUM","expected_horizon":"ONE_DAY","likely_direction":"NEUTRAL","catalyst_type":"company event","reason":"The event directly concerns a specific listed market asset.","missing_evidence":[]}`, "require a non-empty ticker"},
		{"proxy empty ticker", `{"market_relevance":"HIGH","mapping_status":"PROXY","ticker":"","mapping_confidence":"MEDIUM","expected_horizon":"ONE_DAY","likely_direction":"NEUTRAL","catalyst_type":"macro","reason":"The event has a defensible sector-level market relationship.","missing_evidence":[]}`, "require a non-empty ticker"},
		{"ticker format", `{"market_relevance":"HIGH","mapping_status":"PROXY","ticker":"bad ticker","mapping_confidence":"MEDIUM","expected_horizon":"ONE_DAY","likely_direction":"NEUTRAL","catalyst_type":"macro","reason":"The event has a conservative sector-level market proxy.","missing_evidence":[]}`, "invalid format"},
		{"categorical confidence", `{"market_relevance":"HIGH","mapping_status":"PROXY","ticker":"SPY","mapping_confidence":"70","expected_horizon":"ONE_DAY","likely_direction":"NEUTRAL","catalyst_type":"macro","reason":"The event has a conservative sector-level market proxy.","missing_evidence":[]}`, "mapping_confidence must be"},
		{"null ticker", `{"market_relevance":"LOW","mapping_status":"UNRESOLVED","ticker":null,"mapping_confidence":"LOW","expected_horizon":"UNCLEAR","likely_direction":"UNCLEAR","catalyst_type":"unclear","reason":"The available event record does not identify an asset.","missing_evidence":[]}`, "not null"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := ParseAndValidate(tt.raw)
			if !hasError(errs, tt.want) {
				t.Fatalf("errors %v do not contain %q", errs, tt.want)
			}
		})
	}
}

func TestEmptyTickerAcceptedOnlyWithUnresolved(t *testing.T) {
	valid := `{"market_relevance":"LOW","mapping_status":"UNRESOLVED","ticker":"","mapping_confidence":"LOW","expected_horizon":"UNCLEAR","likely_direction":"UNCLEAR","catalyst_type":"unclear","reason":"The available record cannot support one principal asset mapping.","missing_evidence":["A named asset or principal exposure"]}`
	if result, errs := ParseAndValidate(valid); result == nil || len(errs) != 0 {
		t.Fatalf("UNRESOLVED empty ticker should be accepted: %v", errs)
	}
	for _, status := range []string{"DIRECT", "PROXY"} {
		raw := strings.Replace(valid, `"mapping_status":"UNRESOLVED"`, `"mapping_status":"`+status+`"`, 1)
		if _, errs := ParseAndValidate(raw); !hasError(errs, "non-empty ticker") {
			t.Fatalf("%s empty ticker should be rejected: %v", status, errs)
		}
	}
}

func TestOutputSchemaHasFlatNonNullableContract(t *testing.T) {
	schema := OutputSchema()
	properties := schema["properties"].(map[string]any)
	if properties["ticker"].(map[string]any)["type"] != "string" {
		t.Fatal("ticker must have only string type")
	}
	for _, removed := range []string{"resolved_asset", "asset_mapping_type", "confidence"} {
		if _, ok := properties[removed]; ok {
			t.Fatalf("legacy property %s remains in v2 schema", removed)
		}
	}
}

func TestLegacySchemaResultRemainsReadableWithoutConfidenceCoercion(t *testing.T) {
	raw := []byte(`{"market_relevance":"HIGH","resolved_asset":"NVDA","asset_mapping_type":"direct","expected_horizon":"1d","likely_direction":"positive","confidence":85,"catalyst_type":"earnings","reason":"A concrete company catalyst may move the named equity.","missing_evidence":[]}`)
	decoded, err := DecodePersistedResult(LegacySchemaVersion, raw)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Legacy == nil || decoded.Current != nil || decoded.Legacy.Confidence != 85 {
		t.Fatalf("legacy numeric confidence was not preserved: %#v", decoded)
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
	provider := &sequenceProvider{responses: []ProviderResponse{{Content: `{"bad":true}`}, {Content: `{"market_relevance":"LOW","mapping_status":"UNRESOLVED","ticker":"","mapping_confidence":"LOW","expected_horizon":"UNCLEAR","likely_direction":"UNCLEAR","catalyst_type":"unclear","reason":"The available record does not support a conservative asset mapping.","missing_evidence":[]}`}}}
	repo := &memoryRepository{counts: SafetyCounts{Approvals: 2, PaperTickets: 1}}
	event := BenchmarkEvent{ID: "00000000-0000-0000-0000-000000000001", Input: EventInput{Title: "x", Entities: []string{}, ReceiptEvidence: []string{}}, InputFingerprint: "fingerprint", Decision: "NO_TRADE", Mapping: evidencequality.Mapping{}, Outcome1H: .01, Outcome1D: .02}
	manifest, err := NewManifest([]BenchmarkEvent{event}, 1)
	if err != nil {
		t.Fatal(err)
	}
	runner := Runner{Config: Config{Provider: "ollama", Model: "test", MaxEvents: 1}, Provider: provider, Repository: repo, OutputRoot: t.TempDir()}
	report, paths, err := runner.Run(manifest, []BenchmarkEvent{event})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 || len(repo.attempts) != 2 || report.RetryCount != 1 {
		t.Fatalf("retry was not bounded to one: calls=%d attempts=%d", provider.calls, len(repo.attempts))
	}
	if len(provider.requests) != 2 {
		t.Fatalf("captured requests=%d, want 2", len(provider.requests))
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
}

func TestRunnerDoesNotLoopAfterSecondRejection(t *testing.T) {
	provider := &sequenceProvider{responses: []ProviderResponse{{Content: `{}`}, {Content: `{}`}, {Content: `{}`}}}
	repo := &memoryRepository{}
	event := BenchmarkEvent{ID: "00000000-0000-0000-0000-000000000001", Input: EventInput{Entities: []string{}, ReceiptEvidence: []string{}}, InputFingerprint: "fp", Decision: "WATCH", Mapping: evidencequality.Mapping{Mapped: true, Direct: true, Symbol: "SPY"}}
	manifest, _ := NewManifest([]BenchmarkEvent{event}, 1)
	_, _, err := Runner{Config: Config{Provider: "ollama", Model: "test", MaxEvents: 1}, Provider: provider, Repository: repo, OutputRoot: t.TempDir()}.Run(manifest, []BenchmarkEvent{event})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 {
		t.Fatalf("calls=%d, want 2", provider.calls)
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
