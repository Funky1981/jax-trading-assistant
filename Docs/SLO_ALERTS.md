# SLO and Alert Policy

This document defines minimum service objectives and alert triggers for trader/research production operation.

## Service SLOs

1. API availability (`jax-trader` and `jax-research`): `>= 99.9%` per 30-day window.
2. API p95 latency (`/api/v1/runs`, `/api/v1/signals`, `/api/v1/ai-decisions`): `< 500ms`.
3. Broker bridge health (`/health` on `8092`): no continuous outage longer than `2 minutes`.
4. Data freshness for market quotes: latest quote age `< 15s` during market hours.
5. Provenance integrity: `0` synthetic truth-path rows in `paper`/`live`.

## Alert Triggers

1. Critical: `jax-trader` health endpoint fails for 3 consecutive probes.
2. Critical: `jax-research` health endpoint fails for 3 consecutive probes.
3. Critical: provenance gate returns `fail` state.
4. High: broker bridge unavailable for more than `120s`.
5. High: order submit failure rate > `5%` over 10-minute window.
6. High: decision-to-trade audit join has missing `run_id`/`flow_id`.
7. Medium: p95 latency breach for 15 minutes.
8. Medium: stale quote age breach for 5 minutes.

## Minimum Dashboards

1. Health panel: all service health endpoints.
2. Trade pipeline panel: signal count, order submit count, fill count, failure count.
3. Audit panel: run count, missing provenance fields, synthetic row count.
4. Performance panel: API p50/p95 latency and error rate by endpoint.

## Operator Response

For any `Critical` or `High` alert, follow `Docs/INCIDENT_RUNBOOK.md`.

