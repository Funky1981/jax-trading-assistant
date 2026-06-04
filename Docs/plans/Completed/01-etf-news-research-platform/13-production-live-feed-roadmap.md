# 13 — Production Live-Feed Roadmap

## Development Stack

```text
Finnhub
Alpaca
existing calendar store
NewsAPI helper only
```

## Production Stack

```text
Polygon / Massive primary
Finnhub secondary
Alpaca backup/market stream
IBKR execution
```

## Production Requirements

```text
provider health checks
latency tracking
duplicate detection
source timestamp
ingestion timestamp
feed failover
stale data rejection
replayable evidence
provider-specific audit trail
```

## Rule

Do not pay for production feeds until the paper-trading loop proves useful.
