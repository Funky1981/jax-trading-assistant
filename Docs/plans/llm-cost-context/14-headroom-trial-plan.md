# 13 — Headroom Trial Plan

## Purpose

Evaluate Headroom as an optional compression layer for token saving.

Headroom should not replace LiteLLM or Jax cost controls.

## Target Architecture

```text
Jax evidence/context builder
  ↓
safe-compression policy
  ↓
Headroom for Zone C context only
  ↓
LiteLLM gateway
  ↓
model
```

Alternative:

```text
Jax
  ↓
LiteLLM with Headroom callback
  ↓
model
```

Use whichever is easier to observe and disable.

## Recommended Trial Scope

Start with non-trading-critical workflows:

```text
developer coding sessions
test logs
build logs
search results
documentation Q&A
historical research prose
daily digest source material
post-trade narrative reflection
```

Do not start with:

```text
live approval packets
broker/order state
risk calculations
priced-in verdict generation
stop-loss/target messages
```

## Trial Steps

### Step 1 — Local Install

```text
pip install "headroom-ai[all]"
```

or Docker:

```text
docker run -p 8787:8787 ghcr.io/chopratejas/headroom:latest
```

### Step 2 — Coding Agent Trial

Test with Codex/Claude Code-style sessions first.

Measure:

```text
input tokens before
input tokens after
output quality
missing details
latency
failure rate
```

### Step 3 — Docs/Search Trial

Run Headroom on large docs, search results, test logs, and build logs.

### Step 4 — Jax Historical Research Trial

Use only historical research summaries.

### Step 5 — Jax Evidence Bundle Safety Test

Use fixed golden inputs.

Compare:

```text
without Headroom
with Headroom
```

Check whether the model answer preserves:

```text
symbol
timestamp
priced-in verdict
risk
stop-loss
take-profit
guardrail state
source references
```

## Evaluation Table

For each test case record:

```text
case_id
task_type
original_tokens
compressed_tokens
saving_pct
latency_ms
model_output_pass
missing_facts
numeric_errors
retrieval_needed
cost_saved_estimate
decision
```

## Keep / Disable Rules

Keep Headroom for a workflow only if:

```text
token saving >= 30%
no trading-truth loss
no numeric distortion
latency acceptable
original retrieval works
tests are repeatable
```

Disable Headroom if:

```text
any Zone A field is altered
source references are lost
numeric details are distorted
approval summary changes meaning
retrieval is unreliable
latency harms live workflow
```

## Recommended Initial Use

```text
YES:
- Codex/local dev sessions
- docs Q&A
- logs
- historical prose summaries

NO:
- live trade approval
- broker state
- stop-loss/take-profit
- position sizing
```

## Acceptance Criteria

- Headroom is optional and feature-flagged.
- It is disabled by default for live approval workflows.
- Token savings are measured.
- Accuracy is measured.
- Rollback is one config change.
