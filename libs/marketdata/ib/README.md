# IB Bridge Go Client

Go client library for the IB Bridge Python service.

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "jax-trading-assistant/libs/marketdata/ib"
)

func main() {
    // Create IB provider
    provider, err := ib.NewProvider("<http://localhost:8092">)
    if err != nil {
        log.Fatalf("Failed to create IB provider: %v", err)
    }
    defer provider.Close()
    
    ctx := context.Background()
    
    // Get quote
    quote, err := provider.GetQuote(ctx, "AAPL")
    if err != nil {
        log.Fatalf("Failed to get quote: %v", err)
    }
    
    fmt.Printf("AAPL: $%.2f (bid: $%.2f, ask: $%.2f)\n", 
        quote.Price, quote.Bid, quote.Ask)
    
    // Get candles
    to := time.Now()
    from := to.Add(-24 * time.Hour)
    
    candles, err := provider.GetCandles(ctx, "AAPL", 
        marketdata.Timeframe1Min, from, to)
    if err != nil {
        log.Fatalf("Failed to get candles: %v", err)
    }
    
    fmt.Printf("Retrieved %d candles\n", len(candles))
}

```

## Direct Client Usage

For more control, use the HTTP client directly:

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "jax-trading-assistant/libs/marketdata/ib"
)

func main() {
    client := ib.NewClient(ib.Config{
        BaseURL: "<http://localhost:8092">,
    })
    
    ctx := context.Background()
    
    // Health check
    health, err := client.Health(ctx)
    if err != nil {
        log.Fatalf("Health check failed: %v", err)
    }
    fmt.Printf("Status: %s, Connected: %v\n", health.Status, health.Connected)
    
    // Get quote
    quote, err := client.GetQuote(ctx, "AAPL")
    if err != nil {
        log.Fatalf("Failed to get quote: %v", err)
    }
    fmt.Printf("Price: $%.2f\n", quote.Price)
    
    // Get candles
    candles, err := client.GetCandles(ctx, "AAPL", &ib.CandlesRequest{
        Duration:   "1 D",
        BarSize:    "1 min",
        WhatToShow: "TRADES",
    })
    if err != nil {
        log.Fatalf("Failed to get candles: %v", err)
    }
    fmt.Printf("Candles: %d\n", candles.Count)
    
    // Place order (use with caution!)
    order, err := client.PlaceOrder(ctx, &ib.OrderRequest{
        Symbol:    "AAPL",
        Action:    "BUY",
        Quantity:  10,
        OrderType: "MKT",
    })
    if err != nil {
        log.Fatalf("Failed to place order: %v", err)
    }
    fmt.Printf("Order placed: %d\n", order.OrderID)
}

```

## Features

- ✅ Implements `marketdata.Provider` interface
- ✅ Circuit breaker for resilience
- ✅ Configurable timeouts
- ✅ Automatic error handling
- ✅ Type-safe API

## Configuration

The client uses sensible defaults but can be configured:

```go
client := ib.NewClient(ib.Config{
    BaseURL: "<http://ib-bridge:8092",>
    Timeout: 30 * time.Second,
})

```
