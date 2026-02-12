package ib

import "time"

// ConnectRequest is the request to connect to IB Gateway
type ConnectRequest struct {
	Host     *string `json:"host,omitempty"`
	Port     *int    `json:"port,omitempty"`
	ClientID *int    `json:"client_id,omitempty"`
}

// ConnectResponse is the response from connection attempt
type ConnectResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// QuoteResponse represents real-time quote data
type QuoteResponse struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	BidSize   int64   `json:"bid_size"`
	AskSize   int64   `json:"ask_size"`
	Volume    int64   `json:"volume"`
	Timestamp string  `json:"timestamp"`
	Exchange  string  `json:"exchange"`
}

// Candle represents OHLCV candle data
type Candle struct {
	Timestamp string  `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    int64   `json:"volume"`
	VWAP      float64 `json:"vwap"`
}

// CandlesRequest is the request for historical candles
type CandlesRequest struct {
	Duration   string `json:"duration"`     // e.g., "1 D", "1 W", "1 M"
	BarSize    string `json:"bar_size"`     // e.g., "1 min", "5 mins", "1 hour"
	WhatToShow string `json:"what_to_show"` // e.g., "TRADES", "MIDPOINT", "BID", "ASK"
}

// CandlesResponse is the response with historical candles
type CandlesResponse struct {
	Symbol  string   `json:"symbol"`
	Candles []Candle `json:"candles"`
	Count   int      `json:"count"`
}

// OrderRequest is the request to place an order
type OrderRequest struct {
	Symbol     string   `json:"symbol"`
	Action     string   `json:"action"` // "BUY" or "SELL"
	Quantity   int      `json:"quantity"`
	OrderType  string   `json:"order_type"` // "MKT", "LMT", "STP"
	LimitPrice *float64 `json:"limit_price,omitempty"`
	StopPrice  *float64 `json:"stop_price,omitempty"`
}

// OrderResponse is the response from order placement
type OrderResponse struct {
	Success bool   `json:"success"`
	OrderID int    `json:"order_id"`
	Message string `json:"message"`
}

// OrderStatusResponse is the response for order status
type OrderStatusResponse struct {
	OrderID      int     `json:"order_id"`
	Status       string  `json:"status"`
	FilledQty    int     `json:"filled_qty"`
	AvgFillPrice float64 `json:"avg_fill_price"`
	LastUpdate   string  `json:"last_update"`
}

// Position represents position information
type Position struct {
	Symbol      string  `json:"symbol"`
	Quantity    int     `json:"quantity"`
	AvgCost     float64 `json:"avg_cost"`
	MarketValue float64 `json:"market_value"`
	Account     string  `json:"account"`
}

// PositionsResponse is the response with all positions
type PositionsResponse struct {
	Positions []Position `json:"positions"`
	Count     int        `json:"count"`
}

// AccountResponse represents account information
type AccountResponse struct {
	AccountID      string  `json:"account_id"`
	NetLiquidation float64 `json:"net_liquidation"`
	TotalCash      float64 `json:"total_cash"`
	BuyingPower    float64 `json:"buying_power"`
	EquityWithLoan float64 `json:"equity_with_loan"`
	Currency       string  `json:"currency"`
}

// HealthResponse is the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Connected bool   `json:"connected"`
	Version   string `json:"version"`
}

// Helper functions to create pointer values for optional fields

// StringPtr returns a pointer to a string
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to an int
func IntPtr(i int) *int {
	return &i
}

// Float64Ptr returns a pointer to a float64
func Float64Ptr(f float64) *float64 {
	return &f
}

// ParseTime parses an ISO timestamp
func ParseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
