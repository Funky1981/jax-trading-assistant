# 01 — Target Architecture

## Goal

Add a dedicated LLM cost and context management layer that sits between Jax application workflows and external model providers.

This layer must make model usage predictable, auditable, cache-friendly, compact, and policy-safe.

## Architecture Overview

```text
News / market / research inputs
        ↓
Event classification and deterministic pre-checks
        ↓
Prompt + context layer
        ↓
Cost governor
        ↓
Model router
        ↓
Provider client
        ↓
Output validation
        ↓
Evidence bundle / memory artifact / decision ledger
```

## Main Components

### 1. Prompt Package

A single object describing the full model request before it is sent.

It should include:

- task type
- provider
- model
- cacheable static prefix
- semi-dynamic retrieved memory
- dynamic event data
- response schema
- estimated tokens
- estimated cost
- correlation IDs

### 2. Cacheable Prefix Builder

Builds the stable prompt prefix used across similar calls.

Contains:

- Jax identity
- role boundaries
- ETF-only rules
- paper-trading rules
- advisory-only AI rules
- risk policy summary
- output schema
- forbidden actions
- validation expectations

The prefix should change rarely. If it changes every call, prompt caching becomes mostly useless.

### 3. Dynamic Context Builder

Builds the part of the prompt that changes every request.

Contains:

- current event
- current market snapshot
- current candidate ETF
- current evidence bundle
- current guardrail status
- current open positions if relevant
- retrieved memory references

### 4. Context Compaction Service

Turns raw material into smaller reusable artifacts.

Examples:

- raw article set → event summary
- long research run → compact research artifact
- final decision → decision ledger memory
- failed trade idea → walk-away lesson
- repeated market regime notes → regime memory

### 5. Memory Retrieval Service

Retrieves only relevant artifacts.

It must not dump full history into every prompt.

Retrieval should filter by:

- task type
- strategy id
- ETF symbol
- event type
- age
- quality score
- outcome status
- source confidence

### 6. Model Router

Selects the cheapest acceptable model for the task.

Example routing:

| Task | Preferred route |
|---|---|
| event classification | cheap/local model or deterministic code |
| source summarisation | cheap model |
| long event research | Qwen long-context route |
| final trade reasoning | strongest reasoning route |
| adversarial critique | strongest critic route |
| compaction | cheap/local model |
| schema cleanup | cheap model or deterministic parser |

### 7. Cost Governor

Blocks model calls that exceed limits.

Limits should exist at:

- per call
- per event
- per strategy
- per day
- per month
- per provider
- per model route

### 8. Usage Logger

Records planned and actual usage.

Should capture:

- task type
- provider
- model
- estimated input tokens
- estimated output tokens
- actual input tokens
- actual output tokens
- cached input tokens if provider reports them
- estimated cost
- actual cost if available
- cache eligible flag
- blocked flag
- block reason

## Suggested Go Package Shape

```text
internal/modules/llmcontext/
  prompt_package.go
  prompt_builder.go
  cacheable_prefix.go
  dynamic_context.go
  compaction.go
  memory_retrieval.go
  model_router.go
  cost_governor.go
  usage_logger.go
  provider_client.go
  noop_provider.go
```

If the repo later moves toward a cleaner domain/application split, these can be separated. For now, keep it pragmatic and aligned with the existing Go module shape.

## Core Interfaces

```go
type PromptPackage struct {
    TaskType string
    Provider string
    Model string
    CacheablePrefix string
    RetrievedMemory string
    DynamicContext string
    ResponseSchema string
    EstimatedInputTokens int
    EstimatedOutputTokens int
    EstimatedCostUSD float64
    CorrelationID string
}

type PromptBuilder interface {
    BuildPrompt(task LLMTask) (PromptPackage, error)
}

type ModelRouter interface {
    SelectRoute(task LLMTask, budget BudgetState) (ModelRoute, error)
}

type CostGovernor interface {
    CanRun(pkg PromptPackage, route ModelRoute) (CostDecision, error)
}

type ContextCompactor interface {
    Compact(input RawContext) (MemoryArtifact, error)
}

type UsageLogger interface {
    RecordPlanned(pkg PromptPackage, decision CostDecision) error
    RecordActual(result LLMResult) error
}
```

## Safety Boundary

The LLM context layer is not allowed to execute trades.

It may produce:

- summaries
- rankings
- critiques
- recommendations
- decision packets
- memory artifacts

It must not produce:

- broker order commands
- approval commands
- live-mode enablement
- position-size increases
- stop-loss changes after approval

Existing AI output validation and deterministic guardrails remain the final safety boundary.

## Acceptance Criteria

- LLM workflows depend on PromptPackage, not ad-hoc strings.
- PromptPackage separates cacheable prefix from dynamic context.
- CostGovernor runs before every provider call.
- UsageLogger records planned calls, blocked calls, and completed calls.
- ModelRouter can route at least scanner, research, final judge, critic, and compaction tasks.
- No provider client is called directly from trading or approval code.
- Tests use a no-op/fake provider and never require live API keys.