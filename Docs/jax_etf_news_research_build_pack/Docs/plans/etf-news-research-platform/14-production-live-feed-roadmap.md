# 14 — Production Live-Feed Roadmap

## Goal

Prepare for production-quality live data after paper testing proves the system.

## Development Stack

Use:

```text
Finnhub
Alpaca
existing calendar store
NewsAPI as helper only
```

## Production Stack

Recommended:

```text
Polygon / Massive primary
Finnhub secondary
Alpaca backup/market stream
IBKR execution
```

## Production Requirements

Live provider layer must support:

- provider health checks
- latency tracking
- duplicate detection
- source timestamp
- ingestion timestamp
- feed failover
- stale data rejection
- replayable evidence
- provider-specific audit trail

## Live Feed Safety

No trade candidate if:

- news source unhealthy
- market data provider stale
- source timestamp missing
- event duplicate uncertain
- symbol mapping uncertain
- provider latency too high
- spread data unavailable

## Upgrade Path

### Stage 1

Local historical backfill.

### Stage 2

Daily scheduled backfill.

### Stage 3

Delayed/live-ish feed in paper mode.

### Stage 4

Paid real-time feed in paper mode.

### Stage 5

Production-ready paper trading.

### Stage 6

Only after long evidence period, consider live mode.

## Acceptance Criteria

- Provider status visible in UI.
- Every candidate stores source/provider details.
- Decisions can be replayed later.
- Provider outage blocks trading.
