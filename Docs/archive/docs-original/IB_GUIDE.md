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
   - Port `7497` (paper) or `7496` (live)
   - Trusted IPs: `127.0.0.1`
4. **Start the IB Bridge (Python)**
   - `docker compose up ib-bridge`
   - Health check: `curl http://localhost:8092/health`

## Connection Settings

| Mode | Host | Port | Notes |
| --- | --- | --- | --- |
| Paper | 127.0.0.1 | 7497 | Recommended for testing |
| Live | 127.0.0.1 | 7496 | Use with caution |

## Configuration Example

```json
{
  "ib": {
    "enabled": true,
    "host": "127.0.0.1",
    "port": 7497,
    "client_id": 1
  }
}
```

## Current Implementation Status

- **IB bridge**: documented as production-ready (FastAPI + ib_insync) and exposed on `8092`.
- **Go provider**: relies on the IB bridge HTTP client.
- **Ingestion wiring**: still requires validation from IB → ingest → storage.

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
