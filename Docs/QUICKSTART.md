# Quick Start

## Prerequisites

- Docker Desktop
- Node.js 20+
- Go 1.22+ (optional for local Go runs)

## Start Stack (Compose)

```powershell
docker compose up -d
```

Core runtime services:
- `jax-trader` (`http://localhost:8081/health`, runtime port `8100`)
- `jax-research` (`http://localhost:8091/health`)
- `hindsight` (`http://localhost:8888`)
- `ib-bridge` (`http://localhost:8092/health`)
- `agent0-service` (`http://localhost:8093/health`)

## Start Frontend

```powershell
cd frontend
npm install
npm run dev
```

Frontend URL: `http://localhost:5173`

## Health Checks

```powershell
curl http://localhost:8081/health
curl http://localhost:8091/health
curl http://localhost:8092/health
curl http://localhost:8093/health
curl http://localhost:8888/
```

## Authentication (Optional)

Enable JWT mode by setting `JWT_SECRET` for `jax-trader`.

Optional first-user bootstrap:
- `AUTH_BOOTSTRAP_USERNAME`
- `AUTH_BOOTSTRAP_PASSWORD`
- `AUTH_BOOTSTRAP_ROLE` (`admin` or `user`, default `admin`)

## Common Logs

```powershell
docker compose logs -f jax-trader
docker compose logs -f jax-research
docker compose logs -f ib-bridge
docker compose logs -f agent0-service
docker compose logs -f hindsight
```

## Stop Stack

```powershell
docker compose down
```

Full cleanup (removes volumes):

```powershell
docker compose down -v
```
