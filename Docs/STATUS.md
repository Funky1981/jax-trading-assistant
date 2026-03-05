# Project Status (Condensed)

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge`, `services/agent0-service`, and `services/hindsight`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), `hindsight` (8888), frontend dev server (5173).
- **Recent completions**: artifact validation now requires Gate2+Gate3 evidence, auth login now uses persisted users with lockout policy, provider gaps and deterministic test shims have been tightened.

## What Works (High Confidence)

- **Trader frontend API**: health, risk, testing, artifact, run-history, orchestration-proxy, and trades endpoints (`cmd/trader/frontend_api.go`, `cmd/trader/codex_api.go`).
- **Research runtime**: orchestration, backtest/research pathways, and memory tooling endpoints in-process (`cmd/research`).
- **Artifact workflow**: validation and promotion are tied to persisted trust-gate evidence and approval state transitions.
- **Memory backend integration**: Hindsight-backed memory flows remain active through research runtime wiring.

## Partial / Incomplete

- **Artifact API tests**: additional targeted coverage is still needed for filtered listing and failure-path behavior.
- **Market ingestion confidence**: adapter support is improved, but ingestion and persistence validation remains a hardening area.
- **Ops polish**: observability/reporting and docs hygiene still have ongoing work.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`
- **Tracked backlog**: `Docs/TODO.md`
- **Local bootstrap/runbook**: `Docs/QUICKSTART.md`
