# Jax Analysis Intelligence Layer

## Purpose

This folder turns Jax from a headline-aware system into a structured analysis engine.

It adds two expert analysis layers:

```text
Technical Analysis Engine
Fundamental Analysis Engine
```

Then combines them through:

```text
multi-analyst review
scoring
risk veto
memory feedback
backtesting
human approval
```

## Core idea

Jax should not ask:

```text
Is this chart bullish?
```

Jax should ask:

```text
What is the event?
What should the market do if the event matters?
Did the chart confirm that?
Is the move tradable?
What else could be driving the move?
Is this already priced in?
Can risk be defined?
Should we trade, watch, or walk away?
```

## Relationship to other plans

This folder sits after:

```text
Docs/plans/world-monitor-jax-awareness/
Docs/plans/macro-reaction-engine/
```

Target flow:

```text
World Monitor / Macro Calendar
  ↓
Research trigger
  ↓
Macro Reaction Engine
  ↓
Technical Analysis Engine
  ↓
Fundamental Analysis Engine
  ↓
Multi-Analyst Review
  ↓
Risk Manager
  ↓
Evidence Bundle
  ↓
Paper Candidate or No Trade
```

## Non-negotiable rules

```text
No automatic live trading
No candidate without technical and fundamental evidence
No candidate without risk approval
No candidate from a headline alone
No candidate when another event explains the move
No candidate when chart confirmation is missing
No candidate when priced-in verdict is priced_in or unclear
No single stocks, options, inverse ETFs, leveraged ETFs, crypto, forex, or futures in phase 1
```

## Folder order

```text
01_TECHNICAL_ANALYSIS_ENGINE.md
02_FUNDAMENTAL_ANALYSIS_ENGINE.md
03_EVENT_PLAYBOOK_LIBRARY.md
04_ANALYST_SCORING_MODEL.md
05_MULTI_ANALYST_REVIEW_FLOW.md
06_ANALYST_MEMORY_AND_FEEDBACK.md
07_TA_FA_BACKTESTING_UAT.md
08_CODEX_MASTER_PROMPT.md
09_PHASED_CODEX_TICKETS.md
10_ANALYSIS_OUTPUT_CONTRACTS.md
11_FOLDER_PLACEMENT.md
```

## Desired Jax output

A good Jax result should look like this:

```text
Event:
US CPI hotter than expected.

Fundamental read:
Hawkish rates. Negative for QQQ and TLT. Surprise is large enough to matter.

Technical read:
QQQ broke below the pre-release range, failed VWAP reclaim, volume expanded, and TLT also sold off.

Confounder read:
No major same-time conflicting event detected.

Risk read:
Stop can be placed above failed VWAP reclaim. Reward:risk is acceptable.

Decision:
Paper candidate allowed. Awaiting human approval.
```
