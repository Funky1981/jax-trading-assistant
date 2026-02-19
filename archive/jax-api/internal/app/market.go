package app

import (
	"context"
	"time"
)

type Candle struct {
	TS     time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

type MarketData interface {
	GetDailyCandles(ctx context.Context, symbol string, limit int) ([]Candle, error)
}
