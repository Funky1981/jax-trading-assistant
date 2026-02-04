# Market Data Providers

Production-grade market data ingestion supporting multiple providers (Polygon, Alpaca) with caching, WebSocket streaming, and fallback strategies.

## Features

- **Multi-Provider Support**: Polygon.io, Alpaca Market Data (easily extensible)
- **WebSocket Streaming**: Real-time quotes and trades
- **HTTP REST APIs**: Historical candles, earnings, fundamentals
- **Caching Layer**: Redis-backed cache to reduce API calls
- **Rate Limiting**: Automatic backoff and retry logic
- **Health Checks**: Provider availability monitoring
- **Fallback Strategy**: Automatic failover to backup provider

## Supported Providers

### Polygon.io
- **Free Tier**: 5 API calls/min, 15-minute delayed data
- **Starter**: $29/month, real-time data
- **Coverage**: US stocks, options, forex, crypto
- **WebSocket**: Real-time aggregates, trades, quotes

### Alpaca Market Data
- **Free Tier**: Unlimited, 15-minute delayed
- **Unlimited**: $9/month, real-time IEX
- **Coverage**: US stocks, crypto
- **WebSocket**: Real-time quotes, trades, bars

## Configuration

```json
{
  "providers": [
    {
      "name": "polygon",
      "apiKey": "${POLYGON_API_KEY}",
      "tier": "free|starter|developer",
      "priority": 1
    },
    {
      "name": "alpaca",
      "apiKey": "${ALPACA_API_KEY}",
      "apiSecret": "${ALPACA_API_SECRET}",
      "tier": "free|unlimited",
      "priority": 2
    }
  ],
  "cache": {
    "enabled": true,
    "redis": "localhost:6379",
    "ttl": "5m"
  },
  "symbols": ["AAPL", "MSFT", "GOOGL", "AMZN", "SPY", "QQQ"]
}

```

## Usage

### REST API Quotes

```go
import "jax-trading-assistant/libs/marketdata"

config := &marketdata.Config{
    Providers: []marketdata.ProviderConfig{
        {Name: "polygon", APIKey: os.Getenv("POLYGON_API_KEY"), Priority: 1},
    },
}

client, _ := marketdata.NewClient(config)
quote, _ := client.GetQuote(ctx, "AAPL")
fmt.Printf("AAPL: $%.2f\n", quote.Price)

```

### WebSocket Streaming

```go
stream, _ := client.StreamQuotes(ctx, []string{"AAPL", "MSFT"})

for quote := range stream {
    fmt.Printf("%s: $%.2f\n", quote.Symbol, quote.Price)
}

```

### Historical Candles

```go
candles, _ := client.GetCandles(ctx, "SPY", "1D", 30)
for _, c := range candles {
    fmt.Printf("%s: O=%.2f H=%.2f L=%.2f C=%.2f\n", 
        c.Timestamp, c.Open, c.High, c.Low, c.Close)
}

```

## Installation

```powershell

# Install dependencies

go get github.com/polygon-io/client-go
go get github.com/alpacahq/alpaca-trade-api-go/v3
go get github.com/redis/go-redis/v9

```

## Environment Variables

```powershell
$env:POLYGON_API_KEY = "your-polygon-key"
$env:ALPACA_API_KEY = "your-alpaca-key"
$env:ALPACA_API_SECRET = "your-alpaca-secret"
$env:REDIS_URL = "localhost:6379"

```
