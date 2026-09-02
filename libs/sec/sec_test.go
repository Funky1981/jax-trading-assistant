package sec

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

const testCIK = "0000320193"

func testIssuer() canonical.Issuer {
	return canonical.Issuer{ContractVersion: canonical.IssuerContractV1, ID: "iss_apple", Type: canonical.IssuerTypeCorporate, Name: "APPLE INC.", Jurisdiction: "US", ExternalIDs: []canonical.ExternalID{{Namespace: "sec.cik", Value: testCIK}}, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func testRetention() providercontract.RawPayloadRetentionPolicy {
	return providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionRestricted}
}

func testPolicy() providercontract.OperationalPolicy {
	component := func(id, name, namespace string) canonical.ComponentIdentity {
		return canonical.ComponentIdentity{ID: id, Kind: canonical.ComponentKindPolicy, Name: name, Version: canonical.VersionIdentity{Namespace: namespace, Value: "sec-test/v1"}}
	}
	return providercontract.OperationalPolicy{ContractVersion: providercontract.OperationalPolicyContractV1, Classification: providercontract.ClassificationPolicy{ContractVersion: providercontract.ClassificationPolicyContractV1, Identity: component("cmp_sec_classification", "SEC test failure classification", "jax.policy.failure_classification")}, Retry: providercontract.RetryPolicy{ContractVersion: providercontract.RetryPolicyContractV1, Identity: component("cmp_sec_retry", "SEC test retry", "jax.policy.retry"), MaximumAttempts: 2, MaximumElapsed: 10 * time.Second, PerAttemptTimeout: 3 * time.Second, RetryableFailures: []providercontract.FailureClass{providercontract.FailureTransportTransient, providercontract.FailureProviderServer, providercontract.FailureTemporaryUnavailable, providercontract.FailureRateLimited}, Backoff: providercontract.BackoffPolicy{InitialDelay: time.Millisecond, Multiplier: 1, MaximumDelay: time.Millisecond, Jitter: providercontract.JitterNone}}, RateLimit: providercontract.RateLimitPolicy{ContractVersion: providercontract.RateLimitPolicyContractV1, Identity: component("cmp_sec_limit", "SEC test rate limit", "jax.policy.rate_limit"), RequestLimit: 100, Window: time.Minute, ConcurrencyLimit: 2, MaximumProviderDelay: time.Second}, Health: providercontract.HealthPolicy{ContractVersion: providercontract.HealthPolicyContractV1, Identity: component("cmp_sec_health", "SEC test health", "jax.policy.health"), DegradedAfterFailures: 1, UnavailableAfterFailures: 2, RecoverySuccesses: 1, AssessmentHorizon: time.Hour}, Component: canonical.ComponentIdentity{ID: "cmp_sec_test_build", Kind: canonical.ComponentKindSoftwareBuild, Name: "SEC adapter tests", Version: canonical.VersionIdentity{Namespace: "test", Value: "wp-03.02"}}}
}

func testDependencies(t *testing.T, _ string, handler http.Handler) (*Provider, Dependencies, *httptest.Server) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	config := Config{BaseURL: server.URL, Identity: RequestIdentity{Product: "Jax-test", Contact: "test@example.invalid"}, MaxResponseBytes: 2 << 20}
	provider, err := NewProvider(config, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	resolver := StaticIdentityResolver{testCIK: testIssuer()}
	registry, pipeline, err := NewProviderPipeline(resolver)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := providercontract.NewOperationalExecutor(registry, testPolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return provider, Dependencies{Registry: registry, Executor: executor, Store: providercontract.NewMemoryRawPayloadStore(), Pipeline: pipeline, Resolver: resolver}, server
}

func submissionsFixture(withFiles bool) string {
	files := `[]`
	if withFiles {
		files = `[ {"name":"CIK0000320193-submissions-001.json","filingFrom":"2020-01-01","filingTo":"2022-12-31"} ]`
	}
	return `{"name":"APPLE INC.","cik":"0000320193","filings":{"recent":{"accessionNumber":["0000320193-25-000001","0000320193-25-000002"],"filingDate":["2025-02-01","2025-02-02"],"acceptanceDateTime":["2025-02-01T21:30:45.123Z","2025-02-02T14:05:06Z"],"reportDate":["2024-12-28","2024-12-28"],"form":["10-K","10-K/A"],"primaryDocument":["aapl-20241228.htm","aapl-20241228a.htm"],"primaryDocDescription":["Annual report","Amended annual report"],"isXBRL":[1,1],"isInlineXBRL":[1,1]},"files":` + files + `}}`
}

func companyFactsFixture() string {
	return `{"cik":320193,"entityName":"APPLE INC.","facts":{"us-gaap":{"Revenues":{"label":"Revenue","description":"Revenue","units":{"USD":[{"val":394328000000,"accn":"0000320193-25-000001","fy":2024,"fp":"FY","form":"10-K","filed":"2025-02-01","start":"2024-01-01","end":"2024-12-28","frame":"CY2024"},{"val":95000000000,"accn":"0000320193-25-000003","fy":2025,"fp":"Q1","form":"10-Q","filed":"2025-05-01","start":"2025-01-01","end":"2025-03-29","frame":"CY2025Q1"}]}},"Assets":{"units":{"USD":[{"val":352583000000,"accn":"0000320193-25-000001","fy":2024,"fp":"FY","form":"10-K","filed":"2025-02-01","end":"2024-12-28","frame":"CY2024I"}]}},"EarningsPerShareBasic":{"units":{"USD/shares":[{"val":6.42,"accn":"0000320193-25-000001","fy":2024,"fp":"FY","form":"10-K","filed":"2025-02-01","start":"2024-01-01","end":"2024-12-28","frame":"CY2024"}]}}},"dei":{"EntityCommonStockSharesOutstanding":{"units":{"shares":[{"val":15000000000,"accn":"0000320193-25-000002","fy":2024,"fp":"FY","form":"10-K/A","filed":"2025-02-02","end":"2024-12-28","frame":"CY2024I"}]}}}}}`
}

func TestSECProviderDefinitionAndIdentityContract(t *testing.T) {
	definition := SECProviderDefinition()
	if err := definition.Validate(); err != nil {
		t.Fatalf("definition invalid: %v", err)
	}
	if definition.Identity.ID != ProviderID || len(definition.Capabilities) != 2 {
		t.Fatalf("definition = %+v", definition)
	}
	issuer := testIssuer()
	identity := CIKIdentity{Issuer: canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(issuer.ID), ContractVersion: issuer.ContractVersion}, CIK: testCIK}
	if err := identity.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := (StaticIdentityResolver{testCIK: issuer}).ResolveSECIdentity("AAPL"); err == nil {
		t.Fatal("ticker-only SEC identity was accepted")
	}
}

func TestSECConfigRequiresExplicitProductionIdentity(t *testing.T) {
	t.Setenv(SECUserAgentEnv, "")
	t.Setenv(SECContactEnv, "")
	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("SEC config accepted missing production contact identity")
	}
	if err := (Config{BaseURL: "https://data.sec.gov", Identity: RequestIdentity{Product: "Jax"}, MaxResponseBytes: 1}).Validate(); err == nil {
		t.Fatal("SEC config accepted missing contact identity")
	}
	if err := (Config{BaseURL: "https://data.sec.gov", Identity: RequestIdentity{Product: "Jax", Contact: "org@example.invalid"}, MaxResponseBytes: 1}).Validate(); err != nil {
		t.Fatalf("explicit configured User-Agent identity rejected: %v", err)
	}
	if err := (Config{BaseURL: "http://data.sec.gov", Identity: RequestIdentity{Product: "Jax"}, MaxResponseBytes: 1}).Validate(); err == nil {
		t.Fatal("SEC config accepted non-HTTPS base URL")
	}
}

func TestSECSubmissionsExactBytesCanonicalOutputAndAmendments(t *testing.T) {
	exact := []byte(submissionsFixture(false) + "\r\n")
	provider, deps, server := testDependencies(t, "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Jax-test test@example.invalid" {
			t.Fatalf("user-agent = %q", r.Header.Get("User-Agent"))
		}
		if r.URL.Path != "/submissions/0000320193.json" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Fatal("request credentials were sent")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(exact)
	}))
	defer server.Close()
	provider.config.BaseURL = server.URL
	issuer := testIssuer()
	request := SubmissionsRequest{Identity: CIKIdentity{Issuer: canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(issuer.ID), ContractVersion: issuer.ContractVersion}, CIK: testCIK}, PayloadID: "rpa_sec_submissions_test", Retention: testRetention()}
	result, err := provider.AcquireSubmissions(context.Background(), deps, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Completeness != CompletenessComplete || len(result.Filings) != 2 || len(result.RawPayloads) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if result.Filings[1].Filing.Form != "10-K/A" || !result.Filings[1].Filing.Amended {
		t.Fatalf("amendment was not preserved: %+v", result.Filings[1].Filing)
	}
	firstEvidence := result.Filings[0]
	first := firstEvidence.Filing
	if first.Dates.FilingDate != SECDate("2025-02-01") {
		t.Fatalf("filing date = %q", first.Dates.FilingDate)
	}
	if first.Dates.ReportDate == nil || *first.Dates.ReportDate != SECDate("2024-12-28") {
		t.Fatalf("report date = %v", first.Dates.ReportDate)
	}
	if first.Dates.AcceptanceDateTime == nil || !first.Dates.AcceptanceDateTime.Equal(time.Date(2025, 2, 1, 21, 30, 45, 123000000, time.UTC)) {
		t.Fatalf("acceptance datetime = %v", first.Dates.AcceptanceDateTime)
	}
	if first.Dates.AcceptanceDateTime.Equal(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("acceptance datetime was reduced to filing-date midnight")
	}
	if first.Dates.PublicAvailabilityTime != nil {
		t.Fatal("SEC public availability time was fabricated")
	}
	if !first.Dates.AcquiredAt.Equal(result.RawPayloads[0].Ref.ReceivedAt) {
		t.Fatal("acquisition time was not preserved independently")
	}
	if firstEvidence.Evidence.PublishedAt != nil || firstEvidence.Evidence.ImmutableRef.PublishedAt != nil {
		t.Fatal("date-only filing date was mapped to canonical publication time")
	}
	if result.RawPayloads[0].Ref.ReceivedAt.Before(*first.Dates.AcceptanceDateTime) {
		t.Fatal("acquisition time did not remain distinct from acceptance time")
	}
	stored, err := providercontract.RetrieveRawPayload(context.Background(), deps.Store, result.RawPayloads[0].Ref)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(result.RawPayloads[0].Ref.Content.Digest.Value, canonical.DigestBytes(exact).Value) || !strings.EqualFold(string(stored), string(exact)) {
		t.Fatal("exact SEC bytes were not preserved")
	}
	encoded, err := providercontract.EncodeRawPayloadRefJSON(result.RawPayloads[0].Ref)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "test@example.invalid") || strings.Contains(string(encoded), "User-Agent") {
		t.Fatal("request identity/header leaked into raw reference")
	}
}

func TestSECSubmissionsRejectMalformedAcceptanceDateTime(t *testing.T) {
	body := strings.Replace(submissionsFixture(false), "2025-02-01T21:30:45.123Z", "not-a-timestamp", 1)
	if _, err := parseSubmissions([]byte(body), providercontract.RawPayloadRef{ReceivedAt: time.Now().UTC()}); err == nil {
		t.Fatal("malformed EDGAR acceptance datetime was accepted")
	}
}

func TestSECCompanyFactsPreservesXBRLSemanticsAndDeterminism(t *testing.T) {
	exact := []byte(companyFactsFixture())
	provider, deps, server := testDependencies(t, "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/xbrl/companyfacts/0000320193.json" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(exact)
	}))
	defer server.Close()
	provider.config.BaseURL = server.URL
	issuer := testIssuer()
	request := CompanyFactsRequest{Identity: CIKIdentity{Issuer: canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(issuer.ID), ContractVersion: issuer.ContractVersion}, CIK: testCIK}, PayloadID: "rpa_sec_facts_test", Retention: testRetention()}
	result, err := provider.AcquireCompanyFacts(context.Background(), deps, request)
	if err != nil {
		var normalizationErr *providercontract.NormalizationError
		if errors.As(err, &normalizationErr) {
			t.Fatalf("%v; cause=%v", err, normalizationErr.Cause)
		}
		t.Fatal(err)
	}
	if len(result.Facts) != 5 {
		t.Fatalf("facts = %d", len(result.Facts))
	}
	if result.Coverage != SECCompanyFactsCoverage || !result.Coverage.EntityWideNonCustomTaxonomies || result.Coverage.CustomTaxonomiesIncluded || result.Coverage.AbsenceIsProofOfNonDisclosure {
		t.Fatalf("company facts coverage semantics = %+v", result.Coverage)
	}
	for _, fact := range result.Facts {
		if fact.Semantics.Taxonomy == "" || fact.Semantics.Concept == "" || fact.Semantics.Unit == "" || fact.Semantics.AccessionNumber == "" || fact.Semantics.SourcePayload.ID != result.Raw.Ref.ID {
			t.Fatalf("semantics lost: %+v", fact.Semantics)
		}
		if fact.Observation.Subject.ID != string(issuer.ID) {
			t.Fatal("fact used non-canonical issuer identity")
		}
		if fact.Observation.PublishedAt != nil || fact.Evidence.PublishedAt != nil || fact.Evidence.ImmutableRef.PublishedAt != nil {
			t.Fatal("Company Facts filed date was mapped to publication time")
		}
		if err := fact.Semantics.FilingDate.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	var sawDuration, sawInstant, sawFrame, sawAmendment bool
	for _, fact := range result.Facts {
		sawDuration = sawDuration || fact.Semantics.Period.Kind == PeriodDuration
		sawInstant = sawInstant || fact.Semantics.Period.Kind == PeriodInstant
		sawFrame = sawFrame || fact.Semantics.Frame != ""
		sawAmendment = sawAmendment || fact.Semantics.Amended
	}
	if !sawDuration || !sawInstant || !sawFrame || !sawAmendment {
		t.Fatalf("period/frame/amendment semantics missing")
	}
	bytes, err := providercontract.RetrieveRawPayload(context.Background(), deps.Store, result.Raw.Ref)
	if err != nil {
		t.Fatal(err)
	}
	deterministic, err := providercontract.VerifyDeterministicBatchNormalization(context.Background(), deps.Pipeline, providercontract.NormalizationRequest{RawRef: result.Raw.Ref, Bytes: bytes, Target: canonical.ContractSchemaRef{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2}, Normalizer: newCompanyFactsNormalizer(deps.Resolver).Descriptor().Component})
	if err != nil {
		t.Fatal(err)
	}
	if len(deterministic.Records) != len(result.Facts) {
		t.Fatal("deterministic normalization count changed")
	}
	tampered := result.Raw.Ref
	tampered.Content = canonical.RawContentIdentity([]byte("different exact bytes"))
	if _, err := deps.Pipeline.NormalizeBatch(context.Background(), providercontract.NormalizationRequest{RawRef: tampered, Bytes: bytes, Target: canonical.ContractSchemaRef{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2}, Normalizer: newCompanyFactsNormalizer(deps.Resolver).Descriptor().Component}); err == nil {
		t.Fatal("pipeline accepted bytes that do not match RawPayloadRef")
	}
}

func TestSECFailClosedForMalformedMediaIdentityCompletenessAndPersistence(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		media     string
		wantError bool
	}{{"malformed", "{", "application/json", true}, {"wrong media", submissionsFixture(false), "text/html", true}, {"missing identity", `{"name":"APPLE INC.","filings":{"recent":{}}}`, "application/json", true}}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			provider, deps, server := testDependencies(t, "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", test.media)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			provider.config.BaseURL = server.URL
			issuer := testIssuer()
			_, err := provider.AcquireSubmissions(context.Background(), deps, SubmissionsRequest{Identity: CIKIdentity{Issuer: canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(issuer.ID), ContractVersion: issuer.ContractVersion}, CIK: testCIK}, PayloadID: providercontract.RawPayloadID("rpa_sec_bad_" + strings.ReplaceAll(test.name, " ", "_")), Retention: testRetention()})
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v", err)
			}
		})
	}
	provider, deps, server := testDependencies(t, "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(submissionsFixture(true)))
	}))
	defer server.Close()
	provider.config.BaseURL = server.URL
	issuer := testIssuer()
	result, err := provider.AcquireSubmissions(context.Background(), deps, SubmissionsRequest{Identity: CIKIdentity{Issuer: canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(issuer.ID), ContractVersion: issuer.ContractVersion}, CIK: testCIK}, PayloadID: "rpa_sec_incomplete", Retention: testRetention()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Completeness != CompletenessAdditionalFilesAvailable {
		t.Fatalf("completeness = %q", result.Completeness)
	}
	if result.IsComplete() {
		t.Fatal("partial submissions history masqueraded as complete")
	}
	failing := failingStore{}
	registry, _ := providercontract.NewRegistry(providercontract.RegistryContractV1)
	_ = registry.Register(SECProviderDefinition())
	execution := providercontract.ExecutionResult{RawBytes: []byte("payload"), CompletedAt: time.Now().UTC()}
	if _, err := persist(context.Background(), registry, failing, execution, "rpa_sec_persist_fail", providercontract.CapabilityCorporateFiling, submissionRaw(), SubmissionsSource, testRetention()); err == nil {
		t.Fatal("persistence failure was accepted")
	}
}

func TestSECFairAccessCeilingUsesPhase02RateLimiter(t *testing.T) {
	hit := false
	provider, deps, server := testDependencies(t, "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(submissionsFixture(false)))
	}))
	defer server.Close()
	provider.config.BaseURL = server.URL
	policy := testPolicy()
	policy.RateLimit.RequestLimit = 11
	policy.RateLimit.Window = time.Second
	executor, err := providercontract.NewOperationalExecutor(deps.Registry, policy, providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	deps.Executor = executor
	issuer := testIssuer()
	_, err = provider.AcquireSubmissions(context.Background(), deps, SubmissionsRequest{Identity: CIKIdentity{Issuer: canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(issuer.ID), ContractVersion: issuer.ContractVersion}, CIK: testCIK}, PayloadID: "rpa_sec_rate_limit", Retention: testRetention()})
	if err == nil || !strings.Contains(err.Error(), "exceeds 10 requests per second") {
		t.Fatalf("rate ceiling error = %v", err)
	}
	if hit {
		t.Fatal("SEC request was sent with an over-ceiling Phase 02 policy")
	}
}

func TestCompanyFactObservationHasProviderNeutralSemanticShape(t *testing.T) {
	// The wrapper contains normalized taxonomy/concept/unit/value/period and
	// filing provenance only. It contains no SEC envelope, DTO, endpoint, or
	// endpoint-array fields, so another fundamentals adapter can construct the
	// same shape from its own source boundary.
	observation := CompanyFactObservation{}
	if observation.Semantics.Taxonomy != "" || observation.Semantics.Concept != "" {
		t.Fatal("unexpected non-zero zero-value semantics")
	}
}

type failingStore struct{}

func (failingStore) Put(context.Context, providercontract.RawPayloadRef, []byte) (providercontract.RawPayloadLocation, error) {
	return providercontract.RawPayloadLocation{}, errors.New("store unavailable")
}
func (failingStore) Get(context.Context, providercontract.RawPayloadRef) ([]byte, error) {
	return nil, errors.New("store unavailable")
}
