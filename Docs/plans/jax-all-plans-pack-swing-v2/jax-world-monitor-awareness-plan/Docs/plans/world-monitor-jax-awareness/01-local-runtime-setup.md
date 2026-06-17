# 01 — Local Runtime Setup

## Goal

Run World Monitor separately for personal use and connect it to Jax through a small local adapter.

## Recommended Local Topology

```text
Local machine or home server
├── world-monitor
├── jax-trading-assistant
├── world-monitor-jax-adapter
├── Ollama or LM Studio
└── Postgres
```

## Cost Target

Initial target: **£0/month**.

Avoid paid hosting, cloud LLMs, paid market APIs, and managed Redis until the workflow proves useful.

## Runtime Modes

| Component | Mode | Notes |
|---|---|---|
| World Monitor | Local desktop/web app | Personal awareness dashboard |
| Jax | Research/paper only | No live trading |
| Adapter | Local service | Converts events only |
| LLM | Local Ollama/LM Studio | Optional, not required for first version |
| Database | Jax Postgres | Source of truth for research triggers |

## First Implementation

Start with the simplest approach:

1. Run World Monitor locally.
2. Run Jax locally.
3. Create a local adapter service.
4. Manually or automatically pull World Monitor events/headlines.
5. Convert selected events into Jax research-trigger payloads.
6. POST them to a Jax research ingestion endpoint.

## Avoid For Phase 1

- Hosting World Monitor publicly.
- Connecting World Monitor directly to a broker.
- Cloud LLM providers.
- Real-money trading.
- Complex message brokers.
- Auto-approval.

## Later Options

Only after the local flow works:

- Run on home server.
- Add scheduled polling.
- Add Telegram/phone alerting.
- Add Redis/NATS/RabbitMQ if event volume justifies it.
- Add cloud deployment only if remote access is genuinely needed.
