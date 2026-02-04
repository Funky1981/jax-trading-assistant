# Interactive Brokers Bridge - Testing Guide

This guide will help you test the IB Bridge service and verify the connection to Interactive Brokers.

## Prerequisites

1. **IB Gateway Running**: Ensure IB Gateway is running on your local machine
   - Download from: <https://www.interactivebrokers.com/en/trading/tws.php>
   - Run IB Gateway (not TWS for production use)
   - Default ports:
     - Paper Trading: 7497
     - Live Trading: 7496

2. **API Access Enabled**:
   - In IB Gateway, go to: Configure → Settings → API → Settings
   - Check "Enable ActiveX and Socket Clients"
   - Check "Allow connections from localhost only" (for security)
   - Set "Socket port" to 7497 (paper) or 7496 (live)
   - Check "Read-Only API"

## Quick Test (Without Docker)

### 1. Start IB Gateway

```bash

# Start IB Gateway manually

# Login with your paper trading credentials

```

### 2. Run IB Bridge Locally

```n
cd services/ib-bridge

# Install dependencies

pip install -r requirements.txt

# Copy environment file

cp .env.example .env

# Edit .env if needed (defaults should work)

# Ensure IB_GATEWAY_PORT=7497 for paper trading

# Run the service

python main.py

```

### 3. Test Endpoints

```bash

# Health check

curl <<http://localhost:8092/health>>

# Expected output:

# {"status":"healthy","connected":true,"version":"1.0.0"}

# Get a quote

curl <<http://localhost:8092/quotes/AAPL>>

# Expected output:

# {

#   "symbol": "AAPL",

#   "price": 150.25,

#   "bid": 150.24,

#   "ask": 150.26,

#   "bid_size": 100,

#   "ask_size": 200,

#   "volume": 45678900,

#   "timestamp": "2026-02-04T15:30:00",

#   "exchange": "SMART"

# }

# Get historical candles

curl -X POST <http://localhost:8092/candles/AAPL> \
  -H "Content-Type: application/json" \
  -d '{
    "duration": "1 D",
    "bar_size": "1 min",
    "what_to_show": "TRADES"
  }'

# Get account info

curl <http://localhost:8092/account>

# Get positions

curl <http://localhost:8092/positions>

```

## Docker Test

### 1. Update .env File (optional)

Create or update `.env` in the project root:

```bash

# IB Bridge Configuration

IB_GATEWAY_HOST=host.docker.internal
IB_GATEWAY_PORT=7497
IB_CLIENT_ID=1
IB_AUTO_CONNECT=true
IB_PAPER_TRADING=true
IB_LOG_LEVEL=INFO

```

### 2. Start IB Bridge Service

```bash

# From project root

docker compose up ib-bridge

# Or run in background

docker compose up -d ib-bridge

# View logs

docker compose logs -f ib-bridge

```

### 3. Check Logs

Look for these messages in the logs:

```text
INFO - Starting IB Bridge service...
INFO - Connected to IB Gateway at host.docker.internal:7497 (client_id=1)
INFO - Auto-connected to IB Gateway
INFO - Uvicorn running on <http://0.0.0.0:8092>

```

### 4. Test from Host Machine

```bash

# Health check

curl <<http://localhost:8092/health>>

# Get quote

curl <<http://localhost:8092/quotes/AAPL>>

# Get candles

curl -X POST <http://localhost:8092/candles/MSFT> \
  -H "Content-Type: application/json" \
  -d '{
    "duration": "1 D",
    "bar_size": "5 mins",
    "what_to_show": "TRADES"
  }'

```

## Test Go Client

Create a test file: `test_ib_client.go`

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "jax-trading-assistant/libs/marketdata"
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

    // Test 1: Get Quote
    fmt.Println("=== Test 1: Get Quote ===")
    quote, err := provider.GetQuote(ctx, "AAPL")
    if err != nil {
        log.Fatalf("Failed to get quote: %v", err)
    }
    fmt.Printf("Symbol: %s\n", quote.Symbol)
    fmt.Printf("Price: $%.2f\n", quote.Price)
    fmt.Printf("Bid: $%.2f x %d\n", quote.Bid, quote.BidSize)
    fmt.Printf("Ask: $%.2f x %d\n", quote.Ask, quote.AskSize)
    fmt.Printf("Volume: %d\n", quote.Volume)
    fmt.Printf("Exchange: %s\n", quote.Exchange)

    // Test 2: Get Candles
    fmt.Println("\n=== Test 2: Get Candles ===")
    to := time.Now()
    from := to.Add(-24 * time.Hour)
    candles, err := provider.GetCandles(ctx, "AAPL", marketdata.Timeframe5Min, from, to)
    if err != nil {
        log.Fatalf("Failed to get candles: %v", err)
    }
    fmt.Printf("Retrieved %d candles\n", len(candles))
    if len(candles) > 0 {
        last := candles[len(candles)-1]
        fmt.Printf("Last candle: O=%.2f H=%.2f L=%.2f C=%.2f V=%d\n",
            last.Open, last.High, last.Low, last.Close, last.Volume)
    }

    // Test 3: Direct Client Access
    fmt.Println("\n=== Test 3: Direct Client Access ===")
    client := ib.NewClient(ib.Config{
        BaseURL: "<http://localhost:8092">,
    })

    health, err := client.Health(ctx)
    if err != nil {
        log.Fatalf("Health check failed: %v", err)
    }
    fmt.Printf("Status: %s\n", health.Status)
    fmt.Printf("Connected: %v\n", health.Connected)
    fmt.Printf("Version: %s\n", health.Version)

    fmt.Println("\n✅ All tests passed!")
}

```

Run the test:

```bash
go run test_ib_client.go

```

## WebSocket Test

Test streaming quotes using `wscat`:

```bash

# Install wscat if needed

npm install -g wscat

# Connect to stream

wscat -c ws://localhost:8092/ws/quotes/AAPL

# You should see real-time quote updates every second

```

Or use Python:

```python
import asyncio
import websockets
import json

async def stream_quotes():
    uri = "ws://localhost:8092/ws/quotes/AAPL"
    async with websockets.connect(uri) as websocket:
        for _ in range(10):  # Receive 10 updates

            message = await websocket.recv()
            data = json.loads(message)
            print(f"Price: ${data['price']:.2f}, "
                  f"Bid: ${data['bid']:.2f}, "
                  f"Ask: ${data['ask']:.2f}")

asyncio.run(stream_quotes())

```

## Troubleshooting

### Connection Refused

**Problem**: `Connection refused` when accessing IB Bridge

**Solution**:

1. Ensure IB Gateway is running
2. Check IB Gateway port (should be 7497 for paper trading)
3. Verify API access is enabled in IB Gateway settings
4. Check firewall settings

### Not Connected to IB Gateway

**Problem**: Health check shows `"connected": false`

**Solution**:

1. Check IB Bridge logs: `docker compose logs ib-bridge`
2. Verify IB Gateway is logged in (not just started)
3. Check client ID isn't already in use
4. Try connecting manually: `curl -X POST <http://localhost:8092/connect`>

### Docker Networking Issues

**Problem**: IB Bridge can't reach IB Gateway from Docker

**Solution**:

1. On Windows/Mac: Use `host.docker.internal` as `IB_GATEWAY_HOST`
2. On Linux: Add `--add-host=host.docker.internal:host-gateway` or use `172.17.0.1`
3. Check `extra_hosts` in docker-compose.yml

### Market Data Issues

**Problem**: Getting empty or zero values in quotes

**Solution**:

1. Check if market is open (use delayed data type)
2. Verify you have market data subscriptions for the symbol
3. Check IB Gateway logs for permission errors
4. Try a different symbol (e.g., SPY, QQQ)

### Port Already in Use

**Problem**: Port 8092 already in use

**Solution**:

```bash

# Change port in docker-compose.yml

ports:
  - "8093:8092"  # Use different external port

# Or stop the conflicting service

lsof -ti:8092 | xargs kill -9  # Unix/Mac

netstat -ano | findstr :8092   # Windows (then kill PID)

```

## Integration Test Checklist

- [ ] IB Gateway is running and logged in
- [ ] API access is enabled in IB Gateway
- [ ] IB Bridge service starts without errors
- [ ] Health endpoint returns `connected: true`
- [ ] Can retrieve real-time quote for AAPL
- [ ] Can retrieve historical candles
- [ ] Can get account information
- [ ] Can get positions
- [ ] WebSocket streaming works
- [ ] Go client can connect and retrieve data
- [ ] Circuit breaker handles errors gracefully

## Next Steps

Once all tests pass:

1. **Update Go Services**: Modify `jax-api` to use the IB provider
2. **Configure Strategies**: Update strategy configs to use IB data
3. **Add Monitoring**: Set up alerts for connection failures
4. **Production Setup**: Configure for live trading (with extreme caution!)

## Safety Reminders

⚠️ **IMPORTANT**:

- Always test with **paper trading** first
- Default configuration uses port 7497 (paper trading)
- Never commit IB credentials to version control
- Use read-only API mode when possible
- Test order placement extensively in paper trading before going live
