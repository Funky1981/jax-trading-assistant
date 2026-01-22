# 02 — Repository Scaffold & Service Skeletons

**Goal:** create the skeleton structure that the rest of the plan plugs into.

## Suggested structure

```
/services
  /jax-orchestrator        # Agent0 integration + workflow runner
  /jax-memory              # Hindsight integration facade
  /jax-ingest              # Dexter ingestion adapters
  /jax-api                 # Optional: REST API gateway
/libs
  /contracts               # shared DTOs/events
  /observability           # logging/tracing helpers
  /testing                 # test fixtures
/Docs
  /backend
  /frontend
/.serena
  /templates
  /checklists
```

## Boundaries (important)
- `jax-memory` is the only code that talks to Hindsight directly.
- `jax-ingest` turns “world events” into canonical internal events and publishes them.
- `jax-orchestrator` calls:
  - recall memory
  - plan actions
  - execute tools
  - retain outcomes

## TDD baseline
For each service, add:
- a health endpoint test (if HTTP)
- a basic “can instantiate config” test
- a “can log something” test

