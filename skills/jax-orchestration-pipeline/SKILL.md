---
name: jax-orchestration-pipeline
description: End-to-end orchestration guidance for flows spanning `jax-api`, `jax-orchestrator`, `jax-signal-generator`, memory/UTCP provider wiring, and service-to-service contracts. Use when debugging orchestration issues, modifying orchestration behavior, or tracing request/response paths across services.
---

# Jax Orchestration Pipeline

Trace orchestration behavior quickly without guessing across service boundaries.

## Workflow

1. Start from entrypoint:
   - API path in `services/jax-api/...handlers_orchestration_v1.go`
   - signal path in `services/jax-signal-generator/internal/orchestrator/client.go`
2. Follow downstream clients in orchestrator composition:
   - memory, Agent0, Dexter clients and provider config
3. Validate request/response contracts at each hop before editing internals.
4. Preserve payload shape unless coordinated with all consumers.
5. Run focused tests and smoke checks after changes.

## Guardrails

- Avoid broad refactors across API + orchestrator + signal-generator in one pass.
- Keep orchestration HTTP compatibility until migration phases explicitly remove it.
- Align config assumptions with `config/providers.json`.

## Quick Checks

- health endpoints for involved services
- orchestration integration tests where present
- targeted API handler and client package tests

Use `references/pipeline-map.md` for concrete file-level routing.
