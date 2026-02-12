# Interactive Brokers (IB) Guide

## Purpose

This guide consolidates IB Gateway setup, configuration, and the IB bridge integration.

## Quick Start (Paper Trading)

1. **Install IB Gateway or TWS**
   - Download: <https://www.interactivebrokers.com/en/trading/ib-api.php>
2. **Login (Paper Trading)**
   - Use your IB paper credentials.
3. **Enable API Access**
   - Configure → Settings → API → Settings
   - Enable **ActiveX and Socket Clients**
   - **IB Gateway (paper)**: `4002` (recommended)
   - **TWS (paper)**: `7497` (legacy)
   - Trusted IPs: `127.0.0.1`
4. **Start the IB Bridge (Python)**
   - `docker compose up ib-bridge`
   - Health check: `curl http://localhost:8092/health`

## Connection Settings

| Mode | Host | Port | Notes |
| --- | --- | --- | --- |
| Paper (IB Gateway) | 127.0.0.1 | 4002 | Recommended for testing |
| Paper (TWS) | 127.0.0.1 | 7497 | Legacy paper port |
| Live | 127.0.0.1 | 7496 | Use with caution |

## Configuration Example

```json
{
  "ib": {
    "enabled": true,
    "host": "host.docker.internal",
    "port": 4002,
    "client_id": 1
  }
}
```

## Current Implementation Status

- **IB bridge**: production-ready (FastAPI + ib_insync) and exposed on `8092` for trading/account APIs.
- **Go provider**: `jax-market` connects directly to IB Gateway for market data ingestion.
- **Ingestion wiring**: validated via `jax-market` → Postgres `quotes`/`candles`.

## Troubleshooting

| Symptom | Likely Cause | Fix |
| --- | --- | --- |
| Connection refused | Gateway not running | Start IB Gateway/TWS |
| 502 / no data | Port mismatch | Check paper vs live port |
| No market data | Missing subscription | Paper accounts can be delayed |

## Resources

- IB API docs: <https://interactivebrokers.github.io/tws-api/>
- IB Gateway download: <https://www.interactivebrokers.com/en/trading/ib-api.php>

## Archive

Historical IB docs and full integration reports live in `Docs/archive/ib/` and `Docs/archive/phase3/`.
