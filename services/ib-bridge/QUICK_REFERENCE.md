# IB Bridge Quick Reference

## 🚀 Quick Start

```bash

# 1. Ensure IB Gateway is running (port 7497 for paper trading)

# 2. Start IB Bridge

docker compose up ib-bridge

# 3. Test connection

curl <<http://localhost:8092/health>>

```

## 📡 API Endpoints

### Health Check

```n
curl <<http://localhost:8092/health>>

```

### Get Quote

```n
curl <http://localhost:8092/quotes/AAPL>

```

### Get Historical Candles

```n
curl -X POST <<<http://localhost:8092/candles/AAPL>>> \
  -H "Content-Type: application/json" \
  -d '{
    "duration": "1 D",
    "bar_size": "5 mins",
    "what_to_show": "TRADES"
  }'

```

### Get Account Info

```n
curl <http://localhost:8092/account>

```

### Get Positions

```n
curl <http://localhost:8092/positions>

```

### Place Order (Use with caution!)

```n
curl -X POST <http://localhost:8092/orders> \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "AAPL",
    "action": "BUY",
    "quantity": 10,
    "order_type": "MKT"
  }'

```

## 🔧 Go Integration

### Using Provider Interface

```n
import "jax-trading-assistant/libs/marketdata/ib"

provider, _ := ib.NewProvider("<http://localhost:8092">)
defer provider.Close()

quote, _ := provider.GetQuote(ctx, "AAPL")
fmt.Printf("Price: $%.2f\n", quote.Price)

```

### Using Direct Client

```n
client := ib.NewClient(ib.Config{
    BaseURL: "<http://localhost:8092">,
})

health, _ := client.Health(ctx)
quote, _ := client.GetQuote(ctx, "AAPL")
candles, _ := client.GetCandles(ctx, "AAPL", &ib.CandlesRequest{
    Duration: "1 D",
    BarSize: "5 mins",
})

```

## 🐍 Python Integration

```python
from ib_client import IBClient

client = IBClient(host="127.0.0.1", port=7497)
await client.connect()

# Get quote

quote = await client.get_quote("AAPL")

# Get candles

candles = await client.get_candles("AAPL", "1 D", "5 mins")

# Get account

account = await client.get_account_info()

await client.disconnect()

```

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `IB_GATEWAY_HOST` | `127.0.0.1` | IB Gateway host |
| `IB_GATEWAY_PORT` | `7497` | Port (7497=paper, 7496=live) |
| `IB_CLIENT_ID` | `1` | Client ID |
| `AUTO_CONNECT` | `true` | Auto-connect on startup |
| `PAPER_TRADING` | `true` | Paper trading mode |
| `PORT` | `8092` | API server port |

### Docker Compose Override

```yaml

# docker-compose.override.yml

services:
  ib-bridge:
    environment:
      IB_GATEWAY_HOST: 192.168.1.100  # Your IB Gateway IP

      IB_GATEWAY_PORT: 7497
      LOG_LEVEL: DEBUG

```

## 🧪 Testing

### Python Test Suite

```n
cd services/ib-bridge
python test_bridge.py

```

### Go Test Client

```bash
cd services/ib-bridge/examples
go run test_go_client.go
```

### Manual Tests

```bash
# Health
curl http://localhost:8092/health

# Quotes
curl http://localhost:8092/quotes/AAPL
curl http://localhost:8092/quotes/MSFT
curl http://localhost:8092/quotes/GOOGL

# Candles (1 day, 1-minute bars)
curl -X POST http://localhost:8092/candles/AAPL \
  -H "Content-Type: application/json" \
  -d '{"duration": "1 D", "bar_size": "1 min"}'

# Candles (1 week, 1-hour bars)
curl -X POST http://localhost:8092/candles/AAPL \
  -H "Content-Type: application/json" \
  -d '{"duration": "1 W", "bar_size": "1 hour"}'
```

## 🔍 Monitoring

### View Logs

```bash
# Follow logs
docker compose logs -f ib-bridge

# Last 100 lines
docker compose logs --tail=100 ib-bridge

# Since 1 hour ago
docker compose logs --since 1h ib-bridge
```

### Check Status

```bash
# Container status
docker compose ps ib-bridge

# Health check
docker compose exec ib-bridge python -c "import requests; print(requests.get('http://localhost:8092/health').json())"
```

## 🐛 Troubleshooting

### Connection Refused

```powershell
# Check IB Gateway is running
netstat -an | findstr 7497

# Test connection
Test-NetConnection -ComputerName 127.0.0.1 -Port 7497
```

### Service Won't Start

```bash
# Check logs
docker compose logs ib-bridge

# Rebuild
docker compose build --no-cache ib-bridge
docker compose up ib-bridge
```

### Not Connected to IB Gateway

```bash
# Manual connect
curl -X POST http://localhost:8092/connect

# Check IB Gateway settings
# Configure → Settings → API → Settings
# - Enable ActiveX and Socket Clients
# - Port: 7497 (paper) or 7496 (live)
```

## 📊 Bar Sizes

| Value | Description |
| --------- | ----------- |
| `1 secs` | 1 second |
| `5 secs` | 5 seconds |
| `10 secs` | 10 seconds |
| `15 secs` | 15 seconds |
| `30 secs` | 30 seconds |
| `1 min` | 1 minute |
| `2 mins` | 2 minutes |
| `3 mins` | 3 minutes |
| `5 mins` | 5 minutes |
| `10 mins` | 10 minutes |
| `15 mins` | 15 minutes |
| `20 mins` | 20 minutes |
| `30 mins` | 30 minutes |
| `1 hour` | 1 hour |
| `2 hours` | 2 hours |
| `3 hours` | 3 hours |
| `4 hours` | 4 hours |
| `8 hours` | 8 hours |
| `1 day` | 1 day |
| `1 week` | 1 week |
| `1 month` | 1 month |

## 🕐 Durations

| Value | Description |
| ------ | ----------------------- |
| `60 S` | 60 seconds |
| `120 S` | 120 seconds |
| `1800 S` | 1800 seconds (30 mins) |
| `3600 S` | 3600 seconds (1 hour) |
| `1 D` | 1 day |
| `2 D` | 2 days |
| `1 W` | 1 week |
| `2 W` | 2 weeks |
| `1 M` | 1 month |
| `2 M` | 2 months |
| `1 Y` | 1 year |

## 🛑 Stop/Restart

```bash

# Stop

docker compose down ib-bridge

# Restart

docker compose restart ib-bridge

# Stop all services

docker compose down

# Start specific service

docker compose up -d ib-bridge

```

## 📚 Documentation

- **Main README**: `services/ib-bridge/README.md`
- **Testing Guide**: `services/ib-bridge/TESTING.md`
- **Go Client**: `libs/marketdata/ib/README.md`
- **Complete Summary**: `Docs/PHASE_3.md`

## 🔗 Useful Links

- IB Gateway Download: <https://www.interactivebrokers.com/en/trading/tws.php>
- ib_insync Docs: <https://ib-insync.readthedocs.io/>
- FastAPI Docs: <https://fastapi.tiangolo.com/>

## ⚠️ Safety Reminders

- ✅ Default configuration uses **paper trading**
- ✅ Port 7497 = paper, 7496 = live
- ✅ Test thoroughly before live trading
- ✅ Use read-only API mode when possible
- ✅ Never commit credentials to git
