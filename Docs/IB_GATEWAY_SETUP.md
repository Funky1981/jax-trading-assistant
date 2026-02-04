# Interactive Brokers Gateway Integration

This guide explains how to connect your jax-trading-assistant to Interactive Brokers Gateway for paper trading or live trading.

## Prerequisites

1. **Interactive Brokers Account**
   - Sign up at [interactivebrokers.com](https://www.interactivebrokers.com)
   - For testing, create a **Paper Trading account** (free, no real money)

2. **IB Gateway or TWS Installed**
   - Download from [IB API Downloads](https://www.interactivebrokers.com/en/trading/ib-api.php)
   - **IB Gateway** is lightweight, no UI (recommended for automated trading)
   - **TWS (Trader Workstation)** is the full desktop app

## How IB Gateway Authentication Works

**No API keys required!** IB Gateway uses a different authentication model:

1. **Login once** through the Gateway GUI with your IB username/password
2. **The Gateway runs locally** on your machine (default: `localhost`)
3. **Your trading app connects** via TCP socket to the Gateway
4. **Authentication is handled** by the Gateway - your app just connects to the socket

## Connection Settings

### Paper Trading (Recommended for Testing)
- **Host**: `127.0.0.1` (localhost)
- **Port**: `7497` (default paper trading port)
- **Client ID**: Any integer (e.g., `1`)

### Live Trading
- **Host**: `127.0.0.1`
- **Port**: `7496` (default live trading port)
- **Client ID**: Any integer (e.g., `1`)

### Custom Configuration

You can change these in IB Gateway settings:
1. Launch IB Gateway
2. Go to **Configure > Settings > API > Settings**
3. Change socket port if needed
4. Enable "ActiveX and Socket Clients"

## Setup Steps

### 1. Start IB Gateway

**Windows:**

```

# Navigate to IB Gateway install directory (usually):

cd "C:\Jts\ibgateway\<version>"

# Run the gateway

.\ibgateway.exe

```

**Login with your credentials:**
- Username: Your IB username
- Password: Your IB password
- Trading Mode: Select "Paper Trading" for testing

### 2. Enable API Access in Gateway

1. In IB Gateway, go to **Configure > Settings > API**
2. **Enable "ActiveX and Socket Clients"**
3. **Socket Port**: Confirm it shows `7497` for paper trading
4. **Trusted IPs**: Add `127.0.0.1` if not already there
5. **Read-Only API**: Leave unchecked (you want to place orders)
6. Click **OK** and **Apply**

### 3. Configure jax-trading-assistant

Add IB provider to your `config/jax-ingest.json`:

```json
{
  "database_dsn": "postgresql://jax:jax@postgres:5432/jax",
  "ingest_interval": 300,
  "symbols": ["AAPL", "MSFT", "GOOGL", "SPY", "QQQ"],
  "polygon": {
    "enabled": false,
    "api_key": "",
    "tier": "free"
  },
  "alpaca": {
    "enabled": false,
    "api_key": "",
    "api_secret": "",
    "tier": "free"
  },
  "ib": {
    "enabled": true,
    "host": "127.0.0.1",
    "port": 7497,
    "client_id": 1
  },
  "cache": {
    "enabled": true,
    "redis_url": "redis:6379",
    "ttl": 300
  }
}

```

Or configure via code in `services/jax-market/cmd/jax-market/main.go`:

```go
// Add IB provider
if cfg.IB.Enabled {
    mdConfig.Providers = append(mdConfig.Providers, marketdata.ProviderConfig{
        Name:       marketdata.ProviderIB,
        IBHost:     cfg.IB.Host,     // "127.0.0.1"
        IBPort:     cfg.IB.Port,     // 7497 for paper
        IBClientID: cfg.IB.ClientID, // 1
        Priority:   1,
        Enabled:    true,
    })
}

```

### 4. Run Your Trading Assistant

```powershell

# Start just the database

docker compose up -d postgres

# Run the market data service

go run services/jax-market/cmd/jax-market/main.go -config config/jax-ingest.json

# Or start the full stack

.\start.ps1

```

## Verifying Connection

### Check IB Gateway Status

In IB Gateway window:
- Look for connection message: "Client 1 connected" or similar
- Check Activity Log for incoming requests

### Test with Simple Script

```go
package main

import (
    "context"
    "fmt"
    "log"

    "jax-trading-assistant/libs/marketdata"
)

func main() {
    config := &marketdata.Config{
        Providers: []marketdata.ProviderConfig{
            {
                Name:       marketdata.ProviderIB,
                IBHost:     "127.0.0.1",
                IBPort:     7497, // Paper trading
                IBClientID: 1,
                Priority:   1,
                Enabled:    true,
            },
        },
    }

    client, err := marketdata.NewClient(config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // Test quote fetch
    quote, err := client.GetQuote(context.Background(), "AAPL")
    if err != nil {
        log.Fatalf("Failed to get quote: %v", err)
    }

    fmt.Printf("AAPL: $%.2f at %s\n", quote.Price, quote.Timestamp)
}

```

## Troubleshooting

### Error: "Connection refused" or "502"

**Cause**: IB Gateway is not running or not listening on the port

**Solutions**:
1. Make sure IB Gateway is running and logged in
2. Check the port in Gateway settings matches your config
3. Verify "ActiveX and Socket Clients" is enabled
4. Ensure `127.0.0.1` is in Trusted IPs

### Error: "Can't connect to IB"

**Cause**: Firewall or antivirus blocking the connection

**Solutions**:
1. Add exception for IB Gateway in Windows Firewall
2. Temporarily disable antivirus and test
3. Check if port 7497 is open: `netstat -an | findstr 7497`

### Error: "No market data permissions"

**Cause**: Your IB account doesn't have market data subscriptions

**Solutions**:
1. For **paper trading**: No subscriptions needed, but data may be delayed
2. For **live trading**: Subscribe to market data in Account Management
3. Check IB account permissions at [Account Management](https://www.interactivebrokers.com/portal)

### Gateway Disconnects After Idle Time

**Cause**: IB Gateway has auto-logoff for security

**Solutions**:
1. In Gateway: **Configure > Settings > Lock and Exit**
2. Increase "Auto logoff time" (default is often 24 hours)
3. Or implement reconnection logic in your app

## Next Steps

### To Complete IB Integration (TODO):

The current implementation is a **skeleton**. To make it fully functional:

1. **Install Go IB library**:
   ```powershell
   go get github.com/gofinance/ib
   ```

2. **Complete `provider_ib.go`**:
   - Import the `github.com/gofinance/ib` package
   - Implement actual socket connection in `NewIBProvider`
   - Implement `GetQuote` using IB market data requests
   - Implement `GetCandles` using historical data requests
   - Implement `StreamQuotes` for real-time data

3. **Alternative: Use Python TWS API**:
   - IB officially supports Python better than Go
   - Consider creating a Python microservice for IB connectivity
   - Expose REST API that jax-trading-assistant can call

### Resources

- [IB API Documentation](https://interactivebrokers.github.io/tws-api/)
- [gofinance/ib Library](https://github.com/gofinance/ib)
- [IB API Downloads](https://www.interactivebrokers.com/en/trading/ib-api.php)
- [IB Paper Trading Guide](https://www.interactivebrokers.com/en/index.php?f=1286)

## Security Notes

1. **Never commit credentials** - IB login happens in Gateway GUI, not in code
2. **Use paper trading** for development and testing
3. **Localhost only** - Don't expose IB Gateway to the internet
4. **Trusted IPs** - Only allow connections from `127.0.0.1` unless needed
5. **Auto-logoff** - Configure appropriate timeout in Gateway settings
