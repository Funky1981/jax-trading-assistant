package cboe

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

func cboeRetention() providercontract.RawPayloadRetentionPolicy {
	return providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionRestricted}
}

func cboePolicy() providercontract.OperationalPolicy {
	identity := func(id, name, namespace string) canonical.ComponentIdentity {
		return canonical.ComponentIdentity{ID: id, Kind: canonical.ComponentKindPolicy, Name: name, Version: canonical.VersionIdentity{Namespace: namespace, Value: "fixture/v1"}}
	}
	return providercontract.OperationalPolicy{
		ContractVersion: providercontract.OperationalPolicyContractV1,
		Classification:  providercontract.ClassificationPolicy{ContractVersion: providercontract.ClassificationPolicyContractV1, Identity: identity("cmp_cboe_class", "Cboe fixture classification", "jax.policy.failure_classification")},
		Retry:           providercontract.RetryPolicy{ContractVersion: providercontract.RetryPolicyContractV1, Identity: identity("cmp_cboe_retry", "Cboe fixture retry", "jax.policy.retry"), MaximumAttempts: 1, MaximumElapsed: time.Minute, PerAttemptTimeout: 10 * time.Second, RetryableFailures: []providercontract.FailureClass{providercontract.FailureTransportTransient, providercontract.FailureProviderServer, providercontract.FailureTemporaryUnavailable, providercontract.FailureRateLimited, providercontract.FailureAttemptDeadline}, Backoff: providercontract.BackoffPolicy{InitialDelay: time.Millisecond, Multiplier: 2, MaximumDelay: time.Second, Jitter: providercontract.JitterNone}},
		RateLimit:       providercontract.RateLimitPolicy{ContractVersion: providercontract.RateLimitPolicyContractV1, Identity: identity("cmp_cboe_rate", "Cboe fixture rate limit", "jax.policy.rate_limit"), RequestLimit: 100, Window: time.Minute, ConcurrencyLimit: 4, MaximumProviderDelay: time.Second},
		Health:          providercontract.HealthPolicy{ContractVersion: providercontract.HealthPolicyContractV1, Identity: identity("cmp_cboe_health", "Cboe fixture health", "jax.policy.health"), DegradedAfterFailures: 1, UnavailableAfterFailures: 3, RecoverySuccesses: 1, AssessmentHorizon: time.Hour},
		Component:       canonical.ComponentIdentity{ID: "cmp_cboe_build", Kind: canonical.ComponentKindSoftwareBuild, Name: "Cboe fixture executor", Version: canonical.VersionIdentity{Namespace: "git.commit", Value: "cboe-fixture"}},
	}
}

func cboeFixtureDependencies(t *testing.T, body string) (*Provider, Dependencies) {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	provider, err := NewProvider(Config{HistoryURL: server.URL + "/VIX_History.csv", MaxResponseBytes: 1 << 20}, server.Client())
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
	executor, err := providercontract.NewOperationalExecutor(registry, cboePolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return provider, Dependencies{Registry: registry, Executor: executor, Store: providercontract.NewMemoryRawPayloadStore()}
}

func TestCboeAcquireHistoryPersistsExactRawAndPreservesDateSemantics(t *testing.T) {
	body := "DATE,OPEN,HIGH,LOW,CLOSE\n01/02/2025,17.240000,18.0,16.0,17.240000\n01/03/2025,18.190000,19.0,17.0,N/A\n"
	provider, deps := cboeFixtureDependencies(t, body)
	result, err := provider.AcquireHistory(context.Background(), deps, HistoryRequest{PayloadID: "rpa_cboe_fixture", Retention: cboeRetention()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Completeness != macroevidence.CompletenessComplete || len(result.Observations) != 2 {
		t.Fatalf("result = %+v", result)
	}
	stored, err := deps.Store.Get(context.Background(), result.Raw.Ref)
	if err != nil || string(stored) != body {
		t.Fatalf("exact raw bytes were not retained: %v", err)
	}
	if result.Series.ProviderSeriesID != "VIX" || result.Observations[0].ObservationDate != "2025-01-02" || result.Observations[0].Value.SourceValue != "17.240000" || result.Observations[0].AcquiredAt.IsZero() {
		t.Fatalf("normalized result = %+v", result)
	}
	if result.Observations[1].Value.Present || result.Observations[1].Value.Number != nil || result.Observations[1].Value.SourceValue != "N/A" {
		t.Fatalf("missing VIX value = %+v", result.Observations[1].Value)
	}
	if result.Observations[0].Provenance.Inputs[0].Evidence == nil || result.Observations[0].Provenance.Inputs[0].Evidence.Content.Digest != result.Raw.Ref.Content.Digest {
		t.Fatal("provenance did not point to the stored raw payload")
	}
}

func TestCboeRejectsDuplicateDatesMalformedRowsAndValues(t *testing.T) {
	body := "DATE,OPEN,HIGH,LOW,CLOSE\n01/02/2025,17,18,16,17\n01/02/2025,18,19,17,18\n"
	provider, deps := cboeFixtureDependencies(t, body)
	if _, err := provider.AcquireHistory(context.Background(), deps, HistoryRequest{PayloadID: "rpa_cboe_duplicate", Retention: cboeRetention()}); err == nil {
		t.Fatal("duplicate date was accepted")
	}
	malformed := strings.Replace(body, "01/02/2025,17,18,16,17", "bad,17,18,16,17", 1)
	provider, deps = cboeFixtureDependencies(t, malformed)
	if _, err := provider.AcquireHistory(context.Background(), deps, HistoryRequest{PayloadID: "rpa_cboe_bad_date", Retention: cboeRetention()}); err == nil {
		t.Fatal("malformed date was accepted")
	}
	badValue := strings.Replace(body, "17,18,16,17", "17,18,16,NaN", 1)
	provider, deps = cboeFixtureDependencies(t, badValue)
	if _, err := provider.AcquireHistory(context.Background(), deps, HistoryRequest{PayloadID: "rpa_cboe_bad_value", Retention: cboeRetention()}); err == nil {
		t.Fatal("malformed value was accepted")
	}
}

func TestCboeLiveOfficialCSVSmoke(t *testing.T) {
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
	executor, err := providercontract.NewOperationalExecutor(registry, cboePolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	store := providercontract.NewMemoryRawPayloadStore()
	result, err := provider.AcquireHistory(context.Background(), Dependencies{Registry: registry, Executor: executor, Store: store}, HistoryRequest{PayloadID: "rpa_cboe_live_smoke", Retention: cboeRetention()})
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := store.Get(context.Background(), result.Raw.Ref)
	if err != nil || len(bytes) == 0 || len(result.Observations) == 0 {
		t.Fatalf("live Cboe smoke did not retain and normalize payload: %v", err)
	}
	item := result.Observations[len(result.Observations)-1]
	t.Logf("LIVE_ACQUIRED_NOW provider=%s source=%s raw_payload=%s digest=%s normalized_observation=%s observation_date=%s source_value=%s acquired_at=%s", ProviderID, SourceID, result.Raw.Ref.ID, result.Raw.Ref.Content.Digest.Value, item.ID, item.ObservationDate, item.Value.SourceValue, result.Raw.Ref.ReceivedAt)
}
