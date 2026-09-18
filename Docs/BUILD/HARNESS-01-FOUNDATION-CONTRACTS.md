# HARNESS-01 Foundation & Contracts

Status: **IMPLEMENTED / EXTERNAL REVIEW REQUIRED**.

## Scope

HARNESS-01 establishes versioned, deterministic foundation contracts for the
separate Jax runtime/research harness and Codex engineering harness. It is a
contracts-only package. It does not implement ContextBuilder retrieval,
evidence ranking, contradiction search, memory retrieval, compaction, model
calls, evaluator execution, orchestration, persistence, trading integration,
or any change to the active PAPER-02 experiment.

## Implementation

Canonical package: `internal/modules/harnesscontracts`.

The package provides validated contracts for:

- research objectives, durable task state, context packages and budgets;
- canonical evidence, counter-evidence, memory, tool, checkpoint, and
  provenance references;
- structured research output, evaluator results, and correlated HarnessRun;
- evaluator-input and compaction-record contracts that preserve references
  without implementing evaluation or compaction;
- engineering objectives, repository state, documentation maps, verification
  registries/plans, engineering checkpoints, clean-exit reports, and
  EngineeringHarnessRun.

Context-package content identity is versioned SHA-256 over canonical JSON. The
canonicalizer sorts collections whose order is non-semantic and excludes
volatile package creation/audit metadata. Model-visible content changes the
hash; audit-only timestamps do not. The package validates the hash on read.
Counter-evidence has explicit contradictory/invalidating/unknown-missing
classification and a reserved budget. MemoryReference has no evidence
classification and cannot substitute for EvidenceReference. Evaluator PASS is
research sufficiency only and carries `ExecutionAuthority=NONE`.

## Engineering-harness foundation

The declarative verification registry contains only existing repository
commands. It is metadata, not an execution engine. Clean-exit PASS requires
mandatory check evidence, runtime isolation, handover completeness, pushed
commit state, origin equality, zero divergence, and exact-SHA CI evidence.
`RepositoryState` records repository HEAD/starting SHA separately from the
frozen PAPER-02 runtime SHA.

Persistence is deferred. These contracts are pure Go values and are tested
without introducing a datastore, migration, or duplicate audit ledger.

## Verification and review evidence

The package includes deterministic positive and negative tests covering the
required contract boundaries, canonical hashing, ordering invariance, budget
overflow/reserve rules, fresh-session state, secrets, engineering clean exit,
and PAPER-02 import isolation. Full repository validation and exact-final-SHA
CI results are recorded in the final handover and must be completed before
external review.

## Isolation and non-authorizations

The package is not imported by `cmd/trader`. The pre-existing
`internal/modules/harness` advisory/chat package remains unchanged and is not
the HARNESS-01 foundation package. No PAPER-02 code, policy, identity,
prospective intake, evidence selection, or sample accounting is changed.

HARNESS-02 is not started. PAPER-02 remains active, exploratory, frozen, and
unable to establish demonstrated edge; formal forward paper, live trading, and
Phase 13 remain unauthorized.
