# Phase 08 Requirement — LLM Cost and Context Efficiency Control Plane

## Status

Cross-cutting requirement to be incorporated into Phase 08 work-package planning. This file does not add or authorize a new numbered work package.

## Objective

By the end of Phase 08, Jax's controlled AI/tooling layer should treat inference capacity and spend as explicit, auditable resources.

The system should avoid unnecessary calls, bound context, exploit safe provider caching, route tasks to proven-adequate models, and preserve deterministic safety/replay guarantees.

## Required capability groups

### A. Usage normalization

Provider adapters should map available provider usage into a common Jax record while retaining raw provider usage. Common concepts include input, cached-input/read, cache-write, uncached input when derivable, output and reasoning tokens; request/retry counts; provider/model; finish status; pricing snapshot; calculated cost; and ambiguous/unreported usage state.

Do not fabricate fields a provider does not expose.

### B. Exact result reuse

Provide exact-result reuse keyed by the full validity identity: task, prompt version, output contract, model/provider compatibility rule, evidence/input fingerprint, data vintage/freshness, relevant policy/tool versions and material runtime settings.

Reused outputs retain original provenance and are explicitly labelled as reused rather than newly inferred.

### C. Provider prompt-cache optimisation

Where supported, keep semantically stable instruction/contract prefixes stable, place volatile evidence after stable context, record cache reads/writes, never distort semantics for cache hits, and never make provider cache state a replay dependency.

### D. Model routing

Implement a deterministic, versioned routing policy informed by evaluation evidence. Candidate tiers may include deterministic/no-LLM, local, cheap hosted, strong hosted and premium escalation.

### E. Escalation

Escalate only for defined reasons such as invalid output after permitted retry, unresolved ambiguity/contradiction, a task class proven to exceed the cheaper tier, a validated low-confidence condition or operator request.

Record original tier, escalation reason, next tier, incremental cost and final disposition.

### F. Budget enforcement

Support hard run/task budgets before and during inference. Distinguish conservative pre-call liability, actual reported cost, ambiguous/unreported liability and remaining budget. Budget exhaustion must stop further calls.

### G. Durable-agent context compaction

Persist structured checkpoint state independently from chat transcript: objective, plan, completed steps, evidence inventory, tool outputs, unresolved items, checkpoint summary and model/prompt/tool versions. Reconstruct the minimum sufficient context on resume.

### H. Batch/offline paths

For non-urgent evaluations, backfills or scheduled research, evaluate provider batch discounts where available while preserving provenance, model identity and budget controls.

### I. Observability

Expose requests, avoided requests, cached vs uncached input, cache writes, output/reasoning tokens, retry waste, cost by provider/model/task, cost per successful output, provider-cache/application-reuse savings, escalations and budget rejections.

## Safety rules

No cache/reuse/routing feature may bypass current evidence freshness checks, deterministic eligibility, portfolio/risk evaluation, human approval, execution safety or broker/execution gates.

A cached recommendation is historical reasoning evidence, not authority to act under current market/portfolio state.

## Evaluation requirements

Before a cheaper model becomes a routing default, prove adequacy on frozen task-specific benchmarks, compare invalid/error behaviour, measure escalation rate and actual token/cost profile, and record model alias/identity limitations.

Before exact-result reuse, prove cache-key completeness, invalidation on prompt/model/evidence/policy changes, no cross-vintage reuse, and replay/audit labelling.

Before semantic result reuse, require separate architecture review and dedicated false-hit/freshness tests. Default semantic cache state: disabled.

## Phase 08 completion expectation

Phase 08 should not be considered complete if durable agents can recursively call paid models without explicit task budgets, per-call usage evidence, a routing policy, checkpoint/context limits, spend telemetry and deterministic stop behaviour.
