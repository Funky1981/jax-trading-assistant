# Test Plan

## Objective

Validate the current production path end-to-end for:
- runtime health
- API contract availability
- backend regression stability
- frontend correctness
- replay/golden comparison utility behavior
- memory system (Postgres/pgvector retain, recall, reflect)

## Entry Criteria

- `docker compose up -d` completed successfully
- `jax-trader` and `jax-research` are healthy
- Postgres migrations are applied (including `000020_memory_items` and `000021_memory_items_constraints`)
- Postgres image is `pgvector/pgvector:pg16` (required for `memory_items`)
- frontend dependencies are installed (`frontend/node_modules`)

## Docker Image Requirements

The `postgres` service must use the pgvector image:

```yaml
# docker-compose.yml
postgres:
  image: pgvector/pgvector:pg16
```

Apply migrations:

```powershell
docker run --rm --network host `
  -v "${PWD}/db/postgres/migrations:/migrations" `
  migrate/migrate `
  -path=/migrations -database "postgresql://jax:jax@localhost:5433/jax?sslmode=disable" up
```

## Automated Execution

### Quick Gate (recommended for every change)

```powershell
.\scripts\test-platform.ps1 -Mode quick
```

### Full Gate (pre-release)

```powershell
.\scripts\test-platform.ps1 -Mode full
```

### Full Gate + Visual E2E Report

```powershell
.\scripts\test-platform.ps1 -Mode full -OpenVisualReport
```

### Memory System Integration Tests

Requires Postgres running with pgvector. No OpenAI key required (uses `noopEmbedder`).

```powershell
$env:TEST_DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax?sslmode=disable"
go test -tags integration -count=1 -v ./libs/pgmemory/...
```

Expected: 16 tests pass (12 integration + 4 unit).

### Memory Smoke Script (HTTP end-to-end)

Requires Docker and `jax-research` running. Full Retain/Recall/Reflect cycle requires `OPENAI_API_KEY`.

```powershell
# Without OpenAI key (migration + banks + list-items checks only):
.\scripts\smoke-memory-migration.ps1 -SkipEmbed

# With OpenAI key (full retain/recall/reflect/search cycle):
$env:OPENAI_API_KEY = "<key>"
.\scripts\smoke-memory-migration.ps1
```

## Coverage Matrix

1. Health checks:
   - `8081` trader API
   - `8091` research
   - `8092` ib-bridge
   - `8093` agent0-service
2. API smoke:
   - `/api/v1/signals`
   - `/api/v1/artifacts`
   - `/api/v1/testing/status`
   - `/api/v1/runs`
   - `/api/v1/ai-decisions`
3. Memory API:
   - `GET /v1/memory/banks` — returns `["research","trades","signals","reflections"]`
   - `POST /tools` (memory.retain) — stores item + embedding, returns `id`
   - `POST /tools` (memory.recall) — structured query by symbol/tags/date
   - `POST /tools` (memory.recall with `q`) — vector cosine nearest-neighbour
   - `POST /tools` (memory.reflect) — ephemeral synthesis item
   - `GET /v1/memory/banks/{bank}/items` — paginated list
   - `GET /v1/memory/banks/{bank}/items/{id}` — fetch by ID
   - `GET /v1/memory/search?q=...&bank=...` — search shortcut
4. Backend quality:
   - `scripts/go-verify.ps1` quick/full modes
   - `scripts/golden-check.ps1 -Mode verify`
5. Frontend quality:
   - `npm run lint`
   - `npm run typecheck`
   - `npm run test`
   - `npx playwright test --reporter=html` (full mode)

## Evidence Output

Each automated run writes:
- `Docs/runs/test_run_<timestamp>.md`
- `Docs/runs/test_run_<timestamp>.json`

Playwright full runs generate:
- `frontend/playwright-report/index.html`

## Manual Spot Checks (Release Candidate)

### Core Platform

1. Login/auth status path works (`/auth/status`, `/auth/login` if enabled).
2. Strategy/artifact list pages load with no frontend console errors.
3. Run detail and AI decisions pages load and show timeline data.
4. Artifact validation endpoint (`/api/v1/artifacts/{id}/validate`) returns trust-gate evidence.
5. Audit trail queries in `Docs/AUDIT_TRAIL.md` return expected rows.

### Memory Browser (MemoryBrowserPanel)

1. Open the dashboard and navigate to the Memory panel.
2. Confirm the bank selector shows: **research**, **trades**, **signals**, **reflections**.
3. Select **research** — panel loads without errors; empty state or item list shown.
4. Type a search query (e.g. "AAPL MACD") — results update (requires `OPENAI_API_KEY` for vector search).
5. Open browser DevTools console — confirm no React errors or failed network requests.
6. Verify `GET /v1/memory/banks/research/items` returns `{"items": [...]}` envelope.

### Memory Retain/Recall Round-Trip (requires OPENAI_API_KEY)

```powershell
# 1. Retain a test item
$body = @{
  tool  = "memory.retain"
  input = @{
    bank = "research"
    item = @{
      ts      = (Get-Date -Format "o")
      type    = "signal"
      symbol  = "AAPL"
      summary = "Manual test: strong MACD crossover signal"
      tags    = @("manual-test", "aapl")
      data    = @{ confidence = 0.9 }
      source  = @{ system = "manual" }
    }
  }
} | ConvertTo-Json -Depth 10
$r = Invoke-RestMethod "http://localhost:8091/tools" -Method Post -Body $body -ContentType "application/json"
$id = $r.output.id
Write-Host "Retained id=$id"

# 2. Recall by symbol (structured)
$q = @{ tool="memory.recall"; input=@{ bank="research"; query=@{ symbol="AAPL"; limit=5 } } } | ConvertTo-Json -Depth 10
(Invoke-RestMethod "http://localhost:8091/tools" -Method Post -Body $q -ContentType "application/json").output.items

# 3. Recall by text (vector)
$q = @{ tool="memory.recall"; input=@{ bank="research"; query=@{ q="MACD crossover AAPL"; limit=5 } } } | ConvertTo-Json -Depth 10
(Invoke-RestMethod "http://localhost:8091/tools" -Method Post -Body $q -ContentType "application/json").output.items

# 4. Fetch by ID
Invoke-RestMethod "http://localhost:8091/v1/memory/banks/research/items/$id"
```

Expected: each step returns data without 4xx/5xx errors. Vector recall result count > 0.

### DB Verification

```sql
-- Connect: psql postgresql://jax:jax@localhost:5433/jax
SELECT count(*) FROM memory_items;
SELECT id, bank, type, symbol, LEFT(summary, 60) FROM memory_items ORDER BY created_at DESC LIMIT 10;
-- Confirm embedding is populated (not null):
SELECT id, embedding IS NOT NULL AS has_embedding FROM memory_items LIMIT 5;
```

## Staging Soak (Pre-Production)

Run a continuous paper-mode soak for at least one market session:

1. `JAX_RUNTIME_MODE=paper`
2. `JAX_REQUIRE_EXPLICIT_RUNTIME_MODE=true`
3. Run recurring health checks and capture order/error metrics
4. Verify no provenance integrity regressions during the soak window
5. Confirm memory items are being retained during live signal generation (check `memory_items` table row count increases)

Use `Docs/PRODUCTION_READINESS.md` as the promotion checklist after soak completion.
