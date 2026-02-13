# Debug Command Templates

Use this set in order, stopping when root cause is found.

## Status and Logs

- `docker compose ps -a`
- `docker compose logs --tail=100`
- `docker compose logs -f jax-api`
- `docker compose logs -f jax-memory`

## Health Checks

- `curl http://localhost:8081/health`
- `curl http://localhost:8090/health`
- `curl http://localhost:8888/`

## Port Checks (Windows)

- `netstat -an | Select-String "8081|8090|8888|5173|5432"`

## Local Script Entry

- `.\start.ps1`
- `.\stop.ps1`

## If Environment Is Corrupted

- `docker compose down -v`
- `docker compose up -d`
