package worldmonitorintelligence

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const MarketReactionAlgorithmV1 = "jax.world_monitor_market_reaction/v1"

type MarketPoint struct {
	Instrument string
	At         time.Time
	Price      float64
}

type ReactionState string

const (
	ReactionPositive ReactionState = "POSITIVE"
	ReactionNegative ReactionState = "NEGATIVE"
	ReactionFlat     ReactionState = "FLAT"
	ReactionUnknown  ReactionState = "UNKNOWN"
)

type MarketReaction struct {
	Algorithm      string
	Instrument     string
	EventAt        time.Time
	BeforeAt       time.Time
	AfterAt        time.Time
	BeforePrice    float64
	AfterPrice     float64
	WindowReturn   float64
	BaselineReturn *float64
	AbnormalReturn *float64
	State          ReactionState
	Unknowns       []string
}

// CorrelateMarketReaction selects the nearest valid before/after points for
// each instrument around the cluster event time. It is read-only and reports
// absent points or baselines as unknowns.
func CorrelateMarketReaction(cluster EventCluster, points []MarketPoint, baselines map[string]float64, window time.Duration) ([]MarketReaction, error) {
	if len(cluster.Members) == 0 || window <= 0 {
		return nil, fmt.Errorf("cluster and positive market-reaction window are required")
	}
	eventAt := clusterEventTime(cluster)
	if eventAt.IsZero() {
		return nil, fmt.Errorf("cluster has no usable event time")
	}
	byInstrument := map[string][]MarketPoint{}
	for _, point := range points {
		if point.Instrument == "" || point.At.IsZero() || point.Price <= 0 || math.IsNaN(point.Price) || math.IsInf(point.Price, 0) {
			continue
		}
		byInstrument[point.Instrument] = append(byInstrument[point.Instrument], point)
	}
	instruments := make([]string, 0, len(byInstrument))
	for instrument := range byInstrument {
		instruments = append(instruments, instrument)
	}
	sort.Strings(instruments)
	result := make([]MarketReaction, 0, len(instruments))
	for _, instrument := range instruments {
		var before, after *MarketPoint
		for index := range byInstrument[instrument] {
			point := byInstrument[instrument][index]
			delta := point.At.Sub(eventAt)
			if delta <= 0 && delta >= -window && (before == nil || point.At.After(before.At)) {
				copyPoint := point
				before = &copyPoint
			}
			if delta >= 0 && delta <= window && (after == nil || point.At.Before(after.At)) {
				copyPoint := point
				after = &copyPoint
			}
		}
		reaction := MarketReaction{Algorithm: MarketReactionAlgorithmV1, Instrument: instrument, EventAt: eventAt, State: ReactionUnknown}
		if before == nil || after == nil {
			reaction.Unknowns = []string{"usable before and after market points are not both available"}
			result = append(result, reaction)
			continue
		}
		reaction.BeforeAt, reaction.AfterAt = before.At, after.At
		reaction.BeforePrice, reaction.AfterPrice = before.Price, after.Price
		reaction.WindowReturn = after.Price/before.Price - 1
		if baseline, exists := baselines[instrument]; exists {
			baselineCopy := baseline
			abnormal := reaction.WindowReturn - baseline
			reaction.BaselineReturn, reaction.AbnormalReturn = &baselineCopy, &abnormal
		} else {
			reaction.Unknowns = []string{"baseline return is unavailable; reaction is descriptive only"}
		}
		switch {
		case reaction.WindowReturn > 0.001:
			reaction.State = ReactionPositive
		case reaction.WindowReturn < -0.001:
			reaction.State = ReactionNegative
		default:
			reaction.State = ReactionFlat
		}
		result = append(result, reaction)
	}
	return result, nil
}

func clusterEventTime(cluster EventCluster) time.Time {
	for _, member := range cluster.Members {
		if member.ObservedAt != nil {
			return member.ObservedAt.UTC()
		}
		if member.PublishedAt != nil {
			return member.PublishedAt.UTC()
		}
		if !member.CollectedAt.IsZero() {
			return member.CollectedAt.UTC()
		}
	}
	return time.Time{}
}

type InstrumentLink struct {
	Instrument string
	Reason     string
}

const InstrumentLinkAlgorithmV1 = "jax.world_monitor_instrument_link/v1"

// LinkPlausibleInstruments maps event taxonomy to watchlist instruments. It
// is intentionally a plausibility link, not a recommendation or order.
func LinkPlausibleInstruments(cluster EventCluster, extraction EntityExtraction) []InstrumentLink {
	sets := map[string][]InstrumentLink{
		"central_bank":     {{"TLT", "central-bank event taxonomy"}, {"SPY", "central-bank event taxonomy"}},
		"macro_rates":      {{"TLT", "rate-sensitive macro taxonomy"}, {"SPY", "rate-sensitive macro taxonomy"}, {"QQQ", "rate-sensitive macro taxonomy"}},
		"energy_oil":       {{"XLE", "energy taxonomy"}, {"USO", "energy taxonomy"}},
		"semiconductor_ai": {{"SOXX", "semiconductor taxonomy"}, {"QQQ", "technology taxonomy"}},
		"financial_credit": {{"XLF", "financial-credit taxonomy"}, {"SPY", "financial-credit taxonomy"}},
		"supply_chain":     {{"XLI", "supply-chain taxonomy"}, {"SPY", "supply-chain taxonomy"}},
		"geopolitical":     {{"SPY", "geopolitical taxonomy"}, {"XLE", "geopolitical taxonomy"}},
	}
	links := append([]InstrumentLink(nil), sets[cluster.EventType]...)
	if len(links) == 0 && len(extraction.Entities) > 0 {
		links = []InstrumentLink{{Instrument: "SPY", Reason: "broad issuer/entity watchlist fallback"}}
	}
	sort.Slice(links, func(left, right int) bool { return links[left].Instrument < links[right].Instrument })
	return links
}
