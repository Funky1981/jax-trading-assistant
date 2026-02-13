---
name: jax-debug-runbook
description: Operational debugging playbook for jax-trading-assistant services, startup scripts, and local runtime dependencies. Use when services fail to start, health checks fail, orchestration breaks, or Docker/local environment behavior diverges from expected baseline.
---

# Jax Debug Runbook

Debug failures using a consistent triage sequence instead of ad hoc command guessing.

## Triage Sequence

1. Check process/container status.
2. Check service logs for the first concrete error.
3. Validate dependency health (Postgres, Hindsight, bridge services).
4. Validate API health endpoints.
5. Reproduce with the smallest startup path that still fails.

## Runtime Modes

- Prefer `.\start.ps1` and `.\stop.ps1` for full-stack local startup/shutdown.
- Use Docker logs for service-level diagnosis.
- Switch to local single-service execution only when isolating a specific backend issue.

## Guardrails

- Diagnose before restart loops.
- Capture command output relevant to root cause, not entire log streams.
- Keep fixes narrow to the failing layer first, then broaden if unresolved.

Use `references/commands.md` for command templates by symptom.
