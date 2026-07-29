# Continuous World Monitor pull integration

## Operator commands

From `C:\Projects\Jax`:

```powershell
.\jax-trading-assistant\jax-world-monitor.ps1 up
```

Stop the stack without deleting either PostgreSQL volume:

```powershell
.\jax-trading-assistant\jax-world-monitor.ps1 down
```

Use `status` or `logs` as the action for a bounded operational view. No
TypeScript entry point, migration command, service order, internal port, or
manual HTTP request is required.

## Data flow

```text
RSS/Atom feeds
  -> worldmonitor-events collector
  -> world_monitor_events (monotonic BIGSERIAL sequence)
  -> GET /api/v1/jax/events?after=<cursor>&limit=<1..250>
  -> jax-trader pull worker
  -> genuine event inbox/raw/normalized records
  -> deterministic NO_TRADE/WATCH/CANDIDATE decision
  -> world_monitor_pull_cursors in the same serializable transaction
  -> Evidence Inbox ordered by Jax received_at DESC
```

World Monitor never pushes or acknowledges Jax state. Jax fetches a page before
opening its database transaction. It then locks and rechecks its durable cursor,
idempotently ingests every item, creates or reuses each deterministic decision,
updates the cursor to the final sequence, and commits once. Any item failure
rolls back the whole page. A crash after commit resumes after the committed
position; replay after cursor loss reuses the existing event and decision.

## Configuration

| Variable | Default | Constraint |
|---|---:|---:|
| `WORLD_MONITOR_COLLECT_INTERVAL_MS` | `300000` | 5000–86400000 |
| `WORLD_MONITOR_FEED_TIMEOUT_MS` | `10000` | 1000–30000 |
| `WORLD_MONITOR_PULL_INTERVAL_SECONDS` | `30` | 5–3600 |
| `WORLD_MONITOR_PULL_TIMEOUT_SECONDS` | `10` | 1–30 |
| `WORLD_MONITOR_PULL_PAGE_SIZE` | `100` | 1–250 |

`WORLD_MONITOR_FEEDS_JSON` may replace the three default feeds with a JSON array
of `{ "id", "name", "url" }` objects. Invalid configuration fails closed.

## Safety

The coordinated runtime defaults to paper mode with live trading, execution,
the execution-instruction worker, and broker execution disabled, and maximum
leverage fixed at 1x. The pull worker independently validates all six settings
before it starts. It writes only genuine event records, deterministic decision
records/current projection, and its integration cursor. Unknown assets remain
unknown; no ETF is fabricated.
