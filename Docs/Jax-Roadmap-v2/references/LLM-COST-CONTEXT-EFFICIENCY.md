# LLM Cost and Context Efficiency — Architecture Reference

## Purpose

Define provider-neutral design principles for keeping Jax's AI layer economically bounded, reproducible and efficient without weakening evidence quality or deterministic safety.

This document is architecture guidance. It does not authorize implementation.

## Design goals

Jax should avoid unnecessary LLM calls; minimise tokens on necessary calls; maximise safe provider-side cache reuse without changing semantics; reuse exact prior results only when the full validity key still matches; route work to the least expensive model proven adequate; escalate only under explicit policy; expose usage/cost as first-class evidence; preserve provenance/replayability; keep time-sensitive evidence fresh; and fail closed on budget/configuration ambiguity.

## The four efficiency layers

### 1. No-call reuse

Before inference, determine whether Jax already has a result that is exactly reusable.

A validity key should include at least:

- task/operation identity;
- model/provider compatibility policy;
- prompt version;
- output-contract version;
- relevant policy/resolver/tool versions;
- canonical input/evidence fingerprint;
- data vintage/freshness identity;
- material runtime options;
- safety classification.

Conceptually:

```text
hash(
  task
  + prompt_version
  + output_contract_version
  + policy_versions
  + model_compatibility_identity
  + evidence_fingerprint
  + data_vintage
  + material_runtime_options
)
```

For market/news/research data, timestamps alone are not sufficient. Prefer content hashes, source versions/vintages and explicit freshness policy.

### 2. Provider prompt caching

Where supported, structure requests so large stable prefixes can be recognised by the provider:

```text
stable system policy
stable output/tool contracts
stable task instructions
-------------------------
dynamic evidence
dynamic event/user payload
```

Do not distort semantics or add meaningless padding to chase cache discounts. Avoid volatile timestamps/UUIDs in otherwise stable prefixes unless semantically necessary. Never assume a cache hit unless the provider reports it.

Normalised evidence may include cached-input/read tokens, cache-write tokens, uncached input tokens where derivable, cache age/TTL where exposed, and cache read/write cost.

### 3. Context minimisation

Send the evidence required for the decision, not everything Jax has stored.

Use typed evidence packets, task-specific retrieval, version filtering, duplicate removal, canonical snippets, deterministic pre-computed metrics, incremental changed-evidence sets, provenance-preserving summaries and task-specific context ceilings.

Do not replace primary evidence with lossy summaries where exact wording or numbers are material.

### 4. Capability/cost routing

Model selection should be policy-driven rather than buried in business logic.

```text
deterministic code / exact reusable result
        ↓
local or cheapest proven-adequate model
        ↓
strong low-cost hosted model
        ↓
higher-capability model
        ↓
frontier escalation for exceptional ambiguity
```

Escalation triggers may include invalid structured output after the approved retry, unresolved contradiction, ambiguity, a task class proven to exceed the cheaper model, a validated low-confidence condition, or explicit operator request.

Each routing tier needs evaluation evidence. Cost alone never establishes adequacy.

## Cache classes

### Exact result cache — preferred

Reuse only when the canonical validity key matches. This is the default production reuse mechanism.

### Deterministic intermediate cache

Cache expensive deterministic products such as parsed documents, compatible embeddings, normalised evidence packets, financial metrics, source extracts, hashes and retrieval indexes. These reduce repeated preprocessing and LLM context without caching stochastic judgments.

### Semantic result cache — high risk, later only

Near-duplicate market questions can differ materially. Do not enable semantic-result reuse until separately evaluated for false-hit rate, time sensitivity, issuer/instrument ambiguity, market regime changes, prompt/output-contract compatibility, source-vintage compatibility and auditability.

Default: **disabled until proven safe**.

## Context-budget policy

Each task class should eventually define:

- target/max input tokens;
- max output tokens;
- max reasoning tokens where controllable;
- max retrieved sources/chunks;
- max retries;
- maximum model tier;
- maximum expected cost;
- explicit oversize behaviour.

Oversize handling should first remove irrelevant/duplicate evidence deterministically, then use authorised provenance-preserving compaction, split the task if semantics permit, escalate only if policy permits, or stop for operator decision. Silent truncation of material evidence is unacceptable.

## Durable research and memory

Long-running research should not resend the complete historical transcript. Persist structured state: objective, completed steps, evidence inventory/fingerprints, unresolved questions, contradictions, provisional conclusions, tool outputs, checkpoint summary and model/prompt versions.

On resume, reconstruct the smallest sufficient context from durable state plus new evidence. Conversation transcript is not the canonical research database.

## Cost model

Provider prices are temporally unstable and must not be architectural constants.

At execution time record provider/model, pricing source/effective date, input rate, cached-input/read rate, cache-write rate if applicable, output rate, reasoning billing rules, batch/priority tier, currency, calculated cost and ambiguous liability.

Support both conservative pre-call liability and post-call actual cost from reported usage. Missing/contradictory required usage should fail closed for experiments and be explicitly marked ambiguous under production policy.

## Budget hierarchy

Eventually support per-request, per-task/run, per-research-session and daily/monthly budgets, plus model-tier escalation ceilings. A hard budget is a control, not merely a dashboard figure.

## Telemetry

Expose at least requests, avoided requests via exact reuse, provider/model/task counts, input/cached/cache-write/uncached/output/reasoning tokens, cache-hit ratio, retries, invalid-output rate, cost by provider/model/task, cost per successful recommendation/research output, savings from exact reuse/provider caching, escalation count/reason and budget rejections.

Do not invent provider-cache savings when equivalent usage/rate data is unavailable.

## Replay requirements

Historical AI evidence must remain explainable if provider pricing changes, model aliases move, provider cache state disappears or application cache entries expire. Record the original call/reuse decision and usage metadata; do not depend on provider cache state for replay.

## Security and privacy

Never cache API keys, authorization headers, raw credentials or sensitive connection strings. Cached research evidence inherits the sensitivity/retention requirements of its source data.

## Financial-data freshness

Caching is not permission to use stale evidence. Every reusable class requires an invalidation/freshness policy. Filing/version identity, economic-data vintages, news time sensitivity, market timestamps and portfolio state must be respected. Last-known-good data must carry age/freshness metadata and never silently masquerade as current.

## Benchmark integrity

Unless a benchmark explicitly evaluates caching:

- no application-level result reuse;
- no semantic reuse;
- every case genuinely invokes the model;
- no prompt change purely to improve cacheability after seeing results;
- provider-side caching may occur naturally and is measured rather than optimised;
- evidence distinguishes cached/uncached usage when exposed.

## Recommended implementation order

1. usage/cost observability;
2. task-specific context budgets;
3. exact deterministic/result fingerprinting;
4. exact safe reuse;
5. provider prompt-cache-aware request layout;
6. incremental research state;
7. validated model routing/escalation;
8. batch/offline optimisation;
9. semantic reuse only if separately proven worthwhile and safe.

## Anti-patterns

Reject giant universal prompts, whole-history dumping, agent transcript as canonical memory, arbitrary premium escalation, permanent hard-coded prices, silent alias changes, incomplete cache keys, cross-vintage reuse, stale price/news reuse, hidden retry cost, cached recommendations bypassing current risk checks, and benchmark prompt optimisation after seeing results.
