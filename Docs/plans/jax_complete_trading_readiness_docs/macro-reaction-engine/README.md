# Jax Macro Reaction Engine — Codex-Ready Plan Pack

## Purpose

This pack adds the missing layer between:

```text
World Monitor detects an event
↓
Jax receives a research trigger
```

and:

```text
Jax checks charts, macro data, ETF reaction, priced-in risk, guardrails
↓
Jax creates or rejects a candidate trade
```

This does **not** replace the existing `Docs/plans/world-monitor-jax-awareness/` plan. That plan handles the external awareness boundary. This pack starts after the event is safely inside Jax.

## What this pack builds

```text
Macro calendar ingest
Event classification
ETF mapping confirmation
Chart/candle reaction engine
Priced-in and confounder checks
Evidence bundle generation
Candidate trade creation
Manual approval workflow
Backtesting/UAT validation
```

## Non-negotiable rules

```text
No automatic live trading
No World Monitor direct trade creation
No trade candidate without chart confirmation
No trade candidate without evidence bundle
No trade candidate when event is unclear or already priced in
No candidate outside ETF allowlist
No inverse/leveraged/options/single-stock trades in phase 1
```

## Recommended implementation order

```text
01. Macro event model and calendar data
02. Chart reaction engine
03. ETF mapping and scenario playbooks
04. Priced-in/confounder engine
05. Evidence bundle builder
06. Candidate trade generation
07. UI/API integration
08. Backtesting and UAT
09. Codex master prompt
```

Detailed first-slice plan:

```text
12_PHASE_1_IMPLEMENTATION_PLAN.md
```

## Target runtime flow

```text
World Monitor / calendar event
  ↓
research_trigger
  ↓
macro_event
  ↓
affected ETF set
  ↓
candle reaction snapshot
  ↓
scenario playbook
  ↓
priced-in/confounder checks
  ↓
evidence bundle
  ↓
candidate_trade or rejected_candidate
  ↓
human approval
```
