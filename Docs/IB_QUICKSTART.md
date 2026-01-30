# Interactive Brokers Quick Start

## TL;DR - What You Need to Know

**No API keys needed!** IB Gateway uses socket connections.

### Steps:

1. **Download & Install IB Gateway** from [interactivebrokers.com/api](https://www.interactivebrokers.com/en/trading/ib-api.php)

2. **Launch IB Gateway** and login with:
   - Your IB username/password
   - Select "Paper Trading" mode

3. **Enable API in Gateway**:
   - Configure → Settings → API → Settings
   - ✅ Enable "ActiveX and Socket Clients"
   - Port: `7497` (paper) or `7496` (live)
   - Trusted IPs: `127.0.0.1`

4. **Configure jax-trading-assistant**:
   ```json
   "ib": {
     "enabled": true,
     "host": "127.0.0.1",
     "port": 7497,
     "client_id": 1
   }
   ```

5. **Run**:
   ```powershell
   go run services/jax-market/cmd/jax-market/main.go -config config/jax-ingest-ib.json
   ```

## Connection Ports

| Mode          | Port | Description                  |
|---------------|------|------------------------------|
| Paper Trading | 7497 | Safe testing, no real money  |
| Live Trading  | 7496 | Real money - be careful!     |

## Common Errors

| Error | Meaning | Fix |
|-------|---------|-----|
| 502 | Gateway not running | Start IB Gateway |
| Connection refused | Port mismatch | Check port in Gateway settings |
| No market data | Missing subscription | For paper trading, this is normal (delayed data) |

## Full Documentation

See [IB_GATEWAY_SETUP.md](./IB_GATEWAY_SETUP.md) for complete instructions.

## Current Status

⚠️ **Implementation Status: Skeleton Only**

The IB provider is currently a **placeholder**. To complete it:

1. Install: `go get github.com/gofinance/ib`
2. Implement actual socket connection in `libs/marketdata/provider_ib.go`
3. Add real-time quote fetching
4. Add historical data fetching

**Alternative**: Consider using IB's official Python API and creating a microservice.
