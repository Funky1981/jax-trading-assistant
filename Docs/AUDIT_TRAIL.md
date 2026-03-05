# Audit Trail Guide

This guide shows how to trace both trade execution and AI decision history.

## API Views

Use trader API (`http://localhost:8081`) for timeline and decision records:

```powershell
curl "http://localhost:8081/api/v1/runs?limit=20"
curl "http://localhost:8081/api/v1/runs/<run_id>"
curl "http://localhost:8081/api/v1/ai-decisions?limit=50"
curl "http://localhost:8081/api/v1/ai-decisions/<decision_id>"
curl "http://localhost:8081/api/v1/testing/status"
```

## SQL: Trade Provenance

### 1) Recent trades with artifact and signal lineage

```sql
SELECT
  t.id AS trade_id,
  t.symbol,
  t.side,
  t.quantity,
  t.entry_price,
  t.created_at AS trade_created_at,
  s.id AS signal_id,
  s.orchestration_run_id,
  a.artifact_id,
  a.strategy_name,
  ap.state AS artifact_state,
  ap.validation_passed
FROM trades t
LEFT JOIN strategy_signals s ON s.id = t.signal_id
LEFT JOIN strategy_artifacts a ON a.id = t.artifact_id
LEFT JOIN artifact_approvals ap ON ap.artifact_id = a.id
ORDER BY t.created_at DESC
LIMIT 100;
```

### 2) One trade full path (trade -> signal -> orchestration -> artifact)

```sql
SELECT
  t.id AS trade_id,
  t.created_at AS trade_time,
  s.id AS signal_id,
  s.status AS signal_status,
  o.id AS orchestration_run_id,
  o.status AS orchestration_status,
  a.artifact_id,
  a.hash AS artifact_hash,
  ap.state AS approval_state,
  ap.validation_run_id,
  ap.validation_report_uri
FROM trades t
LEFT JOIN strategy_signals s ON s.id = t.signal_id
LEFT JOIN orchestration_runs o ON o.id = s.orchestration_run_id
LEFT JOIN strategy_artifacts a ON a.id = t.artifact_id
LEFT JOIN artifact_approvals ap ON ap.artifact_id = a.id
WHERE t.id = '<trade_id>'
LIMIT 1;
```

## SQL: AI Decision Trace

### 1) Decision rows by run

```sql
SELECT
  d.id,
  d.run_id,
  d.flow_id,
  d.role,
  d.provider,
  d.model,
  d.schema_valid,
  d.decision,
  d.reasoning,
  d.created_at
FROM ai_decisions d
WHERE d.run_id = '<run_uuid>'::uuid
ORDER BY d.created_at ASC;
```

### 2) Acceptance/override actions tied to decisions

```sql
SELECT
  a.id,
  a.decision_id,
  a.accepted,
  a.accepted_by,
  a.reason,
  a.created_at
FROM ai_decision_acceptance a
JOIN ai_decisions d ON d.id = a.decision_id
WHERE d.run_id = '<run_uuid>'::uuid
ORDER BY a.created_at ASC;
```

### 3) Gate/test evidence for the same run

```sql
SELECT
  tr.id,
  tr.run_id,
  tr.test_name,
  tr.status,
  tr.artifact_uri,
  tr.started_at,
  tr.completed_at
FROM test_runs tr
WHERE tr.run_id = '<run_uuid>'::uuid
ORDER BY tr.created_at ASC;
```

## Operator Pattern

1. Start with run ID from `/api/v1/runs`.
2. Pull decision records from `/api/v1/ai-decisions` (or SQL).
3. Join to trades and artifacts for execution provenance.
4. Confirm test/gate evidence in `test_runs` and `artifact_validation_reports`.
