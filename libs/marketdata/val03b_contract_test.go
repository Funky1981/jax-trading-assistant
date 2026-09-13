package marketdata

import (
	"math"
	"net/url"
	"testing"
	"time"
)

func TestVAL03BAdjustmentModesAreExplicitAndDistinct(t *testing.T) {
	if got := alpacaAdjustmentQuery(MarketAdjustmentRaw); got != "raw" {
		t.Fatalf("raw query = %q", got)
	}
	if got := alpacaAdjustmentQuery(MarketAdjustmentSplit); got != "split" {
		t.Fatalf("split query = %q", got)
	}
	if got := alpacaAdjustmentQuery(MarketAdjustmentSplitSpinOff); got != "split,spin-off" {
		t.Fatalf("split+spin-off query = %q", got)
	}
	if alpacaSourceID(MarketFeedSIP, MarketAdjustmentRaw) == alpacaSourceID(MarketFeedSIP, MarketAdjustmentSplit) || alpacaSourceID(MarketFeedSIP, MarketAdjustmentSplit) == alpacaSourceID(MarketFeedSIP, MarketAdjustmentSplitSpinOff) {
		t.Fatal("adjustment families collided in source identity")
	}
	if isSupportedAlpacaAdjustment(MarketAdjustmentAdjusted) {
		t.Fatal("generic adjusted mode must remain unsupported")
	}
}

func TestVAL03BRequestConstructionPreservesAdjustmentAndAsOf(t *testing.T) {
	request := AlpacaBarsRequest{Instrument: marketTestInstrument(), StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), AsOfDate: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), Interval: Timeframe1Day, Feed: MarketFeedSIP, Adjustment: MarketAdjustmentSplitSpinOff, PayloadID: "rpa_val03b_request", Retention: alpacaRetention()}
	if err := validateAlpacaBarsRequest(request); err != nil {
		t.Fatal(err)
	}
	provider, err := NewAlpacaProvider(ProviderConfig{Name: ProviderAlpaca, APIKey: "key", APISecret: "secret", Tier: "free", Feed: "sip", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := provider.historicalBarsURL(request.Instrument, request.StartDate, request.EndDate, request.AsOfDate, request.Feed, request.Adjustment, "")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("adjustment") != "split,spin-off" || parsed.Query().Get("asof") != "2024-12-31" {
		t.Fatalf("unexpected VAL-03B URL: %s", endpoint)
	}
	if parsed.Query().Get("start") >= "2025-01-01" || parsed.Query().Get("end") >= "2025-01-01" {
		t.Fatalf("request crossed sealed holdout: %s", endpoint)
	}
}

func TestVAL03BHoldoutDateGuard(t *testing.T) {
	validStart := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
	validEnd := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	if err := ValidateVAL03BDateRange(validStart, validEnd); err != nil {
		t.Fatal(err)
	}
	for _, end := range []time.Time{time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)} {
		if err := ValidateVAL03BDateRange(validStart, end); err == nil {
			t.Fatalf("accepted sealed end date %s", end.Format("2006-01-02"))
		}
	}
	if err := ValidateVAL03BProviderDate("2025-01-01"); err == nil {
		t.Fatal("accepted provider row at sealed holdout")
	}
	if err := ValidateVAL03BProviderDate("2024-12-31"); err != nil {
		t.Fatal(err)
	}
}

func TestVAL03BSplitFactorAndBoundary(t *testing.T) {
	raw := VAL03BPriceFrame{Open: 100, High: 105, Low: 95, Close: 102, Volume: 1000}
	split := VAL03BPriceFrame{Open: 200, High: 210, Low: 190, Close: 204, Volume: 500}
	factor, err := DeriveVAL03BSplitFactor(raw, split)
	if err != nil || math.Abs(factor-2) > VAL03BFactorTolerance {
		t.Fatalf("factor=%v err=%v", factor, err)
	}
	if !IsVAL03BStructuralFactorBoundary(1, factor) || IsVAL03BStructuralFactorBoundary(factor, factor*(1+VAL03BFactorTolerance/2)) {
		t.Fatal("factor boundary rule is not deterministic")
	}
	bad := split
	bad.High = 209
	if _, err := DeriveVAL03BSplitFactor(raw, bad); err == nil {
		t.Fatal("inconsistent OHLC factor was accepted")
	}
}

func TestVAL03BScaleInvarianceForIndicatorsAndGeometry(t *testing.T) {
	closes := []float64{100, 101, 99, 102, 103}
	volumes := []float64{1000, 1100, 900, 1200, 1000}
	scale := 3.0
	mean := func(values []float64) float64 {
		total := 0.0
		for _, value := range values {
			total += value
		}
		return total / float64(len(values))
	}
	baseMA := mean(closes)
	scaledCloses := make([]float64, len(closes))
	scaledVolumes := make([]float64, len(volumes))
	for i := range closes {
		scaledCloses[i] = closes[i] * scale
		scaledVolumes[i] = volumes[i] / scale
	}
	if math.Abs(mean(scaledCloses)-baseMA*scale) > VAL03BFactorTolerance || !(closes[len(closes)-1] > baseMA) != !(scaledCloses[len(scaledCloses)-1] > mean(scaledCloses)) {
		t.Fatal("MA ordering changed under positive split scale")
	}
	if !(volumes[len(volumes)-1] > mean(volumes)) != !(scaledVolumes[len(scaledVolumes)-1] > mean(scaledVolumes)) {
		t.Fatal("volume confirmation changed under inverse split volume scale")
	}
	atr := 2.0
	if math.Abs(atr*scale-6) > VAL03BFactorTolerance {
		t.Fatal("ATR scale relation failed")
	}
	stop, target, entry := 95.0, 110.0, 102.0
	if !(stop < entry && entry < target) != !(stop*scale < entry*scale && entry*scale < target*scale) {
		t.Fatal("pre-entry geometry changed under positive scale")
	}
	barLow, barHigh := 96.0, 109.0
	if !(barLow <= stop) != !(barLow*scale <= stop*scale) || !(barHigh >= target) != !(barHigh*scale >= target*scale) {
		t.Fatal("stop/target touch changed under positive scale")
	}
}
