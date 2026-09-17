# Deferred Context Engineering Roadmap Record

Status: **RECORD / DEFER — DO NOT IMPLEMENT**

This is a planning record only. It does not authorize Context Engineering
implementation, a new AI runtime, a new memory/RAG subsystem, PAPER-02
activation, formal forward paper, live execution, or Phase 13.

The canonical roadmap location is
[`Docs/ROADMAP.md`](ROADMAP.md#deferred-context-engineering). This record holds
the deferred architectural contract and validation requirements referenced by
that roadmap section.

## Architectural placement and prerequisites

Context Engineering is a future cross-cutting capability spanning the existing
Phase 06 Research & Recommendation Engine and Phase 08 Controlled AI Tools &
Durable Research Agents. It must not jump ahead of these prerequisites:

- trustworthy source-backed evidence acquisition, provenance, freshness,
  normalisation and contradiction handling;
- canonical evidence packets and relevant-evidence selection;
- durable research task/checkpoint state independent of chat transcripts;
- provenance-aware research memory with safe invalidation/reuse;
- forecasting, uncertainty and calibration contracts;
- BlackBox/auditability and reconstructable model-invocation records; and
- Experience/Judgement interfaces that consume reviewed evidence and outcomes
  without becoming an unbounded hidden context store.

The current Phase 06 and Phase 08 pack already records related requirements in
`06-research-recommendation-engine/LLM-CONTEXT-BUDGET-REQUIREMENTS.md`,
`06-research-recommendation-engine/WP-06.02-research-planner-context-builder.md`,
`08-controlled-ai-tools-durable-research-agents/WP-08.04-durable-research-task-checkpoint-state.md`,
`08-controlled-ai-tools-durable-research-agents/WP-08.07-research-memory-with-provenance.md`,
and `08-controlled-ai-tools-durable-research-agents/WP-08.08-agent-evaluation-harness.md`.
This record consolidates the future dependency and does not claim that a new
Context Builder has been implemented.

## Canonical principle

**Context is not storage.** Raw evidence, historical research, large tool
outputs and long-term memory belong in durable stores. Models receive
deliberately assembled, bounded, provenance-aware context appropriate to the
current objective.

The target shape is:

```text
Research Objective
  -> Context Builder
  -> relevant Evidence + relevant Memory + current State
  -> bounded Context Package
  -> JaxMind
  -> structured output
```

The anti-pattern is:

```text
Evidence Store
  -> load everything
  -> model
```

## Distinct concepts

- **Evidence** is source-backed information about what happened.
- **Memory** is what Jax has learned from previous forecasts, decisions and
  outcomes.
- **State** is what Jax is currently doing: objective, constraints, completed
  work, outstanding work, uncertainties and next actions.
- **Context** is a deliberately constructed, bounded working package assembled
  from durable stores for the current model invocation.

These remain separate contracts. Context Engineering must not collapse them
into one generic memory or RAG subsystem.

## Future Context Package contract

Before implementation becomes eligible, a versioned Context Package contract
must define exactly what JaxMind and research models receive. Construction must
be deliberate, bounded and reproducible. It must not depend on arbitrary
transcript accumulation or loading complete research history.

The contract must include, as applicable, the objective, current state,
selected evidence, relevant memory, constraints, uncertainties,
contradictions, model/prompt/tool versions, budget accounting and structured
output requirements.

## Context budgets and selection

The future Context Builder must enforce explicit working-context budgets:

- small, high-signal model-visible context;
- large evidence and artifacts retained externally;
- additional retrieval on demand;
- no assumption that larger context windows solve long-horizon research; and
- measured model quality and cost under bounded context.

Evidence selection must be objective-relevant and retain evidence IDs, source
provenance, timestamps/freshness where applicable, retrieval reason,
relevance, contradictory evidence and uncertainty/unknowns. Selection must not
construct a support-only thesis context.

## Durable task state and checkpointing

Long-horizon research state must persist outside conversational model memory.
The durable state must include:

- objective and hard constraints;
- completed and outstanding work;
- established facts and evidence references;
- uncertainties and contradictory evidence;
- decisions; and
- next actions.

The model is not the source of truth for whether work was completed. Completion
must be supported by durable state and evidence.

Compaction must not use an unconstrained “summarise everything so far” as its
primary contract. A future structured checkpoint must contain at minimum:

- objective;
- hard constraints;
- established facts;
- evidence references;
- inferences;
- contradictory evidence;
- uncertainties;
- decisions;
- completed work;
- remaining work; and
- next action.

Compaction must preserve the underlying evidence/history and keep original
material recoverable.

Persistent memory must be retrieved only when relevant. Memory retrieval and
evidence retrieval remain architecturally distinct.

## Context provenance and auditability

Significant model-visible context items should, where practical, retain:

- source/evidence identity;
- timestamp;
- retrieval reason;
- relevance; and
- freshness.

For important forecasts and decisions, Jax should eventually reconstruct the
exact model-visible Context Package used. This must integrate with the existing
BlackBox/auditability direction rather than create a duplicate audit trail.

## Mandatory Context Quality Evaluation

Context Quality Evaluation is mandatory before any future implementation is
accepted. It must include forced checkpoint/compaction and resume tests.

The evaluation must determine whether Jax:

- retains the objective and hard constraints;
- resumes correctly;
- retrieves omitted evidence when needed;
- keeps evidence, memory and state distinct;
- preserves contradictory evidence;
- avoids repeating completed research;
- avoids inventing completed work;
- avoids premature completion;
- reconstructs model-visible context; and
- remains stable when context selection changes.

It must also measure, where applicable:

- model quality under bounded context;
- token/context cost;
- retrieval volume;
- unnecessary-context rate; and
- evidence coverage.

A larger or more expensive model is not assumed to be required.

## Integration and authority boundaries

Future implementation must integrate with existing or future JaxMind, research
orchestration, evidence acquisition/normalisation, forecasting, uncertainty,
calibration, BlackBox/auditability, Experience/Judgement and persistent memory
contracts. It must not create a disconnected second AI architecture.

Context output cannot grant execution authority, bypass human approval, change
risk policy, create orders/fills/positions, promote exploratory evidence to
formal evidence, or activate PAPER-02.

## Current status and non-authority

This package records and defers the requirement. It does not implement it.

- PAPER-02 remains `READY_FOR_EXTERNAL_REVIEW / NOT ACTIVE` with sample `0 / 50`.
- PAPER-02 remains blocked only because genuine prospective event intake is
  disabled/unconfigured.
- No event-source remediation is included here.
- No PAPER-02 opportunity, pilot, order, fill, trade or position is created.
- No formal forward paper, demonstrated edge, live trading or Phase 13 work is
  authorized.
