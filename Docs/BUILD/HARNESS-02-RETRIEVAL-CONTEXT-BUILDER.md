# HARNESS-02 — Retrieval and Context Builder

Status: **IMPLEMENTED / EXTERNAL REVIEW REQUIRED**.

This package implements the deterministic, offline retrieval/context-selection
seam described by the canonical harness architecture. It does not call a
model, integrate with JaxMind, evaluate research, persist task state, or enter
the active PAPER-02 runtime.

## Boundary and audit findings

The architecture audit found canonical evidence/provenance abstractions in the
existing research and exploratory-paper modules, and existing memory concepts
in the memory/context packages. The audit also found no safe, reviewed
production adapter that could expose those stores through the HARNESS-01
reference contracts without changing their schemas or semantics. HARNESS-02
therefore adds interfaces and deterministic offline fixture adapters only. It
does not create an evidence store, memory store, audit ledger, datastore,
service, or competing retrieval implementation.

The implementation is in `internal/modules/contextbuilder`, on the
research/platform side of the modular monolith. `cmd/trader` does not import
the package.

## Retriever contracts

The package defines read-only `EvidenceRetriever`, `ContradictionRetriever`,
and `MemoryRetriever` interfaces. Results contain canonical references,
retriever identity/version, retrieval time, policy eligibility, ranking
factors, temporal metadata, and optional stable on-demand references. A
retriever failure is returned as a build failure; an empty result is represented
distinctly from a search that was not performed.

Supporting evidence and counter-evidence are separate operations and separate
budget classes. Counter-evidence supports `CONTRADICTORY`, `INVALIDATING`, and
`UNKNOWN_MISSING` classifications. Memory candidates remain
`MemoryReference` values and cannot satisfy evidence requirements or
corroboration.

## Selection and ranking

Selection considers objective, subject/issuer/instrument, event, causal,
source-quality, recency, independence, corroboration, staleness, uncertainty,
policy eligibility, and temporal integrity. It is not similarity-only. The
rank score is deterministic and auditable; ties are broken by estimated units,
canonical identifier, and record identifier. Duplicate keys are collapsed
before selection, while independent source identities can receive a
corroboration preference only when independence is explicitly known.

An explicit reference/as-of time is required. Future-dated material is excluded
fail-closed. Stale material, unknown observation timing, and unknown publication
timing are represented explicitly and excluded when the configured policy
requires exclusion. Hidden wall-clock time is not used for ranking.

## Budget and omission policy

The builder uses the HARNESS-01 `ContextBudget` contract and does not call a
tokenizer. Candidate sizes use deterministic supplied abstract units. System
policy, objective, task state, supporting evidence, counter-evidence, memory,
tool outputs, working allowance, and structured-output allowance remain
separate budget classes.

Mandatory constraints and required evidence are protected. A mandatory
counter-evidence reserve cannot be consumed by supporting evidence; if it
cannot be satisfied, the build fails closed. Lower-value, duplicate, stale,
future, policy-ineligible, or budget-excluded items are recorded with typed
omission reasons. If a useful omitted item remains retrievable and policy
allows it, a stable deferred/on-demand reference is recorded without copying
the raw payload.

The builder does not claim that selected context is sufficient for research or
trading. It produces a `ContextPackage` plus an audit-only `BuildReport`.

## Context identity and report

The builder reuses the HARNESS-01 canonical ContextPackage SHA-256 contract;
it does not introduce a second hash. Identical semantic inputs, candidate
metadata, explicit reference time, policy, budget, and deterministic selection
produce the same model-visible content and hash. Audit-only volatile metadata
does not change that content hash. The report records builder/retriever
versions, input identities, candidate and selection counts, budget usage,
search states, omission reasons, deferred references, missing information,
warnings, and final package identity/hash.

The package contains 53 named offline behavioral cases covering overload,
contradiction hiding, missing data, budget reduction, duplicates versus
independent corroboration, temporal/future leakage, deterministic ordering and
hashing, memory/evidence separation, deferred references, report identity,
retriever failure, and PAPER-02 isolation.

## Production status and non-authorizations

Production evidence and memory adapters are deferred until a separately
reviewed adapter can reference existing canonical stores without duplicating or
changing them. HARNESS-03 owns checkpoint persistence and compaction. Future
packages own full memory architecture, evaluator execution, reconstruction
tooling, and model/JaxMind integration.

HARNESS-02 does not authorize any change to PAPER-02 context or evidence
selection, candidate/risk/entry/exit/strategy behavior, formal forward paper,
broker execution, live trading, Phase 13, or multi-agent orchestration.
