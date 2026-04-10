# Debug Command Templates

Use this set in order, stopping when root cause is found.

## Status and Logs

- `docker compose ps -a`
- `docker compose logs --tail=100`
- `docker compose logs -f jax-trader`
- `docker compose logs -f jax-research`

## Health Checks

- `curl http://localhost:8081/health`
- `curl http://localhost:8091/health`
- `curl http://localhost:8092/health`

## Port Checks (Windows)

- `netstat -an | Select-String "8081|8091|8092|5173|5433"`

## Local Script Entry

- `.\start.ps1`
- `.\stop.ps1`

## If Environment Is Corrupted

- `docker compose down -v`
- `docker compose up -d`
