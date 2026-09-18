# Engineering Harness Gap Analysis

Status: **HARNESS-02 IMPLEMENTED / EXTERNAL REVIEW REQUIRED**.

This audit covers the infrastructure that enables Codex or another coding agent
to work safely and recoverably in the Jax repository. It does not assess the
quality of a future JaxMind runtime loop except where the two harnesses must
share a contract.

## 1. Existing durable surfaces

| Surface | Current repository evidence | Assessment |
| --- | --- | --- |
| Repository map | Root `AGENTS.md`, `skills/STARTER_PROMPT.md`, `skills/jax-repo-guardrails/` | Useful routing map and safety skill; not yet a machine-readable package-state map. |
| Documentation authority | `Docs/JAX_PRODUCT_CHARTER.md`, `ROADMAP.md`, `STATUS.md`, `CAPABILITY_MATRIX.md`, `BUILD/CURRENT_PACKAGE.md`, `DOCUMENTATION-AUTHORITY.md` | Strong authority chain; current package and status can drift unless validated together. |
| Runtime architecture | `Docs/ARCHITECTURE.md`, ADR-0012, architecture diagram | Current two-runtime intent is clear; ADR contains historical/proposal material that requires careful routing. |
| Research/evidence | `Docs/RESEARCH/`, `Docs/evidence/`, `Docs/validation/`, provider/provenance contracts | Substantial source and validation evidence; no unified engineering-harness retrieval index. |
| Memory/review | `Docs/MEMORY_AND_REVIEW/`, decision/review/lesson concepts | Durable review principles exist; no harness task-memory contract or contamination tests. |
| Build routing | `Docs/BUILD/README.md` and `CURRENT_PACKAGE.md` | Current routing is explicit and safety-aware; no structured command registry with scope/cost/outputs. |
| Skills | Repository-local skills and referenced validation scripts | Good task routing; no automated proof that the selected skill was applicable or read. |
| Verification | `scripts/go-verify.ps1`, `scripts/golden-check.ps1`, CI workflows, import-boundary workflow | Strong existing commands and CI; selection is still largely agent judgment. |
| Handoffs | Root completion rules and prior handover conventions | Human-readable handoffs exist; no canonical machine-readable session checkpoint. |
| Completion | AGENTS completion requirements, CI, package evidence docs | Evidence exists but no single structured definition-of-done record binds scope, diff, tests, review, and known failures. |
| External review | Validation and review evidence under `Docs/validation/` and `Docs/BUILD/` | Mature for trading packages; no standard harness external-review evidence package. |

## 2. Missing or weak capabilities

### A. Authoritative engineering task state

There is no canonical, versioned `EngineeringTaskState` that a fresh agent can
load to learn objective, scope, constraints, changed files, completed checks,
known failures, open questions, and next action. Chat and a handover document
can currently fill this role, but they are not machine-validated and can drift
from the worktree.

### B. Verification command registry

Commands are documented across `AGENTS.md`, skills, `Docs/CONTRIBUTING.md`,
architecture docs, scripts, and CI. The repository lacks one registry mapping
scope to required/optional commands, expected artifacts, environmental needs,
cost, and known unrelated-failure behavior.

### C. Current-state consistency check

Authority documents are individually clear, but there is no documented check
that branch/SHA, current package, roadmap status, capability row, worktree,
and declared pilot/runtime state agree. This is particularly important because
the handover state can change after the older status documents were written.

### D. Structured handoff and fresh-session resume

The repository requires a durable handoff, but the format is not a versioned
schema with objective, state version, evidence references, decisions, failures,
and next action. A future engineering harness should be able to validate that a
handoff is sufficient before a task is considered complete.

### E. Clean-exit and definition-of-done contract

There are distributed completion rules, but no single machine-checkable clean
exit checklist covering diff scope, runtime-file isolation, tests, exact commit,
CI, known failures, status updates, and handoff completeness. This makes
premature completion possible even when individual tests pass.

### F. Plan and status lifecycle

The canonical roadmap/current package are authoritative, while ProjectOS and
many historical plans are explicitly non-authoritative. This is safe by policy
but creates a routing burden. A future engineering harness should detect
competing current-looking documents and identify the canonical source rather
than ingesting all plans.

### G. Review and recovery telemetry

There is no standard engineering `HarnessRun` equivalent capturing selected
documents, omitted documents, commands, tool outputs, retries, and handoff
state. Existing CI records provide part of this after push, but not the full
agent-visible working package.

## 3. Recommended future engineering-harness contracts

The engineering harness should eventually define:

- `EngineeringObjective` — requested outcome, scope, exclusions, and authority;
- `RepositoryState` — branch, starting SHA, worktree status, relevant files,
  and current package identity;
- `DocumentationMap` — authority chain, selected docs, historical exclusions,
  and links used for the task;
- `VerificationPlan` — commands, reasons, expected outputs, environment, and
  skipped-command rationale;
- `EngineeringCheckpoint` — completed work, open questions, failures, next
  action, and durable evidence references;
- `CleanExitReport` — diff scope, runtime isolation, tests, commit/push/CI,
  known failures, and handoff completeness; and
- `EngineeringHarnessRun` — correlation of the above without copying giant
  source files or command outputs unnecessarily.

HARNESS-01 implements these as dependency-light contracts in
`internal/modules/harnesscontracts`; it does not implement orchestration,
durable persistence, command execution, or a fresh-session runner. The
remaining gaps are validation consumers, repository-state consistency checks,
checkpoint persistence/resume, and evidence-producing clean-exit tooling.

## 4. Current strengths to preserve

- The authority chain is short and explicitly ranked.
- `AGENTS.md` instructs agents to use a map rather than making it an
  encyclopedia.
- The repository has a dedicated guardrail skill and validation scripts.
- CI protects Go, frontend, golden/replay, and import-boundary behavior.
- The documentation distinguishes exploratory paper, formal paper, and
  demonstrated edge.
- Product and architecture docs explicitly prohibit live execution and unsafe
  promotion.
- Historical material is retained for traceability rather than silently erased.

## 5. Engineering-harness implementation priorities

The roadmap should address, in order:

1. versioned task/checkpoint and clean-exit contracts;
2. authority/documentation map and verification registry;
3. resumable handoff and current-state consistency checks;
4. command/tool observability with references and hashes;
5. fresh-session and premature-completion evaluation; and
6. external-review evidence for the harness itself.

These priorities remain separate from the Jax runtime/research harness phases.

## 6. HARNESS-01 implementation notes

The verification registry is declarative metadata populated only with existing
repository commands. It is not a shell-execution engine and must not execute
untrusted command strings. Clean-exit validation requires runtime isolation,
check evidence, commit/push state, zero divergence, complete handover, and
exact-SHA CI evidence before a successful result can be recorded.

Persistence was intentionally deferred because the contracts are testable as
pure values and no HARNESS-01 requirement justified a new datastore or a
second audit ledger. Later persistence must use existing repository conventions
and remain outside the active PAPER-02 path.

## 7. Explicit non-actions in HARNESS-01

This package does not rewrite `AGENTS.md`, ProjectOS files, archived plans, CI,
runtime code, or trading behavior outside the canonical routing updates. It
does not implement retrieval, memory behavior, compaction, LLM calls,
evaluation execution, orchestration, PAPER-02 integration, formal paper, live
trading, or Phase 13.
The ProjectOS instructions are retained as historical/process material because
the active documentation authority explicitly says they must not route current
Jax work.

## 8. HARNESS-02 scope note

HARNESS-02 is a runtime/research-harness package only. It does not close the
engineering-harness gaps above: there is still no command executor, durable
engineering checkpoint store, fresh-session runner, current-state consistency
checker, or machine-enforced clean-exit consumer. Its `BuildReport` records
retrieval and context-construction evidence for later observability, but it is
not an engineering handoff or definition-of-done report.
