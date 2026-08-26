package marketdata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

type marketTestResolver struct{ instrument canonical.Instrument }

func (r marketTestResolver) ResolveFinancialDatasetsTicker(string) (canonical.Instrument, error) {
	return r.instrument, nil
}

type marketTestClock struct{ now time.Time }

func (c marketTestClock) Now() time.Time                             { return c.now }
func (c marketTestClock) Sleep(context.Context, time.Duration) error { return nil }
func (c marketTestClock) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

func marketTestPolicy() providercontract.OperationalPolicy {
	identity := func(id, namespace string, kind canonical.ComponentKind) canonical.ComponentIdentity {
		return canonical.ComponentIdentity{ID: id, Kind: kind, Name: id, Version: canonical.VersionIdentity{Namespace: namespace, Value: "v1"}}
	}
	return providercontract.OperationalPolicy{
		ContractVersion: providercontract.OperationalPolicyContractV1,
		Classification:  providercontract.ClassificationPolicy{ContractVersion: providercontract.ClassificationPolicyContractV1, Identity: identity("cmp_market_classification", "jax.policy.failure_classification", canonical.ComponentKindPolicy)},
		Retry:           providercontract.RetryPolicy{ContractVersion: providercontract.RetryPolicyContractV1, Identity: identity("cmp_market_retry", "jax.policy.retry", canonical.ComponentKindPolicy), MaximumAttempts: 2, MaximumElapsed: time.Minute, PerAttemptTimeout: 10 * time.Second, RetryableFailures: []providercontract.FailureClass{providercontract.FailureTransportTransient, providercontract.FailureProviderServer, providercontract.FailureTemporaryUnavailable, providercontract.FailureRateLimited, providercontract.FailureAttemptDeadline}, Backoff: providercontract.BackoffPolicy{InitialDelay: time.Millisecond, Multiplier: 2, MaximumDelay: time.Second, Jitter: providercontract.JitterNone}},
		RateLimit:       providercontract.RateLimitPolicy{ContractVersion: providercontract.RateLimitPolicyContractV1, Identity: identity("cmp_market_rate_limit", "jax.policy.rate_limit", canonical.ComponentKindPolicy), RequestLimit: 10, Window: time.Minute, ConcurrencyLimit: 1, MaximumProviderDelay: time.Second},
		Health:          providercontract.HealthPolicy{ContractVersion: providercontract.HealthPolicyContractV1, Identity: identity("cmp_market_health", "jax.policy.health", canonical.ComponentKindPolicy), DegradedAfterFailures: 1, UnavailableAfterFailures: 3, RecoverySuccesses: 1, AssessmentHorizon: time.Hour},
		Component:       identity("cmp_market_operational", "git.commit", canonical.ComponentKindSoftwareBuild),
	}
}

func marketTestInstrument() canonical.Instrument {
	return canonical.Instrument{ContractVersion: canonical.InstrumentContractV1, ID: "ins_aapl_common", Type: canonical.InstrumentTypeEquity, Name: "Apple Inc. common stock", Currency: "USD", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ExternalIDs: []canonical.ExternalID{{Namespace: "ticker.us", Value: "AAPL"}, {Namespace: "financialdatasets.ticker", Value: "AAPL"}}}
}

func TestFinancialDatasetsHardenedDailyBarsEndToEnd(t *testing.T) {
	payload := []byte(`{"ticker":"AAPL","prices":[{"time":"2026-08-20","open":100,"high":105,"low":99,"close":104,"volume":1000},{"time":"2026-08-21","open":104,"high":108,"low":103,"close":107,"volume":1200}]}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("interval") != "day" {
			t.Fatalf("unexpected interval: %s", r.URL.Query().Get("interval"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	provider, err := NewFinancialDatasetsProvider(ProviderConfig{Name: ProviderFinancialDatasets, APIKey: "fixture-key", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	provider.baseURL, provider.client = server.URL, server.Client()
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	definition := FinancialDatasetsProviderDefinition()
	if err := registry.Register(definition); err != nil {
		t.Fatal(err)
	}
	normals, err := providercontract.NewNormalizerRegistry(registry)
	if err != nil {
		t.Fatal(err)
	}
	resolver := marketTestResolver{instrument: marketTestInstrument()}
	normalizer, err := NewFinancialDatasetsDailyBarsNormalizer(resolver)
	if err != nil {
		t.Fatal(err)
	}
	if err := normals.Register(normalizer); err != nil {
		t.Fatal(err)
	}
	pipeline, err := providercontract.NewNormalizationPipeline(registry, normals)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := providercontract.NewOperationalExecutor(registry, marketTestPolicy(), marketTestClock{now: time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)}, nil, providercontract.NewMemoryInstrumentation())
	if err != nil {
		t.Fatal(err)
	}
	store := providercontract.NewMemoryRawPayloadStore()
	retention := providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionNotAuthorized}
	result, err := provider.AcquireAndNormalizeDailyBars(context.Background(), FinancialDatasetsBarsDependencies{Registry: registry, Executor: executor, Store: store, Pipeline: pipeline}, FinancialDatasetsBarsRequest{Instrument: resolver.instrument, StartDate: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC), Interval: Timeframe1Day, PayloadID: "rpa_market_fd_fixture", Retention: retention})
	if err != nil {
		t.Fatal(err)
	}
	if result.Execution.Status != providercontract.ExecutionSucceeded || result.Execution.Health.Status != providercontract.RuntimeHealthy {
		t.Fatalf("unexpected execution: %+v", result.Execution)
	}
	if got, err := store.Get(context.Background(), result.Raw.Ref); err != nil || string(got) != string(payload) {
		t.Fatalf("raw bytes were not preserved exactly: %v", err)
	}
	if len(result.Normalization.Records) != 10 || len(result.Bars) != 2 {
		t.Fatalf("expected 10 observations and 2 bars, got %d and %d", len(result.Normalization.Records), len(result.Bars))
	}
	for _, bar := range result.Bars {
		if bar.Instrument.ID != string(resolver.instrument.ID) || bar.TimestampSemantics != MarketTimestampProviderDateIntervalEnd || bar.TimestampAuthority != MarketTimestampAuthorityIntervalBoundary || bar.MarketTimezone != "UNKNOWN" || bar.Adjustment != MarketAdjustmentUnknown {
			t.Fatalf("unexpected market semantics: %+v", bar)
		}
		if bar.ProviderDate != bar.Start.Format("2006-01-02") || !bar.Start.Equal(bar.End.AddDate(0, 0, -1)) || bar.Start.Location() != time.UTC || bar.End.Location() != time.UTC {
			t.Fatalf("provider date was not represented as a neutral UTC interval: %+v", bar)
		}
		if _, err := bar.ExchangeCloseTime(); err == nil {
			t.Fatal("provider calendar boundary must not be exposed as an exchange-close timestamp")
		}
		for _, observation := range []canonical.Observation{bar.Open, bar.High, bar.Low, bar.Close, bar.Volume} {
			if err := observation.Validate(); err != nil {
				t.Fatalf("invalid canonical observation: %v", err)
			}
			if observation.Provenance == nil || len(observation.EvidenceIDs) != 1 {
				t.Fatal("observation lacks immutable provenance/evidence")
			}
		}
	}
	encoded, _ := json.Marshal(result.Bars)
	if string(encoded) == "" || string(encoded) == "null" {
		t.Fatal("research-facing projection was empty")
	}
}

func TestFinancialDatasetsHardenedHistoricalBarFreshnessUsesObservedTime(t *testing.T) {
	payload := []byte(`{"ticker":"AAPL","prices":[{"time":"2024-08-20","open":100,"high":105,"low":99,"close":104,"volume":1000}]}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	p, err := NewFinancialDatasetsProvider(ProviderConfig{Name: ProviderFinancialDatasets, APIKey: "fixture-key", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	p.baseURL, p.client = server.URL, server.Client()
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(FinancialDatasetsProviderDefinition()); err != nil {
		t.Fatal(err)
	}
	normals, err := providercontract.NewNormalizerRegistry(registry)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := NewFinancialDatasetsDailyBarsNormalizer(marketTestResolver{instrument: marketTestInstrument()}); err != nil {
		t.Fatal(err)
	} else if err := normals.Register(n); err != nil {
		t.Fatal(err)
	}
	pipeline, err := providercontract.NewNormalizationPipeline(registry, normals)
	if err != nil {
		t.Fatal(err)
	}
	receipt := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	executor, err := providercontract.NewOperationalExecutor(registry, marketTestPolicy(), marketTestClock{now: receipt}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	retention := providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionNotAuthorized}
	result, err := p.AcquireAndNormalizeDailyBars(context.Background(), FinancialDatasetsBarsDependencies{Registry: registry, Executor: executor, Store: providercontract.NewMemoryRawPayloadStore(), Pipeline: pipeline}, FinancialDatasetsBarsRequest{Instrument: marketTestInstrument(), StartDate: time.Date(2024, 8, 20, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2024, 8, 20, 0, 0, 0, 0, time.UTC), Interval: Timeframe1Day, PayloadID: "rpa_market_fd_old_fixture", Retention: retention})
	if err != nil {
		t.Fatal(err)
	}
	observation, ok := result.Normalization.Records[0].Record.(canonical.Observation)
	if !ok {
		t.Fatal("expected canonical observation")
	}
	policies := providercontract.NewFreshnessPolicyRegistry()
	policy := providercontract.FreshnessPolicy{ContractVersion: providercontract.FreshnessPolicyContractV1, Identity: canonical.ComponentIdentity{ID: "cmp_market_freshness", Kind: canonical.ComponentKindPolicy, Name: "market freshness", Version: canonical.VersionIdentity{Namespace: "jax.policy.freshness", Value: "v1"}}, CapabilityID: providercontract.CapabilityMarketBars, Target: result.Normalization.Target, UseClass: providercontract.DataUseResearch, ValidityMode: providercontract.FreshnessValidityAgeBounded, TimestampRole: providercontract.TimestampRoleObservedAt, FreshFor: 24 * time.Hour, ExpireAfter: 72 * time.Hour, MissingTimestamp: providercontract.MissingTimestampFail, LastKnownGood: providercontract.LastKnownGoodPolicy{ContractVersion: providercontract.LastKnownGoodPolicyContractV1, Identity: canonical.ComponentIdentity{ID: "cmp_market_lkg", Kind: canonical.ComponentKindPolicy, Name: "market lkg", Version: canonical.VersionIdentity{Namespace: "jax.policy.lkg", Value: "v1"}}, Mode: providercontract.FallbackProhibited}}
	if err := policies.Register(policy); err != nil {
		t.Fatal(err)
	}
	evaluator, err := providercontract.NewFreshnessEvaluator(registry, policies)
	if err != nil {
		t.Fatal(err)
	}
	evaluation, err := evaluator.Evaluate(providercontract.FreshnessEvaluationRequest{Policy: policy.Identity, UseClass: providercontract.DataUseResearch, EvaluationTime: receipt, Context: providercontract.FreshnessContextCurrentState, Record: providercontract.TemporalRecord{Normalized: result.Normalization.Records[0], Key: providercontract.FreshnessKey{CapabilityID: providercontract.CapabilityMarketBars, Target: result.Normalization.Target, Subject: observation.Subject, Qualifier: observation.Metric}, Lifecycle: providercontract.TemporalRecordLifecycle{State: providercontract.TemporalRecordActive}}})
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.State != providercontract.TemporalExpired || evaluation.RawPayloadID != "rpa_market_fd_old_fixture" || !evaluation.EvaluationTime.Equal(receipt) {
		t.Fatalf("historical observation incorrectly treated as fresh: %+v", evaluation)
	}
}

func TestFinancialDatasetsHistoricalPayloadFailsClosed(t *testing.T) {
	received := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	cases := []string{
		`{"ticker":"AAPL","prices":[{"time":"2026-08-20","open":100,"high":99,"low":98,"close":99,"volume":1}]}`,
		`{"ticker":"AAPL","prices":[{"time":"2026-08-20","open":100,"high":101,"low":99,"close":100,"volume":1},{"time":"2026-08-20","open":100,"high":101,"low":99,"close":100,"volume":1}]}`,
		`{"ticker":"AAPL","prices":[{"time":"2026-08-20T15:00:00Z","open":100,"high":101,"low":99,"close":100,"volume":1}]}`,
	}
	for _, raw := range cases {
		payload, err := parseFinancialDatasetsHistoricalPayload([]byte(raw))
		if err != nil {
			t.Fatalf("fixture should parse before semantic validation: %v", err)
		}
		if _, _, err := validateFinancialDatasetsHistoricalPayload(payload, received); err == nil {
			t.Fatalf("expected malformed market payload to fail: %s", raw)
		}
	}
	if err := validateFinancialDatasetsBarsRequest(FinancialDatasetsBarsRequest{Instrument: marketTestInstrument(), StartDate: received, EndDate: received, Interval: Timeframe15Min, PayloadID: "rpa_invalid", Retention: providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionNotAuthorized}}); err == nil {
		t.Fatal("intraday interval must be rejected by the documented EOD-only path")
	}
}
