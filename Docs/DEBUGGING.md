# Debugging Guide

## Quick Triage

```powershell
docker compose ps -a
docker compose logs --tail=100
docker compose logs -f jax-trader
docker compose logs -f jax-research
```

## Health Checks

```powershell
curl http://localhost:8081/health
curl http://localhost:8100/health
curl http://localhost:8091/health
curl http://localhost:8092/health
curl http://localhost:8093/health
curl http://localhost:8888/
```

## Common Issues

### Services fail to start

```powershell
docker compose build --no-cache
docker compose up -d postgres
docker compose up -d hindsight ib-bridge agent0-service
docker compose up -d jax-research jax-trader
docker compose ps
```

### Database connection/migration failures

```powershell
docker compose logs postgres --tail=200
docker compose exec postgres psql -U jax -d jax -c "SELECT 1;"
docker compose exec postgres psql -U jax -d jax -c "SELECT version, dirty FROM schema_migrations ORDER BY version;"
```

### IB bridge connectivity issues

Checklist:
1. IB Gateway/TWS is running and API socket access is enabled.
2. Host/port in compose env match your local setup (`IB_GATEWAY_HOST`, `IB_GATEWAY_PORT`).
3. Bridge health passes at `http://localhost:8092/health`.

```powershell
docker compose logs ib-bridge --tail=200
Test-NetConnection -ComputerName localhost -Port 8092
```

## Run Go Runtimes Locally

```powershell
# terminal 1
go run ./cmd/research

# terminal 2
go run ./cmd/trader
```

Recommended local env:
- `DATABASE_URL=postgresql://jax:jax@localhost:5433/jax`
- `JAX_ORCHESTRATOR_URL=http://localhost:8091`
- `IB_BRIDGE_URL=http://localhost:8092`
- `AGENT0_SERVICE_URL=http://localhost:8093`

## Full Reset

```powershell
docker compose down -v
docker compose up -d
```
