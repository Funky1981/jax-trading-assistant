# 18 — Codex Prompts

## Prompt 1 — Add Token Saving Docs

```text
Add token-saving playbook docs under:
Docs/plans/llm-cost-context/

Create:
11-token-saving-playbook.md
12-safe-compression-policy.md
13-headroom-trial-plan.md
14-ai-eligibility-gate.md
15-event-clustering-and-dedupe.md
16-template-output-policy.md
17-cost-per-candidate-metrics.md

Docs only. Do not change runtime behaviour.
```

## Prompt 2 — Implement AI Eligibility Gate

```text
Implement an AI eligibility gate before any model call.

Requirements:
- no AI call if symbol is not ETF allowlisted
- no AI call if event is duplicate
- no AI call if quote stale/spread too wide
- no AI call if priced_in or unclear
- no AI call if evidence bundle missing
- no paid call if budget unavailable
- log blocked AI calls
- add tests for each block
```

## Prompt 3 — Implement Safe Compression Zones

```text
Implement compression zones.

Zone A: never compress trading truth.
Zone B: deterministic compaction only.
Zone C: safe compression allowed.

Add tests proving stop-loss, take-profit, risk, approval, priced-in verdict, and guardrail status never pass through compression.
```

## Prompt 4 — Headroom Trial

```text
Add optional Headroom trial integration behind a feature flag.

Use only for Zone C context.
Disabled by default for live approval workflows.
Measure token saving, latency, and correctness.
Add UAT evidence comparing with/without Headroom.
```

## Prompt 5 — Event Clustering

```text
Implement event clustering/dedupe before AI calls.

Duplicate provider headlines should map to one canonical event.
AI should be called at most once per canonical event.
Store source list and dedupe reason.
```

## Prompt 6 — Cost Metrics

```text
Add cost-per-candidate reporting.

Track:
cost per news event
cost per rejected event
cost per candidate
cost per approved paper trade
cost per strategy
cost per ETF
Headroom savings
cache hit rate
paid calls avoided
```
