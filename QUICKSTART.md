# Quick Start Guide

## Prerequisites

- **Docker Desktop** (for backend services)
- **Node.js 18+** (for frontend)
- **Go 1.22+** (optional, for local backend development)

## 🚀 Start the Full Stack

### One-Command Start (Recommended)

```powershell
.\start.ps1
```

This automatically:
1. ✅ Starts backend services (Docker)
2. ✅ Waits for services to be ready
3. ✅ Installs frontend dependencies (first time)
4. ✅ Starts frontend dev server
5. ✅ Opens http://localhost:5173 in your browser

**To stop everything:**
```powershell
# Press Ctrl+C to stop frontend
# Then run:
.\stop.ps1
```

### Manual Start (Alternative)

#### 1. Start Backend Services

```powershell
# Start core backend (Hindsight memory + JAX services)
docker compose up -d

# Wait for services to be ready (~30 seconds)
docker compose logs -f
```

**Services started:**
- Hindsight Memory: http://localhost:8888
- JAX Memory API: http://localhost:8090
- JAX API: http://localhost:8081

### 2. Start Frontend

```powershell
cd frontend
npm install  # First time only
npm run dev
```

**Frontend:** http://localhost:5173

### 3. Open the Dashboard

Navigate to: **http://localhost:5173**

You should see:
- ✅ **Health Status Widget** (top-left) - Shows backend service health
- 📊 **Metrics Dashboard** (top-right) - Real-time metrics
- 🔍 **Memory Browser** (middle) - Search memory banks
- 📈 **Trading Dashboard** (bottom) - Market data & positions

## 🔍 Verify Everything is Working

### Check Backend Health

```powershell
# JAX API health
curl http://localhost:8081/health

# Memory Service health
curl http://localhost:8090/health

# Hindsight API
curl http://localhost:8888/
```

### Check Frontend

Open browser DevTools (F12) and check:
- Console: Should show no errors
- Network tab: API calls to localhost:8081 and localhost:8090
- Health widgets should show green "Healthy" status

## 🐛 Debugging

### Backend Logs

```powershell
# View all service logs
docker compose logs -f

# View specific service
docker compose logs -f jax-api
docker compose logs -f jax-memory
docker compose logs -f hindsight
```

### Frontend Debug Mode

```powershell
cd frontend
npm run dev  # Dev server with hot reload
```

- Open browser DevTools (F12)
- React DevTools extension recommended
- Check Network tab for API calls
- Check Console for errors

### Common Issues

**Backend services not starting:**
```powershell
# Stop all containers
docker compose down

# Rebuild from scratch
docker compose build --no-cache
docker compose up -d
```

**Frontend can't connect to backend:**
- Check `.env` file in frontend folder exists:
  ```env
  VITE_API_URL=http://localhost:8081
  VITE_MEMORY_API_URL=http://localhost:8090
  ```
- Restart frontend dev server

**Port conflicts:**
```powershell
# Check if ports are in use
netstat -ano | findstr "8081"
netstat -ano | findstr "8090"
netstat -ano | findstr "8888"
netstat -ano | findstr "5173"
```

## 🎯 Run Orchestration (Optional)

Test the full intelligence pipeline:

```powershell
# Run orchestration for AAPL
docker compose --profile jobs run jax-orchestrator -symbol AAPL

# View results in frontend Memory Browser
# Search for: AAPL
```

## 📊 Test Data Ingestion (Optional)

```powershell
# Create sample data file
mkdir data
# Place dexter.json in ./data/ folder

# Run ingestion
docker compose --profile jobs run jax-ingest
```

## 🛑 Shutdown

```powershell
# Stop all services
docker compose down

# Stop and remove volumes (clean slate)
docker compose down -v
```

## 📖 Advanced Usage

### Run Backend Locally (Without Docker)

```powershell
# Terminal 1: Start Hindsight (Python)
cd services/hindsight
python -m venv venv
.\venv\Scripts\activate
pip install -e ".[api]"
python -m hindsight.api

# Terminal 2: Start JAX Memory (Go)
go run services/jax-memory/cmd/jax-memory/main.go

# Terminal 3: Start JAX API (Go)
go run services/jax-api/cmd/jax-api/main.go `
  -providers config/providers.json `
  -config config/jax-core.json `
  -strategies config/strategies
```

### VSCode Launch Configurations

Press F5 in VSCode to debug:
- **Frontend**: Launches Chrome with React DevTools
- **JAX API**: Attaches debugger to Go service
- **JAX Orchestrator**: Debug orchestration runs

### Environment Variables

Create `.env` in project root:

```env
# Backend
HINDSIGHT_API_LLM_PROVIDER=openai
HINDSIGHT_API_LLM_API_KEY=your-key-here
JAX_SYMBOL=AAPL

# Frontend (frontend/.env)
VITE_API_URL=http://localhost:8081
VITE_MEMORY_API_URL=http://localhost:8090
```

## 📚 Next Steps

1. **Explore the Dashboard**: Click around, search memories, view metrics
2. **Run Orchestration**: Test the intelligence pipeline with different symbols
3. **Check Documentation**:
   - [Frontend Architecture](Docs/frontend/README.md)
   - [Backend Overview](Docs/backend/00_Context_and_Goals.md)
   - [Phase 5 Integration](Docs/backend/14_Phase_5_Frontend_Integration.md)
4. **Run Tests**:
   ```powershell
   # Frontend tests
   cd frontend
   npm test

   # Go tests
   go test ./...
   ```

## 🎉 You're All Set!

The trading intelligence dashboard is now running with:
- ✅ Real-time health monitoring
- ✅ Memory search and recall
- ✅ Strategy signals
- ✅ Orchestration pipeline
- ✅ Full observability

Happy trading! 🚀
