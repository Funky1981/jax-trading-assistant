# 06 — Agent0 Wiring with Memory (Recall -> Plan -> Act -> Retain)

**Goal:** Make memory part of every decision loop.

## 6.1 — The orchestration pipeline

For each “task” (e.g., evaluate symbol, review earnings):

1) Gather context (Dexter snapshot + user constraints)
2) **Recall** relevant memories
3) Agent0 plans actions using:
   - live context
   - recalled memories
4) Execute tools via go-UTCP
5) **Retain**:
   - plan summary
   - tools used
   - decision outcome (executed/skipped)
   - confidence + rationale

## 6.2 — TDD strategy
- Write tests for the orchestration function:
  - Given a context + recalled memories, Agent0 is called with the merged prompt/context
  - Tool execution order is deterministic
  - Retain is called with a properly formed MemoryItem

Use fakes:
- FakeMemoryStore
- FakeAgent0
- FakeToolRunner

## 6.3 — “What gets retained” template
- `type`: decision
- `symbol`
- `tags`: [strategy, event type, risk level]
- `summary`: 1–3 lines
- `data`:
  - `inputs`: key parameters
  - `plan`: concise steps
  - `action`: executed/skipped
  - `reasoning_notes`: short

## 6.4 — Definition of Done
- One end-to-end unit test that proves:
  - recall is performed
  - a plan is produced
  - actions run
  - retention occurs
