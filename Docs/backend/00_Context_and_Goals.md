# 00 — Context & Goals

## What we are building

**Jax** is an AI trading assistant built from four core pieces:

- **Dexter**: market/event ingestion and signal generation (the “senses”)
- **Agent0**: planner/executor agent (the “brain”)
- **go-UTCP**: tool layer for calling capabilities (the “muscles”)
- **Hindsight (Vectorize)**: long-term memory + reflection (the “hippocampus”)

## What changes in this revision

We are **adding Hindsight** into the architecture as a first-class subsystem, and enforcing **TDD** so:

1) Every service boundary has executable tests.
2) Every meaningful outcome can be **logged and optionally retained to memory**.
3) Agent plans can **recall** relevant prior context before acting.
4) Scheduled **reflection** produces durable insights (“beliefs”) for future decisions.

## High-level requirements

- Run locally via Docker Compose
- Deterministic tests (no flaky web calls)
- Clear interfaces between:
  - data ingestion (Dexter)
  - planning (Agent0)
  - tools (go-UTCP)
  - memory (Hindsight)

## Non-goals (for now)

- Not building a fully automated trading bot that places orders without human review
- Not attempting to “predict markets” using LLM magic
- Not storing secrets or broker credentials in memory
