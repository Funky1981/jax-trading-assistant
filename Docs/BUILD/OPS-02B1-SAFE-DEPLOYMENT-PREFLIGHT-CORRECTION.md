# OPS-02B1 — Safe Deployment Preflight Correction

## Status

OPS-02B1 is implemented and validated on top of
`024641f06b311b274f8e08e30225e6da57e7c5d5`.

OPS-02B remains blocked at deployment preflight. The corrected worker gates
were proven, but the controlled `jax-trader` deployment was not started in
this package.

PAPER-02-2026-01 remains `ABORTED / HISTORICALLY PRESERVED / 0 GENUINE
PROSPECTIVE OPPORTUNITIES`.

## World Monitor restoration

The existing `jax-world-news-monitor-worldmonitor-postgres-1` container was
stopped, while its existing `jax-world-news-monitor_worldmonitor-events-data`
volume was retained. The existing events container was restarting with
`getaddrinfo ENOTFOUND worldmonitor-postgres` because it had no active network
endpoint while failing.

The existing PostgreSQL container was started on its original volume. It
became healthy and accepted connections. The existing events container did
not recover until it was restarted once, which reattached its declared
`jax-world-news-monitor_default` and `jax-trading-assistant_default` networks
and restored the `worldmonitor-postgres` alias. No image, volume, database
replacement, truncation, or container recreation was performed.

The restored genuine endpoint returned `200` and `{"status":"ready"}` with
database and collector readiness. Its retained event response reported 623
events, sequences 3 through 39417, and real provider identities including BBC
World, CNBC Top News, and Federal Reserve. The completed collector cycle
reported 82 observed, 64 persisted, 18 duplicates, and 0 failures. No
synthetic event was injected.

## Preserved persistent state

Before any trader start, the normal Jax PostgreSQL database was queried in a
read-only transaction. The durable scanner state was unchanged:

| Field | Value |
|---|---|
| enabled | `true` |
| status | `ready` |
| interval | 60 seconds |
| minimum confidence | 0.7 |
| human approval required | `true` |

There were 19 strategy instances, 12 enabled. The isolated account
`jax-paper-runtime-v1` had zero paper accounts, ledger events, orders, fills,
lifecycles, outcomes, and reviews. PAPER-02 remained `ABORTED`, with one
incident and zero opportunities.

The scanner state and all strategy instances were left unchanged.

## Process-level worker gates

The trader now accepts four explicit startup controls:

- `OPPORTUNITY_SCANNER_WORKER_ENABLED`
- `TRADE_WATCHER_WORKER_ENABLED`
- `MARKET_INGESTER_WORKER_ENABLED`
- `MOBILE_NOTIFICATION_WORKER_ENABLED`

Absent or empty values default to `true`, preserving existing behavior.
Explicit `true` and `false` values are accepted; any other explicit value
fails configuration. Startup logs and `/ready` diagnostics expose the four
resolved gate values.

When a gate is `false`, its goroutine is not launched. In particular, the
opportunity scanner cannot perform its immediate first `ScanOnce`, and a
disabled scanner does not persist scanner state. The trade watcher, market
ingester, and mobile dispatcher are suppressed at the same process boundary.

The execution-instruction worker safety path, exploratory PAPER entry/review
worker, and World Monitor pull worker were not changed. Scanner, strategy,
risk, and approval logic were not changed.

Normal Compose defaults and `.env.example` values are `true`. The later
bounded OPS-02B proof should explicitly set all four new gates to `false`,
keep `WORLD_MONITOR_PULL_ENABLED=true`, and retain:

```text
JAX_RUNTIME_MODE=paper
PAPER_ACCOUNT_ID=jax-paper-runtime-v1
EXECUTION_ENABLED=false
EXECUTION_INSTRUCTION_WORKER_ENABLED=false
ALLOW_LIVE_TRADING=false
BROKER_EXECUTION_ALLOWED=false
MAX_LEVERAGE=1
```

## IB-bridge boundary

Compose currently declares `jax-trader`'s normal `ib-bridge` dependency as
`service_started`. This package does not redesign that topology or start
`ib-bridge`. Source inspection confirms trader readiness requires broker
readiness only when PAPER execution is enabled; with
`EXECUTION_ENABLED=false`, broker readiness is reported as skipped and does
not block trader readiness. The later controlled proof must use an explicit
no-dependency startup boundary if it starts only the already-running
PostgreSQL plus `jax-trader`.

## Deployment boundary

No `jax-trader` deployment proof was started. No new pilot, human approval,
candidate promotion, order, fill, broker action, strategy validation, or live
execution was performed. No OPS-02B completion claim is made.
