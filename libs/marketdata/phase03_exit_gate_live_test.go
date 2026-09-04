package marketdata

import (
	"context"
	"os"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

// This is an explicitly invoked live verification. The ordinary suite never
// depends on the internet or on a market credential.
func TestPhase03ExitGateLiveAAPLMarketEvidence(t *testing.T) {
	if os.Getenv("JAX_RUN_LIVE_PHASE03_GATE") != "1" {
		t.Skip("set JAX_RUN_LIVE_PHASE03_GATE=1 to run the bounded live Phase-03 gate check")
	}
	key := os.Getenv("FINANCIAL_DATASETS_API_KEY")
	if key == "" {
		t.Skip("FINANCIAL_DATASETS_API_KEY is not configured")
	}

	instrument := marketTestInstrument()
	provider, err := NewFinancialDatasetsProvider(ProviderConfig{Name: ProviderFinancialDatasets, APIKey: key, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(FinancialDatasetsProviderDefinition()); err != nil {
		t.Fatal(err)
	}
	normalizers, err := providercontract.NewNormalizerRegistry(registry)
	if err != nil {
		t.Fatal(err)
	}
	normalizer, err := NewFinancialDatasetsDailyBarsNormalizer(singleInstrumentResolver{instrument: instrument})
	if err != nil {
		t.Fatal(err)
	}
	if err := normalizers.Register(normalizer); err != nil {
		t.Fatal(err)
	}
	pipeline, err := providercontract.NewNormalizationPipeline(registry, normalizers)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := providercontract.NewOperationalExecutor(registry, marketTestPolicy(), providercontract.SystemTimeSource{}, nil, providercontract.NewMemoryInstrumentation())
	if err != nil {
		t.Fatal(err)
	}
	store := providercontract.NewMemoryRawPayloadStore()
	retention := providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionNotAuthorized}
	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -7)
	result, err := provider.AcquireAndNormalizeDailyBars(context.Background(), FinancialDatasetsBarsDependencies{Registry: registry, Executor: executor, Store: store, Pipeline: pipeline}, FinancialDatasetsBarsRequest{Instrument: instrument, StartDate: start, EndDate: end, Interval: Timeframe1Day, PayloadID: "rpa_phase03_gate_market_live", Retention: retention})
	if err != nil {
		t.Fatalf("accepted Financial Datasets live path failed: %v; execution=%+v", err, result.Execution)
	}
	if len(result.Bars) == 0 || result.Raw.Ref.Content.Digest.Value == "" {
		t.Fatal("live market result did not contain normalized bars and raw digest")
	}
	bar := result.Bars[len(result.Bars)-1]
	t.Logf("LIVE_ACQUIRED_NOW provider=%s source=%s raw_payload=%s digest=%s normalized_close=%v observation_boundary=%s acquired_at=%s", FinancialDatasetsProviderID, FinancialDatasetsHistoricalSourceID, result.Raw.Ref.ID, result.Raw.Ref.Content.Digest.Value, *bar.Close.Value.Number, bar.End.Format(time.RFC3339), result.Raw.Ref.ReceivedAt.Format(time.RFC3339))
}
