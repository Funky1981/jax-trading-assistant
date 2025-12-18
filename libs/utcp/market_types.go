package utcp

import "time"

const (
	MarketDataProviderID   = "market-data"
	ToolMarketGetQuote     = "market.get_quote"
	ToolMarketGetCandles   = "market.get_candles"
	ToolMarketGetEarnings  = "market.get_earnings"
	DefaultCandleTimeframe = "1D"
)

type GetQuoteInput struct {
	Symbol string `json:"symbol"`
}

type GetQuoteOutput struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency"`
	Timestamp time.Time `json:"timestamp"`
}

type GetCandlesInput struct {
	Symbol    string `json:"symbol"`
	Timeframe string `json:"timeframe"`
	Limit     int    `json:"limit"`
}

type Candle struct {
	TS     time.Time `json:"ts"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume int64     `json:"volume"`
}

type GetCandlesOutput struct {
	Symbol    string   `json:"symbol"`
	Timeframe string   `json:"timeframe"`
	Candles   []Candle `json:"candles"`
}

type GetEarningsInput struct {
	Symbol string `json:"symbol"`
	Limit  int    `json:"limit"`
}

type EarningsEntry struct {
	Date        string  `json:"date"`
	EPSActual   float64 `json:"eps_actual"`
	EPSEstimate float64 `json:"eps_estimate"`
	SurprisePct float64 `json:"surprise_pct"`
}

type GetEarningsOutput struct {
	Symbol   string          `json:"symbol"`
	Earnings []EarningsEntry `json:"earnings"`
}
