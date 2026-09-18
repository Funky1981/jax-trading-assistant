# HARNESS-03 — Checkpointing, Resumability & Structured Compaction

Status: **IMPLEMENTED / EXTERNAL REVIEW REQUIRED**.

Starting SHA: `23f8f6aa18c0697e45d01d6e0d020a1c457e09ae`<br>
Branch: `capability-reset`<br>
Frozen PAPER-02 runtime SHA: `521a4917c5e6be639ed8571cf383f22d99f1d077`

## Scope and architecture audit

The audit confirmed that `internal/modules/harnesscontracts` already owns the
versioned research objective, task state, evidence, counter-evidence, memory,
tool, checkpoint, provenance, and canonical SHA-256 conventions.
`internal/modules/contextbuilder` is a read-only HARNESS-02 seam that accepts a
`BuildRequest` and does not own persistence. Existing PostgreSQL migrations are
additive; the older `research_task_checkpoints` table is not a complete
HARNESS-03 store and is not repurposed.

HARNESS-03 adds `internal/modules/harnessstate`. It is research-side only, does
not import trading packages, does not call a model, and stores references to
evidence and memory rather than copying canonical payloads. No competing audit
ledger or PAPER-02 storage path was introduced.

## Persistence and state history

`Store` exposes task creation/loading, append-only state versions, historical
loads, checkpoint creation/loading/listing, failure/retry recording, resume,
and derived compaction operations.

`PostgresStore` persists to additive migration
`000075_harness_durable_task_state`:

- `harness_tasks` stores objective identity and the latest-version pointer.
- `harness_task_state_versions` retains every version with a content hash.
- `harness_checkpoints` stores immutable structured continuation records.
- `harness_compactions` stores derived records, never canonical state.
- `harness_failure_events` and `harness_retry_events` preserve operational history.

The external-review correction required no schema migration: the existing event
primary key now stores a deterministic application-generated hash of operation,
task ID, and idempotency key, while the JSON event payload carries the semantic
content used for replay comparison. Migration `000075` remains additive and
unchanged.

`MemoryStore` implements the same semantics for deterministic offline tests and
supports snapshot/restore to model a process boundary.

## Versioning, concurrency, and idempotency

Every append requires `expectedVersion`. PostgreSQL locks the version pointer;
the in-memory store serializes it. A stale writer receives an explicit
`VersionConflictError`; last-writer-wins is not permitted.

State, checkpoint, compaction, failure, and retry operations use explicit
idempotency identities. Failure and retry event identities are deterministic
hashes of the operation type, task ID, and caller key; caller keys are never
used as globally unique event IDs. A repeated identical request returns the
durable record with `Applied=false`; reuse with different semantic content
returns `ErrIdempotencyConflict`, including different failure state or next
action.

## Checkpoint contract and integrity

`CheckpointContent` retains objective identity/version, state version, stage,
completed and pending work, hard constraints, decisions/reasons, open
questions, known unknowns, evidence and counter-evidence references, memory and
tool references, unresolved contradictions, failure state, next action, and
provenance references. It stores references and bounded structured state, not
giant evidence payloads.

Checkpoint hashes are canonical SHA-256 hashes of normalized semantic content.
IDs, creation timestamps, and other volatile audit metadata are excluded.
Unordered collections are sorted before hashing. Loading a record recomputes its
hash, so tampering is detectable.

## Resume protocol

`ResumeTask(TaskID, optional CheckpointID)` loads the objective, selected
historical state version, checkpoint, failure history, and retry history. It
validates that checkpoint content matches the referenced canonical state. The
returned `ResumeBundle` is sufficient for a fresh controller/session with zero
transcript history and includes the next action and continuity-critical
references.

`ResumeBundle.BuildRequest` maps the durable bundle to a
`contextbuilder.BuildRequest`, carries the checkpoint and tool references, and
leaves retrieval adapters caller-owned. Offline fixtures prove
`ResumeBundle -> BuildRequest -> ContextPackage` without a model call.

## Structured compaction

Compaction is deterministic extraction, not LLM summarisation. The v1 policy
retains all loss-sensitive structured fields and their canonical references. A
`CompactionRecord` has its own content hash and `CanonicalStorage=false`; it
never replaces or deletes canonical task-state versions, evidence, or memory.
Repeated compactions are independently addressable and idempotent.

Validation prevents or detects uncertainty becoming fact, unresolved
contradictions being dropped, hard constraints/failure state being removed,
pending work being silently completed, decisions or evidence classification
changing, memory becoming evidence, and state/checkpoint hash or identity
mismatches.

## Failure/retry, security, and isolation

Operational failures are versioned failed states plus durable failure events.
In PostgreSQL, the state-version append, latest-version update, and matching
failure/retry event insert run in one transaction; event persistence errors are
returned and roll back the state transition. Retry appends a new running state
and retry event; the previous failed state remains readable in history. The
tool-failure fixture persists successful work, records a failed required tool,
simulates restart, retries idempotently, and continues without repeating
completed work.

Validation rejects API keys, authorization/cookie headers, passwords, broker
credentials, database URLs, bearer tokens, and similar secret material from
task state, checkpoint content, provenance, and tool references.

`cmd/trader` has no HARNESS-03 import. HARNESS-03 has no PAPER-02 import, does
not consume PAPER-02 state/checkpoints/compaction, and does not modify
candidate, strategy, risk, entry/exit, broker, execution, intake, samples,
policy, or evidence-selection paths. Migration 000075 contains only new
`harness_*` tables and no PAPER-02 alteration or backfill.

## Test evidence

`TestHarness03BehaviouralGates` contains 65 meaningful subtests, supplemented
by focused failure/retry idempotency tests, covering
creation, historical state, optimistic concurrency, state/checkpoint
idempotency, deterministic and corruption-detecting hashes, fresh-session
recovery, checkpoint content, resume identity, repeated compaction,
constraint/contradiction/unknown/decision/reference retention, failure/retry,
tool-failure restart, ContextBuilder reconstruction, secret rejection,
snapshot restore, and the PAPER-02 import firewall.

The migration registry test verifies the additive migration pair and its
PAPER-02 independence. PostgreSQL repository coverage is provided by the
optional `HARNESS_POSTGRES_DSN` integration test when an isolated test database
is configured; it is skipped when that environment variable is absent.

## Non-authorizations and next package

This package does not authorize HARNESS-04, JaxMind/LLM calls, production
evidence adapters, full memory architecture, evaluator execution, automatic
research loops, multi-agent orchestration, strategy changes, formal forward
paper, IB paper execution, broker/live execution, live trading, or Phase 13.

Stop at HARNESS-03 for external review. HARNESS-04 is not implemented or
authorized by this package.
