package treasury

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/macroevidence"
)

func treasuryRetention() providercontract.RawPayloadRetentionPolicy {
	return providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionNotAuthorized}
}

func treasuryPolicy() providercontract.OperationalPolicy {
	identity := func(id, name, namespace string) canonical.ComponentIdentity {
		return canonical.ComponentIdentity{ID: id, Kind: canonical.ComponentKindPolicy, Name: name, Version: canonical.VersionIdentity{Namespace: namespace, Value: "fixture/v1"}}
	}
	return providercontract.OperationalPolicy{
		ContractVersion: providercontract.OperationalPolicyContractV1,
		Classification:  providercontract.ClassificationPolicy{ContractVersion: providercontract.ClassificationPolicyContractV1, Identity: identity("cmp_treasury_class", "Treasury fixture classification", "jax.policy.failure_classification")},
		Retry:           providercontract.RetryPolicy{ContractVersion: providercontract.RetryPolicyContractV1, Identity: identity("cmp_treasury_retry", "Treasury fixture retry", "jax.policy.retry"), MaximumAttempts: 1, MaximumElapsed: time.Minute, PerAttemptTimeout: 10 * time.Second, RetryableFailures: []providercontract.FailureClass{providercontract.FailureTransportTransient, providercontract.FailureProviderServer, providercontract.FailureTemporaryUnavailable, providercontract.FailureRateLimited, providercontract.FailureAttemptDeadline}, Backoff: providercontract.BackoffPolicy{InitialDelay: time.Millisecond, Multiplier: 2, MaximumDelay: time.Second, Jitter: providercontract.JitterNone}},
		RateLimit:       providercontract.RateLimitPolicy{ContractVersion: providercontract.RateLimitPolicyContractV1, Identity: identity("cmp_treasury_rate", "Treasury fixture rate limit", "jax.policy.rate_limit"), RequestLimit: 100, Window: time.Minute, ConcurrencyLimit: 4, MaximumProviderDelay: time.Second},
		Health:          providercontract.HealthPolicy{ContractVersion: providercontract.HealthPolicyContractV1, Identity: identity("cmp_treasury_health", "Treasury fixture health", "jax.policy.health"), DegradedAfterFailures: 1, UnavailableAfterFailures: 3, RecoverySuccesses: 1, AssessmentHorizon: time.Hour},
		Component:       canonical.ComponentIdentity{ID: "cmp_treasury_build", Kind: canonical.ComponentKindSoftwareBuild, Name: "Treasury fixture executor", Version: canonical.VersionIdentity{Namespace: "git.commit", Value: "treasury-fixture"}},
	}
}

func treasuryFixtureDependencies(t *testing.T, body string) (*Provider, Dependencies) {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	provider, err := NewProvider(Config{BaseURL: server.URL, MaxResponseBytes: 1 << 20}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := RegisterProvider(registry); err != nil {
		t.Fatal(err)
	}
	executor, err := providercontract.NewOperationalExecutor(registry, treasuryPolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return provider, Dependencies{Registry: registry, Executor: executor, Store: providercontract.NewMemoryRawPayloadStore()}
}

func treasuryFixture() string {
	return `<?xml version="1.0" encoding="utf-8"?><feed xmlns="http://www.w3.org/2005/Atom"><updated>2026-09-04T02:01:01Z</updated><entry><content><properties xmlns="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata"><NEW_DATE>2026-01-02T00:00:00</NEW_DATE><BC_1MONTH>3.72</BC_1MONTH><BC_1_5MONTH>3.71</BC_1_5MONTH><BC_2MONTH>3.66</BC_2MONTH><BC_3MONTH>3.65</BC_3MONTH><BC_4MONTH>3.62</BC_4MONTH><BC_6MONTH>3.58</BC_6MONTH><BC_1YEAR>3.47</BC_1YEAR><BC_2YEAR>3.47</BC_2YEAR><BC_3YEAR>3.55</BC_3YEAR><BC_5YEAR>3.74</BC_5YEAR><BC_7YEAR>3.95</BC_7YEAR><BC_10YEAR>4.19</BC_10YEAR><BC_20YEAR>4.81</BC_20YEAR><BC_30YEAR>4.86</BC_30YEAR></properties></content></entry><entry><content><properties xmlns="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata"><NEW_DATE>2026-01-05T00:00:00</NEW_DATE><BC_1MONTH>3.71</BC_1MONTH><BC_1_5MONTH>3.68</BC_1_5MONTH><BC_2MONTH>3.64</BC_2MONTH><BC_3MONTH>3.64</BC_3MONTH><BC_4MONTH>3.61</BC_4MONTH><BC_6MONTH>3.57</BC_6MONTH><BC_1YEAR>3.47</BC_1YEAR><BC_2YEAR>3.46</BC_2YEAR><BC_3YEAR>3.53</BC_3YEAR><BC_5YEAR>3.71</BC_5YEAR><BC_7YEAR>3.92</BC_7YEAR><BC_10YEAR>4.17</BC_10YEAR> <BC_20YEAR m:null="true"></BC_20YEAR><BC_30YEAR>4.85</BC_30YEAR></properties></content></entry></feed>`
}

func TestTreasuryAcquireYearPersistsExactRawAndPreservesCurveSemantics(t *testing.T) {
	body := treasuryFixture()
	provider, deps := treasuryFixtureDependencies(t, body)
	result, err := provider.AcquireYear(context.Background(), deps, YearRequest{Year: 2026, PayloadID: "rpa_treasury_fixture", Retention: treasuryRetention()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Completeness != macroevidence.CompletenessComplete || len(result.Observations) != 28 {
		t.Fatalf("result = %+v", result)
	}
	stored, err := deps.Store.Get(context.Background(), result.Raw.Ref)
	if err != nil || string(stored) != body {
		t.Fatalf("exact raw bytes were not retained: %v", err)
	}
	first := result.Observations[0]
	if first.Tenor != "1 Mo" || first.Observation.ObservationDate != "2026-01-02" || first.Observation.Value.SourceValue != "3.72" || first.Observation.Value.Number == nil || first.Observation.AcquiredAt.IsZero() {
		t.Fatalf("normalized result = %+v", first)
	}
	var missing YieldObservation
	for _, item := range result.Observations {
		if item.Tenor == "20 Yr" && item.Observation.ObservationDate == "2026-01-05" {
			missing = item
		}
	}
	if missing.Observation.Value.Present || missing.Observation.Value.Number != nil || missing.Observation.Value.SourceValue != "" {
		t.Fatalf("missing value was not preserved conservatively: %+v", missing)
	}
}

func TestTreasuryRejectsDuplicateDateAndMalformedValue(t *testing.T) {
	duplicate := strings.Replace(treasuryFixture(), "2026-01-05T00:00:00", "2026-01-02T00:00:00", 1)
	provider, deps := treasuryFixtureDependencies(t, duplicate)
	if _, err := provider.AcquireYear(context.Background(), deps, YearRequest{Year: 2026, PayloadID: "rpa_treasury_duplicate", Retention: treasuryRetention()}); err == nil {
		t.Fatal("duplicate date was accepted")
	}
	malformed := strings.Replace(treasuryFixture(), "<BC_3YEAR>3.55</BC_3YEAR>", "<BC_3YEAR>NaN</BC_3YEAR>", 1)
	provider, deps = treasuryFixtureDependencies(t, malformed)
	if _, err := provider.AcquireYear(context.Background(), deps, YearRequest{Year: 2026, PayloadID: "rpa_treasury_malformed", Retention: treasuryRetention()}); err == nil {
		t.Fatal("malformed value was accepted")
	}
}

func TestTreasuryLiveOfficialXMLSmoke(t *testing.T) {
	if os.Getenv("JAX_RUN_LIVE_WP0305") != "1" {
		t.Skip("set JAX_RUN_LIVE_WP0305=1 to run the live official-source smoke test")
	}
	provider, err := NewProvider(DefaultConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := RegisterProvider(registry); err != nil {
		t.Fatal(err)
	}
	policy := treasuryPolicy()
	policy.Retry.PerAttemptTimeout = 60 * time.Second
	policy.Retry.MaximumElapsed = 2 * time.Minute
	executor, err := providercontract.NewOperationalExecutor(registry, policy, providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	store := providercontract.NewMemoryRawPayloadStore()
	result, err := provider.AcquireYear(context.Background(), Dependencies{Registry: registry, Executor: executor, Store: store}, YearRequest{Year: 2026, PayloadID: "rpa_treasury_live_smoke", Retention: treasuryRetention()})
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := store.Get(context.Background(), result.Raw.Ref)
	if err != nil || len(bytes) == 0 || len(result.Observations) == 0 {
		t.Fatalf("live Treasury smoke did not retain and normalize payload: %v", err)
	}
}
