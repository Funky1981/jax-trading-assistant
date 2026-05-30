# 04 — Integration Contracts

Planned contracts:

```text
POST /research/backfill/run
GET  /research/backfill/runs/{run_id}
GET  /api/v1/system/providers
GET  /api/v1/system/market-data-status
GET  /api/v1/candidates/{id}/approval-summary
POST /api/v1/candidates/{id}/approval
POST /api/v1/notifications/outcomes
```

Jax should send candidate_created webhook with candidate id only. n8n fetches authoritative summary from Jax.
