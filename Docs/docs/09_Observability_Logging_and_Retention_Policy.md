# 09 — Observability (Logs, Metrics, Traces) + Retention Policy

**Goal:** Make everything testable, debuggable, and safe to store.

## 9.1 — Logging
- Use structured logs (JSON)
- Include correlation IDs per run:
  - `run_id`
  - `task_id`
  - `symbol`

## 9.2 — Metrics
Track:
- number of recalls per run
- latency per tool call
- retain success/fail
- reflection duration

## 9.3 — What to retain to memory vs logs
- Logs: verbose debugging details (rotated / not “remembered”)
- Memory: durable summaries and outcomes

## 9.4 — Safety redaction
- Implement a redaction helper:
  - remove broker credentials
  - remove raw order payloads if sensitive

**TDD:** unit tests for redaction:
- given sensitive fields -> ensure removed

## 9.5 — Definition of Done
- Every tool call logs start/end with run_id
- Retention attempts log result
- Redaction is tested
