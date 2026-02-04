# IB Bridge Service

Python bridge service that connects the Go backend to Interactive Brokers using `ib_insync`.

## Architecture

```text
Go (jax-api) <--> HTTP/WebSocket <--> IB Bridge (Python) <--> IB Gateway

```

## Features

- ✅ REST API for market data and order management
- ✅ WebSocket support for streaming real-time quotes
- ✅ Automatic reconnection with exponential backoff
- ✅ Health check endpoint
- ✅ Paper trading by default (safety first!)
- ✅ Comprehensive error handling and logging

## API Endpoints

### Connection Management

- `POST /connect` - Connect to IB Gateway
- `POST /disconnect` - Disconnect from IB Gateway
- `GET /health` - Health check

### Market Data

- `GET /quotes/{symbol}` - Get real-time quote
- `POST /candles/{symbol}` - Get historical candles
- `WS /ws/quotes/{symbol}` - Stream real-time quotes (WebSocket)

### Trading

- `POST /orders` - Place an order
- `GET /positions` - Get current positions
- `GET /account` - Get account information

## Quick Start

### Local Development

```bash

# Install dependencies

pip install -r requirements.txt

# Copy environment file

cp .env.example .env

# Edit .env with your IB Gateway settings

# Run the service

python main.py

```

### Docker

```bash

# Build

docker build -t ib-bridge .

# Run

docker run -p 8092:8092 \
  -e IB_GATEWAY_HOST=host.docker.internal \
  -e IB_GATEWAY_PORT=7497 \
  ib-bridge

```

### Docker Compose

```bash

# Start IB Bridge service

docker compose up ib-bridge

# Service will auto-connect to IB Gateway

```

## Configuration

Configure via environment variables (see `.env.example`):

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `HOST` | `0.0.0.0` | Server bind address |
| `PORT` | `8092` | Server port |
| `IB_GATEWAY_HOST` | `127.0.0.1` | IB Gateway host |
| `IB_GATEWAY_PORT` | `7497` | IB Gateway port (7497=paper, 7496=live) |
| `IB_CLIENT_ID` | `1` | IB client ID |
| `AUTO_CONNECT` | `true` | Auto-connect on startup |
| `PAPER_TRADING` | `true` | Enable paper trading mode |

## Safety Features

The service includes safety checks:

- Default to paper trading mode
- Validate port matches trading mode
- Cannot use live port (7496) when `PAPER_TRADING=true`
- Cannot use paper port (7497) when `PAPER_TRADING=false`

## Testing

```bash

# Check health

curl <http://localhost:8092/health>

# Get quote

curl <http://localhost:8092/quotes/AAPL>

# Get candles

curl -X POST <http://localhost:8092/candles/AAPL> \
  -H "Content-Type: application/json" \
  -d '{"duration": "1 D", "bar_size": "1 min"}'

# Stream quotes (WebSocket)

wscat -c ws://localhost:8092/ws/quotes/AAPL

```

## Troubleshooting

### Connection Issues

1. Ensure IB Gateway is running
2. Check IB Gateway port (7497 for paper, 7496 for live)
3. Verify API connections are enabled in IB Gateway settings
4. Check firewall settings

### Docker Networking

- Use `host.docker.internal` as `IB_GATEWAY_HOST` when running in Docker
- On Linux, add `--add-host=host.docker.internal:host-gateway` to docker run

## Dependencies

- `ib-insync` - IB API wrapper
- `fastapi` - Web framework
- `uvicorn` - ASGI server
- `pydantic` - Data validation
- `websockets` - WebSocket support
