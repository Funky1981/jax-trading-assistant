package marketdata

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	providercontract "jax-trading-assistant/libs/contracts/provider"
)

// This is an explicitly invoked live verification. The ordinary suite never
// depends on the internet or on a market credential.
func TestPhase03ExitGateLiveAAPLMarketEvidence(t *testing.T) {
	if os.Getenv("JAX_RUN_LIVE_PHASE03_GATE") != "1" {
		t.Skip("set JAX_RUN_LIVE_PHASE03_GATE=1 to run the bounded live Phase-03 gate check")
	}
	key := os.Getenv("ALPACA_API_KEY")
	secret := os.Getenv("ALPACA_API_SECRET")
	if key == "" || secret == "" {
		t.Skip("ALPACA_API_KEY and ALPACA_API_SECRET are not configured")
	}

	instrument := marketTestInstrument()
	feed := MarketFeed(strings.ToLower(strings.TrimSpace(os.Getenv("ALPACA_DATA_FEED"))))
	if feed == "" {
		feed = MarketFeedSIP
	}
	if err := validateAlpacaFeed(feed); err != nil {
		t.Fatal(err)
	}
	provider, err := NewAlpacaProvider(ProviderConfig{Name: ProviderAlpaca, APIKey: key, APISecret: secret, Tier: "free", Feed: feed.String(), Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := providercontract.NewRegistry(providercontract.RegistryContractV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(AlpacaProviderDefinition()); err != nil {
		t.Fatal(err)
	}
	normalizers, err := providercontract.NewNormalizerRegistry(registry)
	if err != nil {
		t.Fatal(err)
	}
	normalizer, err := newAlpacaBarsNormalizer(instrument, feed, MarketAdjustmentUnadjusted)
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
	retention := alpacaRetention()
	end := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -2)
	start := end.AddDate(0, 0, -7)
	result, err := provider.AcquireAndNormalizeDailyBars(context.Background(), AlpacaBarsDependencies{Registry: registry, Executor: executor, Store: store, Pipeline: pipeline}, AlpacaBarsRequest{Instrument: instrument, StartDate: start, EndDate: end, Interval: Timeframe1Day, Feed: feed, Adjustment: MarketAdjustmentUnadjusted, PayloadID: "rpa_phase03_gate_market_alpaca_live", Retention: retention})
	if err != nil {
		t.Fatalf("accepted Alpaca live path failed: %v; executions=%d", err, len(result.Executions))
	}
	if len(result.Bars) == 0 || len(result.RawPayloads) == 0 || result.RawPayloads[0].Ref.Content.Digest.Value == "" {
		t.Fatal("live market result did not contain normalized bars and raw digest")
	}
	bar := result.Bars[len(result.Bars)-1]
	if bar.Close.Value.Number == nil {
		t.Fatal("live market close has no numeric value")
	}
	t.Logf("LIVE_ACQUIRED_NOW provider=%s source=%s feed=%s adjustment=%s raw_payload=%s digest=%s normalized_close=%v provider_date=%s acquired_at=%s", AlpacaProviderID, alpacaSourceID(feed, MarketAdjustmentUnadjusted), feed, MarketAdjustmentUnadjusted, result.RawPayloads[0].Ref.ID, result.RawPayloads[0].Ref.Content.Digest.Value, *bar.Close.Value.Number, bar.ProviderDate, result.RawPayloads[0].Ref.ReceivedAt.Format(time.RFC3339))
}
