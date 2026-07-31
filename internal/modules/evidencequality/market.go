package evidencequality

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type marketIndex struct {
	series map[string][]Candle
	end    time.Time
}

type seriesResult struct {
	EffectiveAnchor time.Time
	Start           float64
	End             float64
	RawReturn       float64
	Range           float64
	MFE             float64
	MAE             float64
	Count           int
	Source          string
	Semantics       string
}

func newMarketIndex(candles []Candle) marketIndex {
	index := marketIndex{series: map[string][]Candle{}}
	for _, candle := range candles {
		key := seriesKey(candle.Symbol, candle.Timeframe, candle.Source)
		index.series[key] = append(index.series[key], candle)
		end := candle.Timestamp
		if candle.Timeframe == "1h" {
			end = end.Add(time.Hour)
		} else if candle.Timeframe == "1d" {
			end = marketClose(candle.Timestamp)
		}
		if end.After(index.end) {
			index.end = end
		}
	}
	for key := range index.series {
		sort.Slice(index.series[key], func(i, j int) bool { return index.series[key][i].Timestamp.Before(index.series[key][j].Timestamp) })
	}
	return index
}

func seriesKey(symbol, timeframe, source string) string {
	return strings.ToUpper(symbol) + "|" + timeframe + "|" + source
}

func (index marketIndex) sources(symbol, timeframe string) []string {
	seen := map[string]bool{}
	for key := range index.series {
		parts := strings.Split(key, "|")
		if len(parts) == 3 && parts[0] == strings.ToUpper(symbol) && parts[1] == timeframe {
			seen[parts[2]] = true
		}
	}
	priority := map[string]int{"ib-bridge": 0, "alpaca": 1}
	result := make([]string, 0, len(seen))
	for source := range seen {
		result = append(result, source)
	}
	sort.Slice(result, func(i, j int) bool {
		pi, okI := priority[result[i]]
		if !okI {
			pi = 100
		}
		pj, okJ := priority[result[j]]
		if !okJ {
			pj = 100
		}
		if pi == pj {
			return result[i] < result[j]
		}
		return pi < pj
	})
	return result
}

func (index marketIndex) outcome(symbol string, anchor time.Time, horizon string, maxIntradayDelay time.Duration) (seriesResult, bool) {
	timeframe := "1d"
	if horizon == "1h" {
		timeframe = "1h"
	}
	for _, source := range index.sources(symbol, timeframe) {
		series := index.series[seriesKey(symbol, timeframe, source)]
		var result seriesResult
		var ok bool
		if horizon == "1h" {
			result, ok = intradayOutcome(series, anchor, maxIntradayDelay)
		} else {
			bars := 1
			if horizon == "1w" {
				bars = 5
			}
			result, ok = dailyOutcome(series, anchor, bars)
		}
		if ok {
			return result, true
		}
	}
	return seriesResult{}, false
}

func (index marketIndex) intradayAnchorPrice(symbol string, anchor time.Time, maxDelay time.Duration) (time.Time, float64, bool) {
	for _, source := range index.sources(symbol, "1h") {
		for _, candle := range index.series[seriesKey(symbol, "1h", source)] {
			if !candle.Timestamp.Before(anchor) {
				if candle.Timestamp.Sub(anchor) <= maxDelay && candle.Open > 0 {
					return candle.Timestamp, candle.Open, true
				}
				break
			}
		}
	}
	return time.Time{}, 0, false
}

func intradayOutcome(series []Candle, anchor time.Time, maxDelay time.Duration) (seriesResult, bool) {
	for _, candle := range series {
		if candle.Timestamp.Before(anchor) {
			continue
		}
		if candle.Timestamp.Sub(anchor) > maxDelay || candle.Open <= 0 {
			return seriesResult{}, false
		}
		return buildSeriesResult([]Candle{candle}, candle.Timestamp), true
	}
	return seriesResult{}, false
}

func dailyOutcome(series []Candle, anchor time.Time, bars int) (seriesResult, bool) {
	for i, candle := range series {
		openAt := marketOpen(candle.Timestamp)
		if openAt.Before(anchor) {
			continue
		}
		if i+bars > len(series) || candle.Open <= 0 {
			return seriesResult{}, false
		}
		window := series[i : i+bars]
		for j := 1; j < len(window); j++ {
			if tradingDateGap(window[j-1].Timestamp, window[j].Timestamp) > 4*24*time.Hour {
				return seriesResult{}, false
			}
		}
		return buildSeriesResult(window, openAt), true
	}
	return seriesResult{}, false
}

func buildSeriesResult(window []Candle, effective time.Time) seriesResult {
	start := window[0].Open
	high, low := window[0].High, window[0].Low
	for _, candle := range window[1:] {
		if candle.High > high {
			high = candle.High
		}
		if candle.Low < low {
			low = candle.Low
		}
	}
	end := window[len(window)-1].Close
	return seriesResult{
		EffectiveAnchor: effective, Start: start, End: end, RawReturn: end/start - 1,
		Range: (high - low) / start, MFE: high/start - 1, MAE: low/start - 1,
		Count: len(window), Source: window[0].Source, Semantics: window[0].TimestampSemantics,
	}
}

func marketOpen(timestamp time.Time) time.Time {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(fmt.Sprintf("load market timezone: %v", err))
	}
	year, month, day := sessionDate(timestamp, location)
	return time.Date(year, month, day, 9, 30, 0, 0, location).UTC()
}

func marketClose(timestamp time.Time) time.Time {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(fmt.Sprintf("load market timezone: %v", err))
	}
	year, month, day := sessionDate(timestamp, location)
	return time.Date(year, month, day, 16, 0, 0, 0, location).UTC()
}

func tradingDateGap(left, right time.Time) time.Duration {
	location, _ := time.LoadLocation("America/New_York")
	ay, am, ad := sessionDate(left, location)
	by, bm, bd := sessionDate(right, location)
	dateA := time.Date(ay, am, ad, 0, 0, 0, 0, time.UTC)
	dateB := time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC)
	return dateB.Sub(dateA)
}

func sessionDate(timestamp time.Time, location *time.Location) (int, time.Month, int) {
	utc := timestamp.UTC()
	if utc.Hour() == 0 && utc.Minute() == 0 {
		return utc.Year(), utc.Month(), utc.Day()
	}
	local := timestamp.In(location)
	return local.Year(), local.Month(), local.Day()
}

func calculateOutcomes(event Event, mapping Mapping, index marketIndex, rules Ruleset) []Outcome {
	if !mapping.Mapped {
		return nil
	}
	anchors := []struct {
		Name string
		At   *time.Time
	}{
		{Name: "publication", At: &event.PublicationAt},
		{Name: "collection", At: event.CollectionAt},
		{Name: "receipt", At: &event.ReceiptAt},
		{Name: "decision", At: &event.DecisionAt},
	}
	horizons := []string{"1h", "1d", "1w"}
	result := []Outcome{}
	for _, anchor := range anchors {
		if anchor.At == nil {
			continue
		}
		for _, horizon := range horizons {
			market, ok := index.outcome(mapping.Symbol, *anchor.At, horizon, time.Duration(rules.MaximumIntradayAnchorDelayMinutes)*time.Minute)
			if !ok {
				continue
			}
			outcome := Outcome{
				Anchor: anchor.Name, AnchorAt: *anchor.At, EffectiveAnchorAt: market.EffectiveAnchor,
				AnchorDelaySeconds: market.EffectiveAnchor.Sub(*anchor.At).Seconds(), Horizon: horizon,
				Symbol: mapping.Symbol, Benchmark: mapping.Benchmark, StartPrice: market.Start, EndPrice: market.End,
				RawReturn: market.RawReturn, AbsoluteRawReturn: math.Abs(market.RawReturn),
				RealisedRange: market.Range, MaximumFavourableExcursion: market.MFE,
				MaximumAdverseExcursion: market.MAE, CandleCount: market.Count,
				MarketDataSource: market.Source, TimestampSemantics: market.Semantics,
			}
			if mapping.Benchmark != "" {
				benchmark, found := index.outcome(mapping.Benchmark, *anchor.At, horizon, time.Duration(rules.MaximumIntradayAnchorDelayMinutes)*time.Minute)
				if found && absDuration(benchmark.EffectiveAnchor.Sub(market.EffectiveAnchor)) <= time.Hour {
					abnormal := market.RawReturn - benchmark.RawReturn
					absolute := math.Abs(abnormal)
					outcome.AbnormalReturn = &abnormal
					outcome.AbsoluteAbnormalReturn = &absolute
				}
			}
			result = append(result, outcome)
		}
	}
	return result
}

func absDuration(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}

func coverageRows(candles []Candle) []CoverageRow {
	type accumulator struct {
		row  CoverageRow
		last time.Time
	}
	values := map[string]*accumulator{}
	for _, candle := range candles {
		key := seriesKey(candle.Symbol, candle.Timeframe, candle.Source)
		item, ok := values[key]
		if !ok {
			item = &accumulator{row: CoverageRow{Symbol: candle.Symbol, Timeframe: candle.Timeframe, Source: candle.Source,
				TimestampSemantics: candle.TimestampSemantics, RegularTradingHours: candle.RegularTradingHours,
				MarketDataClassification: candle.MarketDataClassification, First: candle.Timestamp, Last: candle.Timestamp}}
			values[key] = item
		}
		if !item.last.IsZero() {
			gap := candle.Timestamp.Sub(item.last)
			if (candle.Timeframe == "1h" && gap > 4*24*time.Hour) || (candle.Timeframe == "1d" && tradingDateGap(item.last, candle.Timestamp) > 4*24*time.Hour) {
				item.row.GapCount++
			}
		}
		item.last = candle.Timestamp
		item.row.Count++
		if candle.Timestamp.Before(item.row.First) {
			item.row.First = candle.Timestamp
		}
		if candle.Timestamp.After(item.row.Last) {
			item.row.Last = candle.Timestamp
		}
	}
	result := make([]CoverageRow, 0, len(values))
	for _, item := range values {
		result = append(result, item.row)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Symbol != result[j].Symbol {
			return result[i].Symbol < result[j].Symbol
		}
		if result[i].Timeframe != result[j].Timeframe {
			return result[i].Timeframe < result[j].Timeframe
		}
		return result[i].Source < result[j].Source
	})
	return result
}
