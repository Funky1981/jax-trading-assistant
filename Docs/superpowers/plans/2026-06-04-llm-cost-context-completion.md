# LLM Cost Context Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the runtime work described by `Docs/plans/llm-cost-context` with staged, tested commits.

**Architecture:** Keep LLM cost/context behavior in `internal/modules/llmcontext` and expose narrow integration points to `internal/modules/chat` and `cmd/trader`. All provider calls must flow through a gateway-aware client or an adapter that logs usage and fails closed before provider invocation.

**Tech Stack:** Go 1.24, pgx/pgxpool, PostgreSQL migrations, `go test`.

---

### Task 1: Commit Foundation Slice

**Files:**
- Create: `internal/modules/llmcontext/*.go`
- Create: `internal/modules/llmcontext/*_test.go`
- Create: `db/postgres/migrations/000026_llm_cost_context.up.sql`
- Create: `db/postgres/migrations/000026_llm_cost_context.down.sql`
- Create: `db/postgres/migrations/llm_cost_schema_test.go`

- [ ] **Step 1: Verify existing foundation tests**

Run: `go test ./internal/modules/llmcontext ./db/postgres/migrations`

Expected: package tests pass.

- [ ] **Step 2: Commit foundation**

Run:

```powershell
git add internal/modules/llmcontext db/postgres/migrations/000026_llm_cost_context.up.sql db/postgres/migrations/000026_llm_cost_context.down.sql db/postgres/migrations/llm_cost_schema_test.go
git commit -m "feat: add llm cost context foundation"
```

Expected: commit contains only foundation package and migration files.

### Task 2: Persist Usage Logs and Cost Rollups

**Files:**
- Create: `internal/modules/llmcontext/postgres_usage_logger.go`
- Create: `internal/modules/llmcontext/postgres_usage_logger_test.go`

- [ ] **Step 1: Write failing tests**

Add tests using `go-sqlmock` for `PostgresUsageLogger.RecordPlanned`, `PostgresUsageLogger.RecordActual`, and `PostgresUsageLogger.UpsertRollup`.

- [ ] **Step 2: Run red tests**

Run: `go test ./internal/modules/llmcontext -run Postgres`

Expected: fails because `NewPostgresUsageLogger` is undefined.

- [ ] **Step 3: Implement logger**

Implement a logger that accepts an interface with `ExecContext`, writes to `llm_usage_logs`, updates actual usage by `correlation_id`, and upserts `llm_cost_rollups`.

- [ ] **Step 4: Verify and commit**

Run:

```powershell
gofmt -w internal/modules/llmcontext
go test ./internal/modules/llmcontext ./db/postgres/migrations
git add internal/modules/llmcontext/postgres_usage_logger.go internal/modules/llmcontext/postgres_usage_logger_test.go
git commit -m "feat: persist llm usage and cost rollups"
```

Expected: tests pass and commit contains only persistence work.

### Task 3: Add Headroom Trial Adapter

**Files:**
- Create: `internal/modules/llmcontext/headroom.go`
- Create: `internal/modules/llmcontext/headroom_test.go`

- [ ] **Step 1: Write failing tests**

Add tests proving Headroom is disabled by default, rejects Zone A/Zone B fields, allows Zone C only when enabled, records original/compressed tokens, latency, saving percentage, and disables live approval compression.

- [ ] **Step 2: Run red tests**

Run: `go test ./internal/modules/llmcontext -run Headroom`

Expected: fails because Headroom types are undefined.

- [ ] **Step 3: Implement adapter**

Implement `HeadroomTrial` with a feature flag config, an injected HTTP client for tests, and output `CompressionTrialResult`.

- [ ] **Step 4: Verify and commit**

Run:

```powershell
gofmt -w internal/modules/llmcontext
go test ./internal/modules/llmcontext
git add internal/modules/llmcontext/headroom.go internal/modules/llmcontext/headroom_test.go
git commit -m "feat: add optional headroom compression trial"
```

Expected: llmcontext tests pass and commit contains Headroom adapter only.

### Task 4: Wire Chat Through Gateway-Aware Boundary

**Files:**
- Modify: `internal/modules/chat/llm.go`
- Modify: `internal/modules/chat/llm_test.go`
- Modify: `cmd/trader/chat_handlers.go`
- Modify: `cmd/trader/ai_overview_handlers.go`
- Add tests where needed under existing package tests.

- [ ] **Step 1: Write failing tests**

Add tests proving `NewOpenAIChatClientFromEnv` prefers `AI_GATEWAY_BASE_URL`/`AI_GATEWAY_API_KEY`, refuses direct `https://api.openai.com` unless explicitly enabled, preserves tool-call decoding, and AI overview exposes LLM cost metrics.

- [ ] **Step 2: Run red tests**

Run: `go test ./internal/modules/chat ./cmd/trader -run "OpenAI|Chat|AIOverview|LLM"`

Expected: fails on missing gateway policy behavior and metrics.

- [ ] **Step 3: Implement wiring**

Update chat client env handling to prefer LiteLLM gateway keys, block direct paid provider defaults unless `AI_ALLOW_DIRECT_PROVIDER=true`, and expose cost metrics in AI overview by reading `llm_usage_logs`.

- [ ] **Step 4: Verify and commit**

Run:

```powershell
gofmt -w internal/modules/chat cmd/trader
go test ./internal/modules/chat ./cmd/trader ./internal/modules/llmcontext
git add internal/modules/chat cmd/trader
git commit -m "feat: route chat ai through llm gateway policy"
```

Expected: tests pass and commit contains only chat/API integration.

### Task 5: Final Full Verification

**Files:**
- No new files expected.

- [ ] **Step 1: Run full Go test suite**

Run: `go test ./...`

Expected: all Go packages pass.

- [ ] **Step 2: Report any skipped script**

If `scripts/go-verify.ps1` still fails because unrelated repository files need gofmt, report that precisely and do not reformat unrelated files.
