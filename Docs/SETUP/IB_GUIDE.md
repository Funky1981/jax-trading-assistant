# Interactive Brokers Guide

## Purpose

This guide covers IB Gateway/TWS setup and how `ib-bridge` is wired into the current runtime.

## Current Runtime Wiring

- `ib-bridge` service: `http://localhost:8092/health`
- `jax-trader` uses `IB_BRIDGE_URL` for market/execution integration
- `agent0-service` also references `ib-bridge` for planning context

## Quick Start (Paper Trading)

1. Install IB Gateway or TWS.
2. Login with paper credentials.
3. Enable API socket access in settings.
4. Configure trusted localhost access.
5. Start bridge:

```powershell
docker compose up -d ib-bridge
curl http://localhost:8092/health
```

## ETF Phase-1 IB Rules

- ETF phase 1 uses IB paper trading only.
- `IB_PAPER_TRADING=true` is required for ETF paper pilot sessions.
- ETF entry orders must originate from the approval workflow; direct order-ticket ETF entries are blocked before reaching `ib-bridge`.
- Operators may still use close, cancel, and protection actions to manage existing paper ETF exposure.
- Before approving an ETF candidate, confirm IB/TWS is connected and paper trading is visible in `/api/v1/trading/pilot-status`.

## Connection Settings

| Mode | Host | Port | Notes |
| --- | --- | --- | --- |
| Paper (IB Gateway) | 127.0.0.1 | 4002 | Preferred |
| Paper (TWS) | 127.0.0.1 | 7497 | Legacy |
| Live | 127.0.0.1 | 7496 | Use with explicit controls |

## Compose Environment Knobs

- `IB_GATEWAY_HOST` (default `host.docker.internal`)
- `IB_GATEWAY_PORT` (default `4002`)
- `IB_CLIENT_ID` (default `1`)
- `IB_AUTO_CONNECT` (default `true`)
- `IB_PAPER_TRADING` (default `true`)

## Validation Checks

```powershell
docker compose logs -f ib-bridge
Invoke-RestMethod http://localhost:8092/health
```

If bridge is healthy but market data is stale, verify IB session login and market data subscription state.

## Troubleshooting

| Symptom | Likely Cause | Fix |
| --- | --- | --- |
| Connection refused | IB Gateway/TWS not running | Start Gateway/TWS |
| Bridge unhealthy | Host/port mismatch | Verify `IB_GATEWAY_HOST` and `IB_GATEWAY_PORT` |
| No live quotes | Paper account delay/no entitlement | Confirm subscriptions and paper-data behavior |

## Historical References

Legacy IB reports are archived under `Docs/archive/ib/` and `Docs/archive/phase3/`.
