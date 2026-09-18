# Jax Harness Implementation Roadmap

Status: **HARNESS-03 IMPLEMENTED / EXTERNAL REVIEW REQUIRED**.
Execution status: **foundation contracts and offline retrieval/context
construction only; no model or harness runtime integration is authorized by
this document**.

This is the detailed roadmap referenced by `Docs/ROADMAP.md`. It is subordinate
to the product charter, current roadmap, status, capability matrix, and current
build routing. Each package requires its own reviewed gate.

## Sequencing rules

- Start with the smallest single-controller architecture that can be evaluated.
- Keep Engineering Harness and Jax Runtime/Research Harness contracts separate.
- Preserve `NO_TRADE`, deterministic risk, human approval, and the two-runtime
  ADR-0012 boundary.
- Prefer references and hashes to duplicate raw evidence or audit payloads.
- Do not implement a later package merely because this roadmap describes it.
- Any integration with PAPER-02 or trading requires a separate post-PAPER-02
  gate and external review.

## HARNESS-00 — Architecture specification and repository audit

**Status:** IMPLEMENTED / EXTERNAL REVIEW REQUIRED.

**Purpose:** Define the two harnesses, stable conceptual contracts, context
budgeting, retrieval, memory separation, resumability, compaction, evaluator,
falsification, provenance, evaluation laboratory, engineering gaps, and gated
sequence.

**Dependencies:** Existing authority chain, ADR-0012 architecture, evidence and
memory/review documentation, current CI/verification surface, and frozen
PAPER-02 boundary.

**Deliverables:** `Docs/HARNESS/` canonical index, architecture specification,
evaluation specification, implementation roadmap, engineering gap analysis,
and reconciled links from the canonical roadmap/status/current package.

**Tests/checks:** Documentation review, link/authority review, `git diff --check`,
relevant repository validation, runtime-file isolation review, and exact-SHA CI.

**Acceptance criteria:** Documentation-only diff; all required architecture
concerns are covered; no duplicate roadmap authority is created; PAPER-02
isolation is explicit; later packages remain gated.

**Failure conditions:** Missing contract boundary, support-only retrieval design,
unreconstructable provenance proposal, stale/current authority conflict,
runtime change, or an implied authorization for trading/formal paper/Phase 13.

**Does not authorize:** HARNESS-01, any runtime implementation, PAPER-02
mutation, formal paper, live execution, or Phase 13.

## HARNESS-01 — Foundation and contracts

**Status:** IMPLEMENTED / EXTERNAL REVIEW REQUIRED.

**Implementation evidence:** `internal/modules/harnesscontracts` contains the
stdlib-only versioned contracts and deterministic validation suite. The package
implements contract identity, fail-closed validation, SHA-256 context-package
content hashing, budget invariants, provenance references, engineering
verification metadata, and clean-exit evidence. Persistence is deferred; no
runtime integration or model/tool execution was added.

**Isolation evidence:** The package is not imported by `cmd/trader`; no
PAPER-02 code, policy, identity, evidence selection, intake, or sample state
was changed. The older `internal/modules/harness` package remains untouched and
is not silently designated as the new foundation package.

**Purpose:** Version and validate the conceptual contracts needed by a future
single-controller harness: `ResearchObjective`, `TaskState`, `Checkpoint`,
`HarnessRun`, structured outputs, evaluator inputs/results, permissions, and
stable references.

**Dependencies:** HARNESS-00 external review; existing canonical evidence,
provenance, BlackBox/audit, Experience/Judgment, and ADR-0012 boundaries.

**Deliverables:** Versioned Go contracts, validation rules, deterministic hash
policy, engineering-harness state/verification/clean-exit contracts, and
read-only deterministic fixtures. No trading integration or persistence
migration.

**Tests:** More than 40 meaningful contract and negative-path cases covering
objective/state validity, fresh-session sufficiency, canonical hashing and
ordering, budget overflow/reserve rules, evidence/memory boundaries, checkpoint
and tool lifecycle, structured output, evaluator decisions, provenance,
engineering state, verification trust, clean-exit evidence, secrets, and
execution isolation.

**Acceptance criteria:** A fresh session can load a valid task state; invalid or
ambiguous states fail closed; contracts do not conflate evidence, memory,
state, and context; no trader import boundary is weakened.

**Failure conditions:** Transcript-only state, mutable identity, unsafe default,
unversioned output, hidden policy, or a contract that requires one model's
private format.

**Does not authorize:** HARNESS-02, context selection, JaxMind calls,
evidence/memory retrieval changes, compaction, evaluator execution,
orchestration, PAPER-02 integration, or execution.

## HARNESS-02 — Retrieval and Context Builder

**Status:** IMPLEMENTED / EXTERNAL REVIEW REQUIRED.

**Implementation evidence:** `internal/modules/contextbuilder` provides
read-only evidence, contradiction, and memory retriever interfaces plus
deterministic fixture adapters. The builder uses explicit reference time,
separate supporting/counter-evidence budgets, deterministic multi-factor
ranking and tie-breaking, source-diversity preference, temporal fail-closed
rules, omission/deferred references, and HARNESS-01 canonical package hashing.
The offline suite contains 53 named behavioral cases. No production adapter,
model call, JaxMind integration, persistence, or trading integration was added.

**Isolation evidence:** `cmd/trader` does not import
`internal/modules/contextbuilder`; no PAPER-02 code, state, policy, intake,
evidence selection, or sample was changed. The package is research-side and
read-only.

**Purpose:** Build bounded, objective-specific Context Packages from evidence,
counter-evidence, memory, state, constraints, and tool references.

**Dependencies:** HARNESS-01; qualified evidence/provenance references; reviewed
memory separation; deterministic budget policy.

**Deliverables:** Read-only retriever interfaces, contradiction retrieval,
selection rationale, budget allocator, package hashing, deferred/on-demand
handles, and auditable omission behavior.

**Tests:** Context overload, contradiction hiding, missing data, budget reduction,
source attribution, evidence/memory/state separation, and deterministic package
rebuild.

**Acceptance criteria:** Required constraints and counter-evidence reserve cannot
be silently consumed; omitted material is labelled; similarity alone cannot
produce sufficiency; package hash/references are reconstructable.

**Failure conditions:** Support-only context, all-history loading, silent
truncation, memory presented as evidence, or non-deterministic selection with
no trace.

**Does not authorize:** HARNESS-03, model/JaxMind calls, evaluator execution,
task-state persistence, full memory architecture, trading decisions, PAPER-02
context/evidence changes, or a new AI subsystem outside the research boundary.

## HARNESS-03 — Checkpointing, resumability, and compaction

**Purpose:** Make long-horizon work survive context exhaustion, process restart,
tool failure, and fresh sessions.

**Dependencies:** HARNESS-01 and HARNESS-02; durable storage and audit links.

**Deliverables:** Append/version state transitions, checkpoint writer/reader,
compaction record, retry/failure state, resume protocol, and operator recovery
view.

**Tests:** Long-horizon continuity, forced multiple compactions, fresh-session
resume, interrupted tool call, duplicate retry, unresolved contradiction, and
constraint retention.

**Acceptance criteria:** Resume reconstructs objective, constraints, completed
work, unresolved questions, evidence references, decisions/reasons, and next
action without chat history; canonical evidence remains intact.

**Failure conditions:** Compaction erases contradictions, state says complete
without evidence, retry duplicates durable work, or original references cannot
be recovered.

**Does not authorize:** Outcome leakage, historical replay as prospective input,
or PAPER-02 ledger mutation.

**Implementation status:** HARNESS-03 is implemented in
`internal/modules/harnessstate` and is pending external review. It provides the
versioned store, isolated PostgreSQL migration, deterministic compaction,
resume bundle, failure/retry state, and ContextBuilder resume seam described
above. HARNESS-04 remains unimplemented and unauthorized.

## HARNESS-04 — Memory architecture

**Purpose:** Integrate relevant Experience/Judgment lessons without treating
memory as current evidence or allowing stale/irrelevant analogies to dominate.

**Dependencies:** HARNESS-01 through HARNESS-03; existing review and lesson
promotion concepts; leakage-safe outcome boundaries.

**Deliverables:** Memory identity/provenance contract, retrieval ranking,
regime/expiry controls, contamination labels, calibration links, and memory
evaluation fixtures.

**Tests:** Memory contamination, regime mismatch, stale memory, outcome leakage,
confirmation bias, irrelevant analogy, and memory-off ablation.

**Acceptance criteria:** Memory is visibly distinct, time-bounded, provenance-
linked, and unable to satisfy a current evidence requirement by itself.

**Failure conditions:** Memory silently becomes evidence, leaks future outcomes,
or cannot be invalidated when the regime/source changes.

**Does not authorize:** Automatic strategy/policy tuning, promotion of lessons to
rules, or any PAPER-02 behavior change.

## HARNESS-05 — Independent evaluator and falsification

**Purpose:** Add independent sufficiency, quality, causal, uncertainty, policy,
and premature-completion evaluation with active counter-thesis retrieval.

**Dependencies:** HARNESS-01 through HARNESS-04; structured outputs and
reconstruction-grade package references.

**Deliverables:** Evaluator contract, PASS/REVISE/ABSTAIN result, correction
loop, falsification protocol, evaluator independence boundary, and fixtures.

**Tests:** Falsification, unsupported claims, causal gaps, hidden contradiction,
missing data, premature completion, evaluator ablation, and revision convergence.

**Acceptance criteria:** Producer cannot self-certify; evaluator findings are
structured and provenance-linked; PASS is explicitly non-trading and does not
grant approval.

**Failure conditions:** Rubber-stamp evaluator, absent counter-evidence, hidden
uncertainty, or a PASS that can reach execution without existing gates.

**Does not authorize:** Trading/risk approval, formal evidence promotion, live
execution, or automatic policy changes.

## HARNESS-06 — Provenance, observability, and reconstruction

**Purpose:** Make important harness runs observable and answer what the model
actually knew for a historical decision.

**Dependencies:** HARNESS-01 through HARNESS-05; existing BlackBox/audit and
canonical raw-payload/reference stores.

**Deliverables:** `HarnessRun` correlation, `ContextProvenance`, invocation and
tool records, stable hashes, model/prompt/version records, token/latency/cost
telemetry, and reconstruction tooling/read model.

**Tests:** Full chain reconstruction, missing-reference detection, hash mismatch,
tool failure, provider/version change, duplicate-persistence audit, and trace
completeness.

**Acceptance criteria:** Important runs are replayable or explicitly labelled
partial/failed reconstruction; canonical sources are not duplicated without
reason; observability never hides uncertainty or omissions.

**Failure conditions:** “Exact context” claim without hash/reference proof,
silent source drift, untraceable tool output, or a second conflicting audit
ledger.

**Does not authorize:** Model upgrade claims, trading integration, or changes to
existing BlackBox semantics without a separate architecture decision.

## HARNESS-07 — Evaluation laboratory and ablation

**Purpose:** Establish repeatable quality, safety, efficiency, continuity, and
ablation evidence for the harness itself.

**Dependencies:** HARNESS-01 through HARNESS-06.

**Deliverables:** Versioned scenario corpus, oracle format, runner, metrics
collector, comparison reports, baseline/ablation protocol, and external-review
evidence package.

**Tests:** All scenarios in `Docs/HARNESS/EVALUATION.md`, repeated runs,
budget/model variation, tool failure, reconstruction, and engineering clean
exit.

**Acceptance criteria:** Component value is measurable by scenario; no single
aggregate hides regressions; removable components are identified; multi-agent
claims require demonstrated material benefit.

**Failure conditions:** Unrepeatable fixture, aggregate-only score, leakage,
missing negative cases, or complexity added without benefit evidence.

**Does not authorize:** Production use in PAPER-02, formal paper, or live
trading. It authorizes evaluation evidence only.

## HARNESS-08 — Advanced orchestration (CONDITIONAL)

**Purpose:** Evaluate specialist separation only if HARNESS-07 shows that it
materially improves quality or reliability enough to justify cost and failure
surface.

**Dependencies:** HARNESS-07 external review and a specific measured deficiency
that a specialist boundary addresses.

**Possible specialists:** Evidence Researcher, Quant Context Researcher,
Counter-Thesis/Falsification Researcher, and Evaluator.

**Deliverables:** A narrowly scoped orchestration experiment, ownership and
context boundaries, cost/latency budget, failure containment, and ablation
comparison against the single-controller baseline.

**Tests:** Specialist-isolation, conflict resolution, budget sharing, tool
permissions, failure recovery, provenance, and no-swarm enforcement.

**Acceptance criteria:** Demonstrated material improvement on named scenarios,
acceptable cost/latency, clear ownership, bounded concurrency, and graceful
fallback to the simpler architecture.

**Failure conditions:** Agent duplication, context leakage, unclear authority,
unbounded loops, higher cost without quality benefit, or inability to reconstruct
which specialist saw what.

**Does not authorize:** An uncontrolled agent swarm, autonomous trading, or any
integration into frozen PAPER-02.

## Global gate and review policy

Each phase must ship its own schema/version notes, test evidence, risk review,
explicit non-authorizations, and durable handoff. A phase is not complete merely
because code or documentation exists. External review is required before the
next phase becomes current work.

The next package after HARNESS-03 is **HARNESS-04**, but it is not started or
authorized by this handover. Stop after HARNESS-03 pending external review.
