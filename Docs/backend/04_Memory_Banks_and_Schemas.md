# 04 — Memory Banks & Schemas

**Goal:** Define *what* we store so recall works and reflection isn’t garbage-in-garbage-out.

## 4.1 — Core memory banks (start minimal)
- `market_events` — earnings, CPI, rate decisions, mergers, halts
- `signals` — “Dexter found X”
- `trade_decisions` — candidate trades + rationale + confidence
- `trade_outcomes` — what happened after we acted (or didn’t)
- `strategy_beliefs` — reflection output (higher-level insights)

## 4.2 — Canonical memory item shape
Keep a consistent base envelope:

```json
{
  "id": "optional",
  "ts": "2025-12-18T09:00:00Z",
  "type": "earnings_event | decision | outcome | belief | signal",
  "symbol": "AAPL",
  "tags": ["earnings", "volatility", "gap-up"],
  "summary": "Short human-readable summary (1-3 lines).",
  "data": { "structured_fields": "go here" },
  "source": { "system": "dexter|agent0|user", "ref": "optional link/id" }
}
```

## 4.3 — TDD requirements
- Write schema validation tests:
  - required fields exist
  - timestamps are RFC3339
  - `summary` length bounds
  - `tags` are lowercased and max N entries
- Add a golden-file test for JSON serialization.

## 4.4 — Query shapes
Queries should support:
- `symbol`
- `type`
- `time_window`
- `tags`
- free-text query

Example:

```json
{
  "q": "earnings surprise gap up fades",
  "symbol": "NVDA",
  "types": ["market_events","trade_outcomes"],
  "from": "2025-01-01T00:00:00Z",
  "to": "2025-12-31T23:59:59Z",
  "tags": ["earnings","gap"]
}
```

## 4.5 — Definition of Done
- Schemas live in `.serena/templates/memory/`
- Validation helpers exist in `libs/contracts` (or `libs/testing`)
- Unit tests cover at least:
  - valid item
  - missing required field
  - invalid timestamp

