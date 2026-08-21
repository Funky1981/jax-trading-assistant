# Phase 01 Reference Notes — Contracts, Provenance, Replay

## Fincept concepts worth extracting
- `docs/ARCHITECTURE.md`: bounded contexts and explicit dependency direction.
- `docs/ALPHA_ARENA.md`: persisted ticks, prompts, raw responses, parsed decisions, risk verdicts, orders, fills and append-only events.
- `src/storage/repositories/`: repository separation and typed persistence concepts.
- `src/core/result/Result<T>`: explicit error/result contract pattern.

## Jax-specific interpretation
Do not replace Jax's existing modular monolith, artifact trust gates or Postgres model. The useful idea is **reconstructability**: every future research/recommendation artifact should be attributable to exact inputs and versions.

## Avoid
- designing a generic enterprise event-sourcing platform;
- duplicating existing Jax run/history/artifact models;
- introducing a new event bus merely because Fincept has DataHub.
