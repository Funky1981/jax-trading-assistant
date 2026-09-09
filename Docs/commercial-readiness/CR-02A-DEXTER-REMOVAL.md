# Jax Commercial-Readiness CR-02A — Dexter Complete Removal

Date: 2026-09-09  
Repository: `C:\Projects\Jax\jax-trading-assistant`  
Branch: `capability-reset`  
Starting HEAD: `831537c13e68ce12824c90b202bc8a29e4e77a90`  
Scope: bounded removal of Dexter and Dexter-only Jax seams. CR-02B and CR-02C
were not started.

> Historical current-state note (CR-02B, 2026-09-09): this record describes the
> pre-Agent0-removal state and remains immutable evidence for CR-02A. Agent0 was
> subsequently replaced by the Jax-owned planner and removed from the supported
> runtime; see `CR-02B-AGENT0-REMOVAL.md`.

## Result

**DEXTER IS NOT PART OF THE SUPPORTED JAX RUNTIME.**

This change removes the embedded TypeScript/Bun Dexter application, its Go
client module, its UTCP service/types, its orchestration adapter and research
query path, its active configuration and Compose seam, and its Dexter-only
ingestion projections. Shared Jax research, memory, market-data, risk,
workflow, execution-disabled, Agent0 and provider capabilities remain.

No broker, approval, candidate, ticket, order, trade, fill or portfolio state
was created or mutated. `ALLOW_LIVE_TRADING=false`,
`BROKER_EXECUTION_ALLOWED=false`, live worker disabled and maximum leverage 1x
remain unchanged.

## Pre-removal inventory and classification

The exhaustive pre-removal search found the following classes:

| Class | Findings | Action |
| --- | --- | --- |
| ACTIVE_RUNTIME | `cmd/research`, `internal/modules/orchestration`, `libs/utcp`, `libs/ingest` | Removed Dexter wiring and Dexter-only contracts; retained generic Jax boundaries |
| ACTIVE_TEST | UTCP provider fixtures, memory source golden fixtures, observability service label | Replaced stale Dexter identity with Jax-native test identities |
| ACTIVE_CONFIG | `go.work`, `config/jax-core.json`, `config/providers.json`, `docker-compose.yml`, trader-import guard | Removed Dexter module/provider/endpoint/env seams |
| ACTIVE_DOC | root README, architecture/overview/contributing docs, current roadmap pointers, project TODO | Updated supported-architecture statements |
| ARCHIVE/HISTORICAL | `Docs/archive/**`, historical evidence, the body of ADR-0012, CR-01 audit snapshot | Preserved; ADR and CR-01 now carry clear superseded/historical context |
| SHARED CAPABILITY | Agent0, OpenAI/Anthropic support outside Dexter, Financial Datasets, market data, UTCP memory/risk/backtest/broker surfaces | Retained; no broad provider cleanup was performed |

The `cmd/trader/sentiment_feature_test.go` match was a generic “sentiment
provider disabled” test and was not a Dexter reference.

## Exact tracked files removed

The authorized deletion set is exactly the following 65 tracked paths:

```text
dexter/.gitignore
dexter/README.md
dexter/UPSTREAM.md
dexter/bun.lock
dexter/jest.config.js
dexter/package.json
dexter/src/agent/__tests__/agent.test.ts
dexter/src/agent/agent.ts
dexter/src/agent/answer-generator.ts
dexter/src/agent/index.ts
dexter/src/agent/prompts.ts
dexter/src/agent/schemas.ts
dexter/src/agent/task-executor.ts
dexter/src/agent/task-planner.ts
dexter/src/cli.tsx
dexter/src/cli/types.ts
dexter/src/components/AnswerBox.tsx
dexter/src/components/CompletedTurnView.tsx
dexter/src/components/CurrentTurnView.tsx
dexter/src/components/DebugMessages.tsx
dexter/src/components/Input.tsx
dexter/src/components/Intro.tsx
dexter/src/components/ModelSelector.tsx
dexter/src/components/QueueDisplay.tsx
dexter/src/components/Spinner.tsx
dexter/src/components/StatusMessage.tsx
dexter/src/components/TaskList.tsx
dexter/src/components/TaskProgress.tsx
dexter/src/components/index.ts
dexter/src/hooks/useAgentExecution.ts
dexter/src/hooks/useApiKey.ts
dexter/src/hooks/useQueryQueue.ts
dexter/src/index.tsx
dexter/src/model/llm.ts
dexter/src/theme.ts
dexter/src/tools-server.ts
dexter/src/tools/finance/api.ts
dexter/src/tools/finance/constants.ts
dexter/src/tools/finance/crypto.ts
dexter/src/tools/finance/estimates.ts
dexter/src/tools/finance/filings.ts
dexter/src/tools/finance/fundamentals.ts
dexter/src/tools/finance/index.ts
dexter/src/tools/finance/metrics.ts
dexter/src/tools/finance/news.ts
dexter/src/tools/finance/prices.ts
dexter/src/tools/finance/segments.ts
dexter/src/tools/index.ts
dexter/src/tools/search/index.ts
dexter/src/tools/search/tavily.ts
dexter/src/tools/types.ts
dexter/src/utils/config.ts
dexter/src/utils/context.ts
dexter/src/utils/env.ts
dexter/src/utils/index.ts
dexter/src/utils/message-history.ts
dexter/tsconfig.json
libs/dexter/client.go
libs/dexter/client_test.go
libs/dexter/go.mod
libs/dexter/mock.go
libs/dexter/mock_test.go
libs/utcp/dexter_service.go
libs/utcp/dexter_service_test.go
libs/utcp/dexter_types.go
```

The embedded tree contained 57 tracked files and `libs/dexter` contained 5;
the three Dexter-only UTCP files bring the exact tracked deletion set to 65.
Historical/archive files were not deleted.

## Runtime and contract changes

- `cmd/research` no longer accepts or constructs `DEXTER_SERVICE_URL` and now
  creates a generic `NewToolRunner()` with no Dexter field.
- `internal/modules/orchestration` no longer exposes `DexterClient`,
  `DexterClientAdapter`, `WithDexter`, or `ResearchQueries`. Agent0, memory,
  strategies, audit and the generic bounded tool boundary remain.
- `libs/utcp` no longer exposes Dexter service/types or treats `dexter` as a
  truth-path provider. Shared UTCP services remain.
- `libs/ingest` retains memory and SQL helpers while removing only the
  Dexter-specific payload/observation parser and timestamp projection.
- Memory fixtures use `jax-research` as the source identity rather than the
  removed external service.

## Configuration and dependencies

- Removed `./libs/dexter` from `go.work`.
- Removed `useDexter` from `config/jax-core.json`.
- Removed the Dexter provider from `config/providers.json`.
- Removed the Dexter environment binding from the research Compose service.
- Removed the Dexter-only trader-import deny-list entry.
- No root `go.mod` dependency or lockfile required modification.
- Anthropic support remains where independently used by Agent0; Financial
  Datasets remains where independently used by Jax market-data/research paths.
  Google Gemini and Tavily were Dexter-only in the deleted tree and were not
  removed globally because broader vendor cleanup is outside CR-02A.

## Documentation and history

Supported architecture docs now describe Agent0, memory and Jax-native tools;
the current roadmap identifies CR-02A as the active cleanup package. CR-01 is
retained as a pre-removal audit snapshot with a superseded-state note. ADR-0012
retains its historical body with a banner stating that its Dexter references
are no longer current. Archive and accepted historical evidence were preserved
to avoid rewriting the record.

## Safety and financial status

This cleanup made no trading or execution changes. Phase 12 remains research
only; `HYP-EVENT-001A` remains `NOT VALIDATED` and
`TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE` remains in force. Phase
13 remains `NOT STARTED / BLOCKED BY CLEANUP GATE`. No hosted AI inference,
broker request, paid provider call, credential change or external spend was
made.

## Verification

Targeted verification passed after the code cleanup:

- `go test ./internal/modules/orchestration ./libs/utcp ./libs/ingest ./libs/contracts ./libs/observability ./cmd/research -count=1`

The full CR-02A verification also passed:

- `go test ./... -count=1`
- `go vet ./...`
- frontend `npm run test` (56 files, 156 tests), `npm run typecheck`, and `npm run build`
- `go test ./tests/golden ./tests/replay -count=1`
- provider/orchestration/UTCP/ingest focused tests listed above
- `docker compose config --quiet`
- `scripts/check-trader-imports.ps1`
- `git diff --check`

The deleted Dexter Bun tests were intentionally not run because the supported
runtime no longer contains that application. Frontend build warnings were
non-blocking existing chunk-size/Browserslist warnings.

## Adversarial CR-02A review

The fresh review attacks hidden active references, stale environment/config
seams, `go.work` module inclusion, old provider removal, shared-provider
regressions, generic tool-runner behavior, historical evidence preservation,
and execution/safety boundary changes. Required final disposition:

**Blocking CR-02A findings remaining: 0**

The final supported-runtime search must contain no active Go, frontend,
configuration, Compose or CI Dexter reference. Remaining matches are limited
to the CR-02A record, CR-01's historical audit snapshot, explicit historical
ADR/evidence, archive content and guardrail/history context.

## Decision and what's left

Implementation status: **COMPLETE — EXTERNAL CR-02A REVIEW REQUIRED**.  
Recommended external decision: **GO CR-02A**.

What's left:

- External technical-lead review of this bounded CR-02A evidence.
- CR-02B Agent0 removal/consolidation is not started.
- CR-02C broader LiteLLM/vendor/dependency cleanup is not started.
- Phase 13 remains not started and blocked by the commercial-readiness gate.
