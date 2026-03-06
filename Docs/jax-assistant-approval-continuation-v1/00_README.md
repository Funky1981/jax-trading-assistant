# Jax Assistant + Approval Continuation Pack v1 (work branch)

This pack continues from the current **work** branch state and is designed for Codex desktop execution.

## What this pack covers
Three separate workstreams in separate folders:
1. `01-always-on-trade-watcher/`
2. `02-human-approval-flow/`
3. `03-frontend-chat-assistant/`

## Repo state this pack is based on
Verified against `work`:
- `cmd/trader/frontend_api.go` already exposes:
  - signals list/detail
  - approve/reject
  - signal analyze
  - recommendations
  - trades
  - orchestrate run access
  - strategies list/detail
  - trading guard
  - risk calc
- `cmd/research/main.go` already hosts orchestration in-process and exposes:
  - `/orchestrate`
  - `/backtest`
  - `/metrics/prometheus`
- `cmd/trader/strategy_instances_loader.go` already loads strategy instances from JSON into DB
- `frontend/src/app/App.tsx` still routes only:
  - `/`
  - `/trading`
  - `/system`
  - placeholder pages
- `frontend/src/components/layout/AppShell.tsx` nav also still only exposes:
  - Dashboard / Trading / System / Portfolio / Blotter / Settings

## Continuation principle
This pack does **not** replace the existing architecture.
It extends the current `cmd/trader` + `cmd/research` direction.

## Non-negotiables
- AI is advisory only
- no chatbot direct order execution
- no bypass of approval/risk/flatten rules
- no fake data in any research/trading truth path
