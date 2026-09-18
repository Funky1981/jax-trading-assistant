# Jax Harness Architecture Specification

Status: **HARNESS-00 IMPLEMENTED / EXTERNAL REVIEW REQUIRED**
Scope: **documentation and architecture only**
Runtime status: **not implemented**

## 1. Purpose and boundaries

Jax needs two related but separate harnesses. They share durable-state,
provenance, safety, evaluation, and resumability principles, but they do not
share an execution loop or a source of authority.

### Engineering harness

The engineering harness enables Codex or another coding agent to work safely
in the repository. It must help an agent:

- understand Jax and find authoritative documentation;
- reconstruct current package, branch, state, constraints, and risks;
- respect runtime, scientific, safety, and ownership boundaries;
- select the smallest correct verification set;
- make incremental, reviewable changes;
- leave a durable handoff and completion evidence; and
- recover from a fresh session without relying on chat history.

### Jax runtime/research harness

The runtime/research harness surrounds JaxMind and research workflows. It must
control research objectives, task state, evidence and memory retrieval, context
construction and budgets, tool use, checkpoints, compaction, structured output,
evaluation, falsification, provenance, and observability.

The runtime/research harness is advisory and research infrastructure. It cannot
grant execution authority, approve a trade, mutate broker state, change risk,
promote exploratory observations to formal evidence, or activate a pilot.

### Current repository placement

The active platform is the ADR-0012 modular monolith:

- `cmd/trader` is the deterministic runtime and safety-critical boundary.
- `cmd/research` is the research, orchestration, backtest, and memory-tool
  boundary.
- Postgres and existing audit/provenance stores remain durable system-of-record
  dependencies.
- `services/ib-bridge` remains an explicit external boundary.

The future runtime/research harness belongs on the research side of this
boundary unless a separately reviewed contract proves that a read-only,
deterministic artifact is safe for another boundary. No HARNESS package may
introduce a research dependency into `cmd/trader`.

## 2. Non-negotiable principle: context is not storage

**Context is not storage.** The architecture must keep four concepts distinct:

- **Evidence** — source-backed information about the world, with identity,
  provenance, timing, freshness, and quality.
- **Memory** — durable lessons Jax has learned from previous forecasts,
  decisions, observations, and outcomes. Memory is not current evidence.
- **State** — what the current task or research process is doing: objective,
  constraints, completed work, unresolved questions, decisions, retries, and
  next action.
- **Context** — a deliberately constructed, bounded working package supplied
  to a model for one objective and invocation.

The prohibited architecture is:

```text
Evidence Store -> load everything -> model
```

The target architecture is:

```text
Research Objective
        |
        v
Task Controller
        |
        v
Context Builder
        |
        +--> relevant Evidence
        +--> relevant Memory
        +--> current State
        +--> constraints and policies
        +--> contradictions and counter-evidence
        |
        v
Bounded Context Package
        |
        v
JaxMind
        |
        v
Structured Research Output
        |
        v
Independent Evaluation
        |
   PASS / REVISE / ABSTAIN
```

The Context Builder is a selector and assembler, not a durable evidence store,
and a model transcript or compaction summary is never the canonical task state.

## 3. Design principles

1. **Repository and durable state are the system of record.** A zero-history
   agent must be able to reconstruct work from repository documents, structured
   task state, canonical evidence references, and audit records.
2. **AGENTS.md is a map.** It routes to authority, skills, boundaries, and
   verification. Detailed architecture, policy, status, plans, and evidence
   remain in their canonical documents.
3. **Minimal high-signal context.** A larger context window is not a reason to
   inject every related document or evidence item.
4. **Just-in-time retrieval.** Durable references are retained; details are
   retrieved when the current objective needs them.
5. **Long-horizon durability.** Restart, context exhaustion, compaction, model
   replacement, and tool failure must not destroy the ability to continue.
6. **Structured checkpoints.** Machine-readable state is preferred. Compaction
   is a derived working artifact, not canonical storage.
7. **Independent evaluation.** The producer of research reasoning cannot be
   the sole authority for sufficiency or completion.
8. **Falsification first-class.** Counter-thesis and invalidating-evidence
   retrieval are explicit operations, not optional prose.
9. **Observable invocations.** Important model calls must be traceable without
   duplicating canonical raw payloads unnecessarily.
10. **Model-independent contracts.** Interfaces describe objectives, evidence,
    state, budgets, outputs, and evaluation rather than quirks of one model.
11. **Complexity must earn its place.** Start with one controller, one bounded
    context builder, and one independent evaluator. Specialists are conditional
    on measured benefit.
12. **Safety is a separate authority.** Harness output is never execution
    authority and cannot override deterministic risk or human approval.

## 4. Conceptual contracts

These are architecture-level responsibilities, not Go type commitments. Each
contract must have a version, identity, schema validation, and a stable reference
strategy before implementation.

| Contract | Responsibility | Must not do |
| --- | --- | --- |
| `ResearchObjective` | Defines the question, decision purpose, scope, success criteria, time boundary, allowed tools, hard constraints, and invalidation questions. | Imply that a trade is approved or that evidence exists. |
| `TaskController` | Creates and advances task state, chooses the next bounded work unit, enforces retry/stop rules, requests retrieval, checkpoints progress, and routes output to evaluation. | Treat chat history as state, silently skip required stages, or authorize trading. |
| `TaskState` | Durable current-task record: objective/version, stage, completed and pending work, evidence references, decisions/reasons, open questions, known unknowns, constraints, failures, checkpoint, and next action. | Store a giant transcript or overwrite prior state without versioned history. |
| `ContextBuilder` | Selects and assembles objective-relevant evidence, counter-evidence, memory, state, tools, policies, and output instructions under an explicit budget. | Load all storage, hide contradictions, or make unsupported facts. |
| `ContextPackage` | Immutable, hashable, model-visible package for one invocation plus audit references. | Become a mutable cache or the system of record. |
| `ContextBudget` | Allocates, reserves, measures, and reports tokens/bytes across context classes and output allowance. | Spend all remaining model-window space without priorities or reservations. |
| `EvidenceRetriever` | Finds candidate and supporting evidence using provenance, relevance, quality, recency, causal relation, independence, contradiction, staleness, and uncertainty. | Rank by embedding similarity alone or collapse source identity. |
| `ContradictionRetriever` | Searches for disconfirming, invalidating, conflicting, and missing evidence under the same objective and time boundary. | Be a post-hoc “bear case” paragraph with no source references. |
| `MemoryRetriever` | Retrieves relevant prior lessons using regime, instrument, issuer, event, recency, reliability, outcome quality, and calibration. | Treat memory as current evidence or let it dominate current evidence. |
| `Checkpoint` | Versioned durable snapshot sufficient for safe continuation and evaluator review. | Replace canonical evidence or silently mark work complete. |
| `CompactionRecord` | Records how a working context was reduced, what references were retained, what was omitted/deferred, and what remained unresolved. | Delete canonical data, resolve uncertainty by compression, or become irreversible storage. |
| `ToolInvocationRecord` | Records requested tool, permission, arguments hash, start/end, outcome, error, output reference, and policy decision. | Grant a tool more authority than its permission tier. |
| `StructuredResearchOutput` | Model response with claims, evidence references, reasoning links, uncertainty, assumptions, open questions, invalidation criteria, and requested next action. | Emit executable order instructions or unsupported confidence. |
| `EvaluatorInput` | Binds objective, task-state version, context package reference, structured output, evidence/memory references, policies, and evaluation rubric. | Rely only on the producer's prose. |
| `EvaluatorResult` | Independent PASS, REVISE, or ABSTAIN decision with structured findings and required corrections. | Mean “trade approved” or replace human/risk approval. |
| `ContextProvenance` | Describes package assembly inputs, selected/omitted items, retrieval reasons, rankings, versions, hashes, budget decisions, and reconstruction method. | Duplicate giant payloads when canonical references suffice. |
| `HarnessRun` | Correlates objective, task state, invocations, checkpoints, context packages, tool records, output, evaluation, revisions, and operational metrics. | Become an alternative trade, evidence, or audit ledger. |

## 5. Runtime/research control flow

The smallest acceptable future loop is:

1. Create a versioned `ResearchObjective` and a durable `TaskState`.
2. `TaskController` selects one next stage and validates hard constraints.
3. `ContextBuilder` requests candidate evidence, contradiction evidence, and
   relevant memory independently.
4. The builder applies `ContextBudget`, reserves policy/objective/state and
   counter-evidence capacity, and emits an immutable `ContextPackage`.
5. JaxMind produces only `StructuredResearchOutput` against the declared schema.
6. The controller writes a checkpoint and submits `EvaluatorInput` to an
   independent evaluator.
7. `PASS` completes only the stated research objective; `REVISE` creates a
   bounded correction stage; `ABSTAIN` records insufficient/unsafe evidence.
8. Any downstream recommendation remains subject to deterministic risk,
   workflow, human approval, and paper/live gates already in Jax.

`PASS` is not a trading decision and cannot create an order, fill, position, or
execution instruction.

## 6. Context Package contract

The canonical package is immutable after creation. The following classification
is normative at the architecture level.

| Field group | Examples | Classification |
| --- | --- | --- |
| Identity | `ContextPackageID`, `TaskID`, `ObjectiveVersion`, package schema/version, created-at, parent package ID | Immutable identity/audit |
| Purpose | model purpose, requested output schema, objective statement, scope and time boundary | Model-visible; hash-covered |
| Controls | token budget, policy versions, permissions, safety constraints, required abstentions | Model-visible and audit |
| Selected material | selected evidence, counter-evidence, memory, current state, open questions, known unknowns, assumptions | Model-visible; each item carries canonical reference and selection reason |
| Tool surface | allowed tool references, prior tool-result references, deferred retrieval handles | Model-visible where needed; full invocation records audit-only |
| Continuity | checkpoint reference, retry/revision number, next-action contract | Model-visible and audit |
| Provenance | builder version, retrieval versions, ranking/selection decisions, omitted/deferred references, source hashes | Mostly audit-only; stable subset visible for citation |
| Derived integrity | content hash, token counts, truncation/degradation decisions, assembly timestamp | Derived/audit-only; hash covers model-visible bytes |

At minimum the package must include:

```text
ContextPackageID
TaskID
Objective + ObjectiveVersion
ModelPurpose
TokenBudget
Constraints
PolicyVersions
SelectedEvidence[]
CounterEvidence[]
SelectedMemory[]
CurrentState
OpenQuestions[]
KnownUnknowns[]
Assumptions[]
ToolReferences[]
CheckpointReference
Provenance
ContentHash
```

An item reference must preserve source identity, source timestamp, retrieval
time, freshness/status, selection reason, quality/uncertainty, and the canonical
location of the full artifact. A summary may be model-visible, but the source
reference and uncertainty cannot be removed merely to save tokens.

## 7. Context budgeting

Budgeting is deterministic or auditable. The package must report planned,
reserved, used, omitted, and deferred budget by class:

1. system and policy constraints;
2. objective and output contract;
3. current task state and checkpoint;
4. supporting evidence;
5. counter-evidence and contradiction reserve;
6. relevant memory;
7. tool results;
8. model working allowance;
9. structured-output allowance.

Hard constraints, objective, current state, and counter-evidence reserve are
protected allocations. Supporting evidence cannot consume them silently.

When material exceeds budget, the builder must:

- filter and rank by objective relevance, provenance quality, causal value,
  freshness, independence, contradiction, and uncertainty;
- preserve source references while using bounded summaries where safe;
- include high-value counter-evidence before low-value supporting evidence;
- record omitted and deferred items with the reason and retrieval handle;
- request additional evidence just in time when the task can safely continue;
- revise the task or return `ABSTAIN` when sufficient evidence cannot fit; and
- never imply that omitted evidence was reviewed.

The builder must not simply fill the model context window. Budget reductions are
an evaluation variable, and degradation must be measurable.

## 8. Evidence and contradiction retrieval

Evidence retrieval is provenance-aware and multi-factor. Selection considers:

- objective and claim relevance;
- source quality, issuer/instrument relevance, and causal relevance;
- source independence and corroboration;
- recency, information-state timestamp, freshness, and staleness;
- contradiction, invalidation power, uncertainty, and missingness; and
- retrieval cost and expected information value.

The harness keeps these categories distinct:

- **candidate evidence** — retrieved as potentially relevant;
- **supporting evidence** — selected evidence that supports a stated claim;
- **contradictory evidence** — evidence that conflicts with a claim or source;
- **invalidating evidence** — evidence that triggers a declared thesis failure;
- **unknown/missing evidence** — information required but unavailable or not
  yet retrieved.

Similarity may assist candidate retrieval, but it cannot be the sufficiency
gate. A source-backed claim must retain its provenance and information-state
boundary. Counter-evidence retrieval is a first-class operation with its own
budget, trace, and recall metric.

## 9. Memory architecture

Memory integrates with Jax Experience/Judgment and the existing review concepts
(`Docs/MEMORY_AND_REVIEW/`) without becoming a second evidence store.

Memory records are lessons derived from prior forecasts, decisions, observations,
reviews, and outcomes. A memory candidate is retrieved using:

- objective and task relevance;
- market regime similarity and regime validity interval;
- instrument, issuer, sector, and event-type similarity;
- recency and expiry/invalidation status;
- reliability and source-of-lesson quality;
- outcome quality and calibration history; and
- known contamination or outcome-leakage risk.

Memory is visibly labelled as memory in the package. It cannot satisfy a current
evidence requirement unless a current source independently supports the claim.
Controls include stale-memory expiry, regime mismatch penalties, leakage-safe
time boundaries, confirmation-bias counter-retrieval, analogy limits, and a
maximum memory allocation. Current evidence and unresolved unknowns outrank
historical analogy.

## 10. Durable state, checkpoints, and resumability

`TaskState` is the minimum fresh-session contract. It must include:

- objective and objective version;
- current stage and stage status;
- hard constraints and non-authorizations;
- completed work with evidence references;
- pending work and next action;
- open questions and known unknowns;
- contradictions and invalidation checks;
- decisions already made and their reasons;
- tool failures, retry state, and attempt count;
- latest checkpoint and context-package references; and
- task-state version, parent version, created-at, and content hash.

Every meaningful transition appends a version or records a parent version. A
fresh model session must be able to continue from the latest durable state with
zero transcript history. The model cannot declare work complete without a
state transition backed by the required evidence or evaluator result.

### Structured compaction

Compaction reduces working context while retaining objective, constraints,
unresolved contradictions, important evidence references, decisions and
reasons, open questions, next action, and provenance. A `CompactionRecord`
contains input checkpoint/package references, the compactor version, retained
and omitted references, uncertainty/contradiction preservation checks, and a
new content hash.

Compaction must not delete canonical evidence, turn uncertain claims into facts,
erase contradictions, silently drop constraints, or become irreversible
canonical storage. The original task state and evidence remain recoverable.

## 11. Tool control

Tools are registered by stable identifier, schema, version, permission tier,
cost/latency budget, timeout, cancellation semantics, and provenance policy.
Tool use is requested by the controller and recorded in `ToolInvocationRecord`.
Read-only research tools, artifact-generation tools, and any future mutating
tools require separate permission tiers. HARNESS-00 authorizes no tool runtime.

Required record fields include task/run ID, context-package ID, tool ID/version,
arguments hash, permission decision, start/end, retry, result/error code, output
reference, output hash, and whether the result was model-visible.

## 12. Structured output and independent evaluator

`StructuredResearchOutput` must separate claims, evidence references, inference,
assumptions, uncertainty, confidence rationale, contradictions, invalidation
criteria, missing data, and requested next action. Schema validation happens
before evaluation.

The evaluator receives an `EvaluatorInput` containing the objective, required
criteria, package reference/hash, structured output, selected and omitted
evidence references, memory references, state version, policy versions, and
the independent rubric. It evaluates at least:

- `ObjectiveSatisfied`;
- `EvidenceSufficient`;
- `EvidenceQualityAcceptable`;
- `ContradictionsAddressed`;
- `CausalChainSupported`;
- `UnsupportedClaims`;
- `ConfidenceSupported`;
- `UncertaintyRepresented`;
- `InvalidationDefined`;
- `PolicyCompliant`; and
- `PrematureCompletionDetected`.

The evaluator returns `PASS`, `REVISE`, or `ABSTAIN`, plus structured reasons,
missing evidence, required corrections, confidence/uncertainty findings, and
its own version and provenance. Evaluator PASS means only that the stated
research output met the research rubric. It never means trade approval,
execution authority, formal-paper promotion, or live authorization.

## 13. BlackBox, audit, and provenance integration

The harness should extend existing Jax BlackBox/audit direction by linking
stable IDs and hashes rather than creating duplicate raw stores. The target
chain is:

```text
prediction/recommendation
  -> HarnessRun
  -> TaskState version
  -> ContextPackage + hash
  -> selected/omitted evidence references
  -> selected memory references
  -> ToolInvocationRecords
  -> model/version and structured output
  -> EvaluatorResult and revisions
  -> later decision/workflow record
  -> outcome and review/memory lesson
```

For an important historical decision, the system must answer: “What exactly did
the model know?” Exact model-visible bytes are preferred when storage policy
permits; otherwise, immutable canonical component references, hashes, builder
version, deterministic assembly rules, selection decisions, and model/prompt/
tool versions must permit faithful reconstruction. Canonical raw evidence and
existing audit records remain authoritative; `HarnessRun` links to them.

If reconstruction is impossible, the run is not reconstruction-complete and
must be labelled accordingly. Missing provenance cannot be filled from model
memory or a later source fetch.

## 14. Engineering harness relationship

The engineering harness is not a JaxMind research loop. Its durable package is
the repository plus structured engineering state:

```text
Repository authority map
  -> task/package state
  -> scoped change plan
  -> selected verification commands
  -> incremental edits
  -> diff and test evidence
  -> durable handoff
```

Its completion authority is repository policy, review evidence, and passing
verification—not a model's statement that the task “looks done.” The current
gap analysis and roadmap define the documentation and future implementation
needed to make that flow machine-checkable.

## 15. Isolation rule for PAPER-02

`PAPER-02-2026-01` is active, exploratory, and frozen. No HARNESS
implementation may enter its runtime, decision path, context, evidence
selection, policies, thresholds, sample accounting, or 50-opportunity ledger.

HARNESS work must not create, admit, replay, relabel, or evaluate a PAPER-02
prospective observation. Any future integration into trading or research
decision paths requires a separately reviewed post-PAPER-02 gate with explicit
identity, behavior, safety, and scientific review.

## 16. Explicit non-scope of HARNESS-00

This specification does not implement `ContextBuilder`, `TaskController`,
JaxMind changes, evidence retrieval, memory retrieval, evaluators, tools,
multi-agent orchestration, BlackBox persistence, database migrations, trading,
risk, candidate logic, strategy logic, broker execution, formal forward paper,
Phase 13, or any PAPER-02 change.
