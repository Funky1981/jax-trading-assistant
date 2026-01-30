# Debugging Guide

This guide covers how to debug JAX Trading Assistant when things go wrong.

## Quick Diagnosis

### 1. Check Service Status
```powershell
# See all containers and their status
docker compose ps -a

# Check logs for all services
docker compose logs

# Follow logs in real-time
docker compose logs -f
```

### 2. Check Individual Service Logs
```powershell
docker compose logs jax-api
docker compose logs jax-memory
docker compose logs hindsight
docker compose logs postgres
```

---

## Common Issues

### Docker Build Fails

**Symptom:** `go mod download` fails with "no such file or directory"

**Solution:** The Dockerfile needs access to the `libs/` folder. This should be fixed now, but if it happens:
```powershell
# Force rebuild without cache
docker compose build --no-cache
```

---

### Services Won't Start

**Symptom:** `start.ps1` exits with code 1

**Debug Steps:**
```powershell
# 1. Check what's actually running
docker compose ps -a

# 2. Check for build errors
docker compose build 2>&1 | Select-String -Pattern "ERROR|error|failed"

# 3. Check container logs
docker compose logs --tail=50

# 4. Start services one at a time
docker compose up postgres -d
docker compose up hindsight -d
docker compose up jax-memory -d
docker compose up jax-api -d
```

---

### Database Connection Issues

**Symptom:** "connection refused" or "ECONNREFUSED"

**Debug Steps:**
```powershell
# 1. Check if postgres is running
docker compose ps postgres

# 2. Check postgres logs
docker compose logs postgres

# 3. Test connection from host
psql -h localhost -U jax -d jax

# 4. Test connection from inside Docker
docker compose exec postgres psql -U jax -d jax -c "SELECT 1"
```

---

### Health Check Failures

**Symptom:** `start.ps1` says "Backend services may not be fully ready"

**Debug Steps:**
```powershell
# 1. Manually test health endpoints
curl http://localhost:8081/health
curl http://localhost:8090/health

# 2. Check if ports are exposed
docker compose ps

# 3. Check for port conflicts
netstat -an | Select-String "8081|8090|5432"
```

---

### IB Gateway Connection Issues

**Symptom:** "connection refused" to IB Gateway

**Checklist:**
1. ✅ IB Gateway (not TWS) is running
2. ✅ Logged in with paper trading credentials
3. ✅ API enabled: Configure → Settings → API → Enable ActiveX and Socket Clients
4. ✅ Port is correct (7497 for paper, 7496 for live)
5. ✅ "Allow connections from localhost only" is checked (or add your IP)

**Test IB Gateway manually:**
```powershell
# Check if port is listening
Test-NetConnection -ComputerName localhost -Port 7497
```

---

## Running Services Locally (Without Docker)

For debugging, you might want to run services directly:

### Prerequisites
```powershell
# Start only postgres in Docker
docker compose up postgres -d
```

### Run jax-api locally
```powershell
cd services/jax-api
$env:DATABASE_URL="postgresql://jax:jax@localhost:5432/jax"
go run ./cmd/jax-api -providers ../../config/providers.json
```

### Run jax-memory locally
```powershell
cd services/jax-memory
$env:HINDSIGHT_URL="http://localhost:8888"
go run ./cmd/jax-memory
```

### Run frontend locally
```powershell
cd frontend
npm run dev
```

---

## VS Code Debugging

### Go Services

Create `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug jax-api",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/services/jax-api/cmd/jax-api",
      "args": ["-providers", "${workspaceFolder}/config/providers.json"],
      "env": {
        "DATABASE_URL": "postgresql://jax:jax@localhost:5432/jax"
      }
    },
    {
      "name": "Debug jax-memory",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/services/jax-memory/cmd/jax-memory",
      "env": {
        "HINDSIGHT_URL": "http://localhost:8888"
      }
    }
  ]
}
```

### Frontend (React)

Use browser DevTools or VS Code's built-in JavaScript debugger.

---

## Logs Location

| Service | Docker Logs | Local Logs |
|---------|-------------|------------|
| jax-api | `docker compose logs jax-api` | stdout |
| jax-memory | `docker compose logs jax-memory` | stdout |
| hindsight | `docker compose logs hindsight` | stdout |
| postgres | `docker compose logs postgres` | `/var/lib/postgresql/data/log/` |
| frontend | Terminal running `npm run dev` | Browser console |

---

## Reset Everything

When all else fails:
```powershell
# Stop and remove everything
docker compose down -v

# Remove all images (optional, forces full rebuild)
docker compose down --rmi all

# Start fresh
docker compose up -d
```

---

## Getting Help

1. Check the logs first (90% of issues are visible in logs)
2. Check GitHub Issues: https://github.com/Funky1981/jax-trading-assistant/issues
3. Include logs when asking for help
