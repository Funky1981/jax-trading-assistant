package marketdata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

func alpacaRetention() providercontract.RawPayloadRetentionPolicy {
	return providercontract.RawPayloadRetentionPolicy{Class: providercontract.RawPayloadRetentionReplayAudit, Policy: canonical.VersionIdentity{Namespace: "jax.raw_retention", Value: "replay-audit/v1"}, Redistribution: providercontract.RawPayloadRedistributionNotAuthorized}
}

func alpacaFixturePayload(next string, timestamp string) string {
	return `{"bars":{"AAPL":[{"c":` + func() string { return "227.16" }() + `,"h":229.12,"l":225.41,"n":100,"o":226.00,"t":"` + timestamp + `","v":1200000,"vw":227.40}]},"next_page_token":` + func() string {
		if next == "" {
			return `null`
		}
		return `"` + next + `"`
	}() + `}`
}

func alpacaFixtureDependencies(t *testing.T, body func(*http.Request) string) (*AlpacaProvider, AlpacaBarsDependencies, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = writer.Write([]byte(body(request)))
	}))
	t.Cleanup(server.Close)
	provider, err := NewAlpacaProvider(ProviderConfig{Name: ProviderAlpaca, APIKey: "alpaca-key", APISecret: "alpaca-secret", Tier: "free", Feed: "sip", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	provider.baseURL = server.URL
	provider.rawClient = server.Client()
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
	normalizer, err := newAlpacaBarsNormalizer(marketTestInstrument(), MarketFeedSIP, MarketAdjustmentUnadjusted)
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
	executor, err := providercontract.NewOperationalExecutor(registry, marketTestPolicy(), providercontract.SystemTimeSource{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return provider, AlpacaBarsDependencies{Registry: registry, Executor: executor, Store: providercontract.NewMemoryRawPayloadStore(), Pipeline: pipeline}, server
}

func TestAlpacaHardenedDailyBarsPersistsBeforeParseAndPreservesFeedSemantics(t *testing.T) {
	payload := alpacaFixturePayload("", "2026-08-28T04:00:00Z")
	provider, dependencies, server := alpacaFixtureDependencies(t, func(request *http.Request) string {
		if request.URL.Path != "/v2/stocks/bars" || request.URL.Query().Get("symbols") != "AAPL" || request.URL.Query().Get("timeframe") != "1Day" || request.URL.Query().Get("feed") != "sip" || request.URL.Query().Get("adjustment") != "raw" {
			t.Fatalf("unexpected Alpaca request: %s", request.URL.String())
		}
		if request.Header.Get("APCA-API-KEY-ID") != "alpaca-key" || request.Header.Get("APCA-API-SECRET-KEY") != "alpaca-secret" {
			t.Fatal("Alpaca authentication headers were not forwarded")
		}
		return payload
	})
	result, err := provider.AcquireAndNormalizeDailyBars(context.Background(), dependencies, AlpacaBarsRequest{Instrument: marketTestInstrument(), StartDate: time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC), Interval: Timeframe1Day, Feed: MarketFeedSIP, Adjustment: MarketAdjustmentUnadjusted, PayloadID: "rpa_alpaca_fixture", Retention: alpacaRetention()})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RawPayloads) != 1 || len(result.Bars) != 1 || result.RawPayloads[0].Ref.Source.ID != "src_alpaca_stock_bars_sip_unadjusted" {
		t.Fatalf("unexpected result shape: %+v", result)
	}
	stored, err := dependencies.Store.Get(context.Background(), result.RawPayloads[0].Ref)
	if err != nil || string(stored) != payload {
		t.Fatalf("raw bytes were not retained exactly: %v", err)
	}
	bar := result.Bars[0]
	if bar.Instrument.ID != "ins_aapl_common" || bar.Feed != MarketFeedSIP || bar.FeedCoverage != MarketFeedCoverageSIP || bar.Adjustment != MarketAdjustmentUnadjusted || bar.TimestampSemantics != MarketTimestampProviderBarTimestamp || bar.MarketTimezone != "America/New_York" {
		t.Fatalf("unexpected Alpaca market semantics: %+v", bar)
	}
	if bar.ProviderDate != "2026-08-28" || !bar.Open.ObservedAt.Equal(bar.Start) || !bar.Close.ObservedAt.Equal(bar.Start) {
		t.Fatalf("provider timestamp semantics were not preserved: %+v", bar)
	}
	if bar.Open.Source.ExternalID == nil || bar.Open.Source.ExternalID.Value != "sip/UNADJUSTED" {
		t.Fatalf("feed/adjustment semantics were not retained: %+v", bar.Open.Source)
	}
	if _, err := bar.ExchangeCloseTime(); err == nil {
		t.Fatal("Alpaca provider timestamp must not be represented as an exchange-close timestamp")
	}
	_ = server
}

func TestAlpacaHardenedPaginationProgressionAndDuplicateRejection(t *testing.T) {
	var mu sync.Mutex
	requests := 0
	provider, dependencies, _ := alpacaFixtureDependencies(t, func(request *http.Request) string {
		mu.Lock()
		defer mu.Unlock()
		requests++
		if request.URL.Query().Get("page_token") == "next" {
			return alpacaFixturePayload("", "2026-09-02T04:00:00Z")
		}
		return alpacaFixturePayload("next", "2026-08-28T04:00:00Z")
	})
	result, err := provider.AcquireAndNormalizeDailyBars(context.Background(), dependencies, AlpacaBarsRequest{Instrument: marketTestInstrument(), StartDate: time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), Interval: Timeframe1Day, Feed: MarketFeedSIP, Adjustment: MarketAdjustmentUnadjusted, PayloadID: "rpa_alpaca_pages", Retention: alpacaRetention()})
	if err != nil || requests != 2 || len(result.RawPayloads) != 2 || len(result.Bars) != 2 {
		t.Fatalf("pagination result=%+v err=%v requests=%d", result, err, requests)
	}

	duplicateRequests := 0
	duplicateProvider, duplicateDependencies, _ := alpacaFixtureDependencies(t, func(request *http.Request) string {
		duplicateRequests++
		if request.URL.Query().Get("page_token") == "next" {
			return alpacaFixturePayload("", "2026-08-28T04:00:00Z")
		}
		return alpacaFixturePayload("next", "2026-08-28T04:00:00Z")
	})
	duplicateResult, duplicateErr := duplicateProvider.AcquireAndNormalizeDailyBars(context.Background(), duplicateDependencies, AlpacaBarsRequest{Instrument: marketTestInstrument(), StartDate: time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC), Interval: Timeframe1Day, Feed: MarketFeedSIP, Adjustment: MarketAdjustmentUnadjusted, PayloadID: "rpa_alpaca_duplicate", Retention: alpacaRetention()})
	if duplicateErr == nil || !strings.Contains(duplicateErr.Error(), "duplicate Alpaca market metric") || duplicateRequests != 2 || len(duplicateResult.RawPayloads) != 2 {
		t.Fatalf("duplicate bar was not rejected after raw-first persistence: err=%v requests=%d raw_payloads=%d", duplicateErr, duplicateRequests, len(duplicateResult.RawPayloads))
	}
}

func TestAlpacaHardenedParserRejectsMalformedAndUnorderedBars(t *testing.T) {
	for name, raw := range map[string]string{
		"malformed":     "{",
		"unknown field": `{"bars":{"AAPL":[{"c":1,"h":1,"l":1,"o":1,"t":"2026-08-28T04:00:00Z","v":1,"unexpected":1}]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseAlpacaBarsPayload([]byte(raw)); err == nil {
				t.Fatal("malformed Alpaca payload was accepted")
			}
		})
	}
	if err := validateAlpacaFeed(MarketFeed("")); err == nil {
		t.Fatal("implicit Alpaca feed was accepted")
	}
	if !strings.Contains(alpacaSourceID(MarketFeedIEX, MarketAdjustmentUnadjusted), "iex") {
		t.Fatal("IEX source identity did not preserve feed semantics")
	}
	if MarketFeedCoverageIEX == MarketFeedCoverageSIP {
		t.Fatal("IEX and SIP coverage must remain distinct")
	}
	bar := CanonicalMarketBar{Instrument: canonical.ContractRef{Kind: canonical.ContractKindInstrument, ID: "ins_aapl_common", ContractVersion: canonical.InstrumentContractV1}, Interval: Timeframe1Day, Start: time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC), TimestampSemantics: MarketTimestampProviderBarTimestamp, TimestampAuthority: MarketTimestampAuthorityProviderTimestamp, Session: MarketSessionProviderEODUnspecified, MarketTimezone: "America/New_York", Feed: MarketFeedIEX, FeedCoverage: MarketFeedCoverageSIP}
	if err := bar.Validate(); err == nil || !strings.Contains(err.Error(), "single-exchange coverage") {
		t.Fatal("IEX feed was allowed to claim consolidated SIP coverage")
	}
}

func TestAlpacaHardenedRejectsFractionalVolume(t *testing.T) {
	provider, dependencies, _ := alpacaFixtureDependencies(t, func(*http.Request) string {
		return `{"bars":{"AAPL":[{"c":227.16,"h":229.12,"l":225.41,"o":226.00,"t":"2026-08-28T04:00:00Z","v":1.5}]}}`
	})
	_, err := provider.AcquireAndNormalizeDailyBars(context.Background(), dependencies, AlpacaBarsRequest{Instrument: marketTestInstrument(), StartDate: time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC), Interval: Timeframe1Day, Feed: MarketFeedSIP, Adjustment: MarketAdjustmentUnadjusted, PayloadID: "rpa_alpaca_fractional_volume", Retention: alpacaRetention()})
	if err == nil {
		t.Fatalf("fractional volume was accepted: %v", err)
	}
}
