# 07 — Separate Systems and Control Layer

## Purpose

This document defines the final intended shape for using World Monitor alongside Jax while keeping both systems independent.

World Monitor should remain a standalone personal intelligence dashboard. Jax should remain the research, evidence, and candidate-trade engine. The systems should communicate only through a small controlled adapter that sends research triggers into Jax.

## Final Shape

```text
World Monitor
= awareness radar

Jax
= research + evidence + trade candidate engine

World Monitor → Jax Adapter
= controlled bridge between them
```

End-to-end flow:

```text
World Monitor
  ↓
World Monitor → Jax Adapter
  ↓
Jax Research Inbox
  ↓
Jax evidence checks
  ↓
Jax candidate trade, if evidence passes
  ↓
Human approval or rejection
```

## Non-Negotiable Boundary

World Monitor must never create orders, execution instructions, or final trade decisions.

World Monitor may only emit:

```text
research_trigger
awareness_event
source_cluster
high_severity_alert
```

World Monitor must not emit:

```text
trade_instruction
position_size
broker_order
risk_override
live_execution_request
```

## Jax Research Inbox

Jax should receive World Monitor items into a holding queue rather than the core trading flow.

Suggested statuses:

```text
new
ignored
researching
candidate_created
rejected
archived
```

This creates a clean review area and prevents noisy external alerts from polluting the core trading path.

## Quality Gate Before Jax Research

The adapter should send an event to Jax only if the event passes a pre-filter.

Minimum gate:

```text
- event matches an allowed theme
- timestamp is fresh
- source count >= 2, unless source is official/tier-1
- source quality passes
- affected ETF mapping can be explained
- event is not a duplicate of a recent trigger
```

Allowed phase-one themes:

```text
rates
inflation
central banks
oil / energy
war / geopolitical escalation
banking / credit stress
AI / semiconductors
cyber outage
supply chain
market panic
major commodity move
```

## Source Quality Scoring

Every incoming event should carry source-quality metadata.

Suggested tiers:

```text
Tier 1 = official sources, central banks, government agencies, Reuters, AP, BBC
Tier 2 = CNBC, Financial Times, MarketWatch, Yahoo Finance, strong regional sources, major think tanks
Tier 3 = aggregators, repost sites, noisy blogs, weak single-source claims
```

Suggested rule:

```text
Prefer 2+ Tier 1 sources,
or 1 Tier 1 source plus clear market confirmation,
or official source directly relevant to the event.
```

## Duplicate Clustering

World Monitor may produce many headlines about the same event. The adapter should group them before sending to Jax.

Example:

```text
10 Fed headlines
→ 1 Jax research trigger
```

Cluster by:

```text
canonical event key
time window
named entities
theme
region
possible affected ETFs
semantic headline similarity
```

## Explainable Confidence

The adapter must not send a bare confidence score without explanation.

Bad:

```json
{"confidence": 0.81}
```

Good:

```json
{
  "confidence": 0.81,
  "confidenceReasons": [
    "4 independent sources",
    "event is macro/rates related",
    "likely affects QQQ, TLT and SPY",
    "published within the last 20 minutes",
    "average source tier is high"
  ]
}
```

A confidence number without supporting reasons is decoration, not evidence.

## Severity Levels

World Monitor events should be normalised into severity levels before Jax receives them.

```text
low      = awareness only, do not send to Jax by default
medium   = send to Jax inbox
high     = trigger immediate Jax research
critical = trigger Jax research and notify operator
```

Examples:

```text
low      = product rumour or weak analyst opinion
medium   = Fed speaker comments on rates
high     = surprise CPI miss or oil supply shock
critical = war escalation, bank failure, major cyber/infrastructure outage
```

## Mobile Notification Rule

Do not notify the operator for every World Monitor alert.

Preferred flow:

```text
World Monitor finds high-severity event
  ↓
Jax researches it
  ↓
Jax evidence passes
  ↓
operator receives notification
```

Notifications should be evidence-backed, not raw-news spam.

## Feedback Loop

Every approval/rejection decision should be saved and linked back to the original World Monitor event.

Capture:

```text
world_monitor_event_id
jax_research_id
candidate_id, if created
operator_decision
operator_reason
later_outcome
```

Example rejection reasons:

```text
already priced in
weak source quality
market closed
late reaction
confounder active
spread too wide
no historical edge
```

This allows Jax to learn which World Monitor event types are useful and which are noise.

## Audit Trail

Keep a complete event-to-decision audit trail.

Minimum fields:

```text
world_monitor_event_id
adapter_received_at
adapter_sent_to_jax_at
jax_research_started_at
sources_used
source_tiers
model_provider
model_name
evidence_bundle_id
candidate_created
operator_decision
outcome
```

This is needed so every suggestion can be explained later.

## Rate Limits and Spend Limits

The adapter should enforce hard limits.

Suggested defaults:

```text
max 20 events/hour into Jax
max 5 high-priority alerts/hour
max daily AI spend cap
max tokens per event
ignore duplicate event within 30 minutes
```

This protects Jax from becoming a token woodchipper.

## Final Responsibility Split

```text
World Monitor = eyes
Jax = brain
DeepSeek/Ollama = analyst assistant
Human operator = decision authority
Broker = execution only after approval
```

World Monitor can say:

```text
This looks important.
```

Jax decides:

```text
This is worth researching.
```

After evidence checks, Jax may say:

```text
This may be worth a paper-trade candidate.
```

Only the operator approves or rejects.
