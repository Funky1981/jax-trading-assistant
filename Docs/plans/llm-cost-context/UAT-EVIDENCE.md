# LLM Cost Context UAT Evidence

Date: 2026-06-04

## Scope

This evidence covers the runtime implementation for `Docs/plans/llm-cost-context`:

- local-first LiteLLM gateway config
- prompt package and cacheable prefix split
- cost estimation and budget gate
- model routing
- planned/actual usage logging
- exact prompt cache policy
- durable memory artifacts and bounded retrieval
- safe compression zones
- optional Headroom trial path
- event clustering and duplicate suppression
- template output limits
- cost rollups and AI overview cost summary
- chat model calls routed through the cost/context gate

## Verification Commands

```powershell
go test ./internal/modules/llmcontext ./db/postgres/migrations
go test ./internal/modules/chat ./cmd/trader
go test ./config/litellm
go test ./...
```

## Headroom Trial Evidence

Headroom remains disabled by default.

Validated behavior:

- disabled config returns `headroom_disabled`
- Zone A trading truth is skipped
- Zone B deterministic-only data is skipped
- live approval workflow compression is skipped
- Zone C article/document context can be compressed
- source IDs, retrieval key, content hash, token savings, and latency are recorded

Automated coverage:

```powershell
go test ./internal/modules/llmcontext -run Headroom
```

## Safety Evidence

Validated behavior:

- eligibility gate blocks non-allowlisted symbols, duplicate events, stale quotes, wide spreads, priced-in/unclear verdicts, missing evidence bundles, live trading paths, and budget failures
- blocked model calls are logged before provider invocation
- approval packets require uncompressed trading truth
- exact cache rejects approval/current-market/broker tasks
- direct OpenAI provider use is blocked unless `AI_ALLOW_DIRECT_PROVIDER=true`
- configured chat LLM calls are wrapped by the LLM cost/context gate

## Cost Visibility Evidence

Validated behavior:

- `llm_usage_logs` stores planned and actual usage
- `llm_cost_rollups` aggregates strategy, symbol, and task type cost windows
- AI overview includes today/month spend, cost per candidate, cost per approved trade, paid calls avoided, cache hit rate, Headroom token savings, rejected-before-AI count, and top expensive workflows

## Remaining Operational Step

Before production-like use, run migrations through the normal DB migration command and set a local LiteLLM virtual key:

```powershell
$env:AI_GATEWAY_BASE_URL="http://localhost:4000"
$env:AI_GATEWAY_API_KEY="<litellm-virtual-key>"
$env:AI_DEFAULT_MODEL="local-small"
```
