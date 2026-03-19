# Phase 4 — Operator Pages

## Objective
Finish the frontend so Jax becomes an actual usable operator workstation.

## Current state
`frontend/src/app/App.tsx` still routes only:
- dashboard
- trading
- system
- placeholders

`AppShell.tsx` nav also lacks the core ops pages.

## Required pages
- `/research`
- `/analysis`
- `/testing`
- `/approvals`
- `/assistant`

## Required page functions
### Research
- launch backtests
- inspect runs
- manage strategy instances
- compare run outputs
- launch re-analysis / RAG research queries

### Analysis
- candidate/trade/run timeline
- blockers
- AI/evidence
- strategy attribution
- dataset/provenance details

### Testing
- gate dashboard
- test trigger buttons
- proof artifacts
- failed gate details

### Approvals
- queue view
- candidate detail
- approve/reject/snooze/reanalyze

### Assistant
- ask scenario questions
- inspect trade/blocker explanations
- compare similar runs
- retrieve research evidence

## Required files
- new page components
- new hooks
- `frontend/src/app/App.tsx` route wiring
- `frontend/src/components/layout/AppShell.tsx` nav wiring
