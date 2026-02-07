# Merge and Stabilise the Audit Logger

The first step towards a production‑ready Jax trading system is to merge and stabilise the **audit logging** feature. The open pull request (`codex/add-auditevent-model-and-logger`) introduces a structured audit logging capability that tracks decisions and outcomes throughout the event detection, trade generation, risk calculation and orchestration workflows.

## Why it matters

Traceability is critical for compliance and debugging. Without an audit trail, it is impossible to reason about how the system arrived at a trade recommendation or why a risk calculation failed. The new `AuditEvent` model, correlation ID helpers and `AuditLogger` provide the foundation for capturing this information.

## Tasks

- Review PR #3 to ensure that:
  - All new files (`audit_context.go`, `audit_ids.go`, `audit_logger.go`, `audit_payloads.go`) follow the clean‑architecture boundary rules.
  - Instrumentation in `EventDetector`, `TradeGenerator`, `RiskEngine` and `Orchestrator` correctly logs start, success, skipped and error outcomes with redacted payloads.
  - UTCP storage has a new `SaveAuditEvent` method and uses a consistent event type prefix (e.g. `audit.<action>`).
  - Unit tests are updated and new tests are added to ensure no nil pointer dereferences when `AuditLogger` is `nil`.

- Merge the branch into `main` once tests pass. Update `README` or API docs to describe the new audit API and how to query audit events from storage.

- Verify that the audit logger works end‑to‑end in a local environment by:
  1. Running `go run ./services/jax-api` with a Postgres instance configured.
  2. Sending a `POST /symbols/{symbol}/process` request with valid parameters.
  3. Inspecting the `utcp.stored_events` table to confirm that audit events are persisted with correlation IDs and timestamps.
