# 05 — go-UTCP “Memory Tools” (Retain/Recall/Reflect)

**Goal:** Expose memory operations as tools Agent0 can call through go-UTCP.

## Tools to add
- `memory.retain`
- `memory.recall`
- `memory.reflect`

Each tool:
- validates input
- calls `MemoryStore` interface
- logs tool invocation (without secrets)
- returns a clean response DTO

## 5.1 — Contract-first (TDD)
1) Define tool request/response structs in `libs/contracts`.
2) Write tests that:
   - invalid payloads are rejected
   - valid payloads call the store with expected parameters
   - responses are stable (golden JSON)

## 5.2 — Minimal payloads
`memory.retain` request:

```jsonokit
{ "bank":"trade_decisions", "item": { ...MemoryItem... } }
```

`memory.recall` request:

```json
{ "bank":"trade_decisions", "query": { "q":"", "symbol":"AAPL", "tags":["earnings"] } }
```

`memory.reflect` request:

```json
{ "bank":"trade_outcomes", "params": { "window_days": 7, "prompt_hint":"Summarise what worked." } }
```

## 5.3 — Logging and safety
- redact:
  - credentials
  - account IDs if sensitive
  - any “raw” broker payloads
- store only what you can defend keeping

## 5.4 — Definition of Done
- UTCP tools registered and discoverable
- Unit tests pass without requiring docker compose
- Integration test shows Agent0 can call `memory.recall` successfully
