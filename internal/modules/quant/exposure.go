package quant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

const PortfolioExposureAlgorithmV1 = "jax.quant.portfolio_exposure/v1"

// Position is a valuation snapshot. Quantity is signed: positive is long and
// negative is short. It is descriptive input only and is never persisted or
// converted into an order.
type Position struct {
	Instrument string  `json:"instrument"`
	Quantity   float64 `json:"quantity"`
	Price      float64 `json:"price"`
}

type InstrumentExposure struct {
	Instrument       string  `json:"instrument"`
	MarketValue      float64 `json:"market_value"`
	AbsoluteExposure float64 `json:"absolute_exposure"`
	GrossWeight      float64 `json:"gross_weight"`
}

// PortfolioExposureResult embeds the standard versioned response and adds a
// stable, instrument-keyed breakdown for consumers that need more than the
// aggregate values.
type PortfolioExposureResult struct {
	Response    Response             `json:"response"`
	Instruments []InstrumentExposure `json:"instruments"`
}

func CalculatePortfolioExposure(dataset FrozenDataset, positions []Position) (PortfolioExposureResult, error) {
	if len(positions) == 0 {
		return PortfolioExposureResult{}, fmt.Errorf("portfolio exposure requires at least one position")
	}
	normalized := append([]Position(nil), positions...)
	for index, position := range normalized {
		if strings.TrimSpace(position.Instrument) == "" || !finite(position.Quantity) || position.Quantity == 0 || !finitePositive(position.Price) {
			return PortfolioExposureResult{}, fmt.Errorf("position %d requires a non-zero finite quantity, positive price, and instrument", index)
		}
	}
	sort.Slice(normalized, func(left, right int) bool {
		if normalized[left].Instrument != normalized[right].Instrument {
			return normalized[left].Instrument < normalized[right].Instrument
		}
		if normalized[left].Quantity != normalized[right].Quantity {
			return normalized[left].Quantity < normalized[right].Quantity
		}
		return normalized[left].Price < normalized[right].Price
	})
	positionBytes, _ := json.Marshal(normalized)
	positionDigest := sha256.Sum256(positionBytes)
	positionInput := InputReference{ID: "portfolio_positions", ContentSHA256: hex.EncodeToString(positionDigest[:])}
	request, err := newCalculationRequest(dataset, []InputReference{positionInput}, "portfolio_exposure", PortfolioExposureAlgorithmV1, []Parameter{
		{Name: "position_aggregation", Value: "same_instrument_sum_market_values"},
		{Name: "position_count", Value: fmt.Sprintf("%d", len(normalized))},
		{Name: "quantity_sign_definition", Value: "positive_long_negative_short"},
		{Name: "valuation_definition", Value: "signed_quantity_times_explicit_price"},
	})
	if err != nil {
		return PortfolioExposureResult{}, err
	}
	byInstrument := make(map[string]float64)
	for _, position := range normalized {
		marketValue := position.Quantity * position.Price
		if !finite(marketValue) {
			return PortfolioExposureResult{}, fmt.Errorf("position valuation is non-finite")
		}
		byInstrument[position.Instrument] += marketValue
	}
	instruments := make([]InstrumentExposure, 0, len(byInstrument))
	gross, net, longExposure, shortExposure := 0.0, 0.0, 0.0, 0.0
	for instrument, marketValue := range byInstrument {
		absolute := math.Abs(marketValue)
		gross += absolute
		net += marketValue
		if marketValue > 0 {
			longExposure += marketValue
		} else {
			shortExposure -= marketValue
		}
		instruments = append(instruments, InstrumentExposure{Instrument: instrument, MarketValue: marketValue, AbsoluteExposure: absolute})
	}
	if !finitePositive(gross) || !finite(net) || !finite(longExposure) || !finite(shortExposure) {
		return PortfolioExposureResult{}, fmt.Errorf("portfolio exposure produced a non-finite or zero gross exposure")
	}
	sort.Slice(instruments, func(left, right int) bool { return instruments[left].Instrument < instruments[right].Instrument })
	maxWeight := 0.0
	for index := range instruments {
		instruments[index].GrossWeight = instruments[index].AbsoluteExposure / gross
		if instruments[index].GrossWeight > maxWeight {
			maxWeight = instruments[index].GrossWeight
		}
	}
	values := []Value{
		{Metric: "gross_exposure", Value: gross, Unit: "account_currency"},
		{Metric: "net_exposure", Value: net, Unit: "account_currency"},
		{Metric: "long_exposure", Value: longExposure, Unit: "account_currency"},
		{Metric: "short_exposure", Value: shortExposure, Unit: "account_currency"},
		{Metric: "max_instrument_gross_weight", Value: maxWeight, Unit: "ratio"},
	}
	for _, instrument := range instruments {
		values = append(values,
			Value{Metric: "instrument_market_value/" + instrument.Instrument, Value: instrument.MarketValue, Unit: "account_currency"},
			Value{Metric: "instrument_gross_weight/" + instrument.Instrument, Value: instrument.GrossWeight, Unit: "ratio"},
		)
	}
	response, err := newCalculationResponse(request, values, nil)
	if err != nil {
		return PortfolioExposureResult{}, err
	}
	return PortfolioExposureResult{Response: response, Instruments: instruments}, nil
}
