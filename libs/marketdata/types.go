package marketdata

import "time"

// Quote represents a real-time or delayed market quote
type Quote struct {
	Symbol    string
	Price     float64
	Bid       float64
	Ask       float64
	BidSize   int64
	AskSize   int64
	Volume    int64
	Timestamp time.Time
	Exchange  string
}

// Candle represents OHLCV data for a given timeframe
type Candle struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int64
	VWAP      float64 // Volume-weighted average price
}

// Trade represents an individual trade execution
type Trade struct {
	Symbol    string
	Price     float64
	Size      int64
	Timestamp time.Time
	Exchange  string
	Conditions []string
}

// Earnings represents earnings report data
type Earnings struct {
	Symbol          string
	FiscalQuarter   string
	FiscalYear      int
	ReportDate      time.Time
	EPS             float64
	EPSEstimate     float64
	Revenue         float64
	RevenueEstimate float64
}

// Timeframe represents supported candle timeframes
type Timeframe string

const (
	Timeframe1Min  Timeframe = "1"
	Timeframe5Min  Timeframe = "5"
	Timeframe15Min Timeframe = "15"
	Timeframe1Hour Timeframe = "60"
	Timeframe1Day  Timeframe = "1D"
	Timeframe1Week Timeframe = "1W"
)

// StreamUpdate represents a real-time data update
type StreamUpdate struct {
	Type      string // "quote", "trade", "bar"
	Quote     *Quote
	Trade     *Trade
	Candle    *Candle
	Timestamp time.Time
}
