# Jax Commercial-Readiness CR-02C — AI Provider / LiteLLM Consolidation

Date: 2026-09-10  
Repository: `C:\Projects\Jax\jax-trading-assistant`  
Branch: `capability-reset`  
Starting HEAD: `095410e5a0278ac71c2a6cd65b9d1784fd9c8483`

## Scope and decision

CR-02C consolidates the overlapping operational planner/chat model paths into
a small Jax-owned inference boundary. It does not change the diagnostic
AI-shadow clients, the separate memory embedding contract, recommendation
logic, or any execution capability. CR-02D and Phase 13 were not started.

The resulting rule is:

```text
Jax feature -> Jax inference.Client -> deterministic/offline,
               OpenAI adapter, or local Ollama adapter
```

Business logic does not depend on LiteLLM or on a vendor-specific HTTP client.
The deterministic default remains zero-cost and network-free. Networked
providers require explicit `JAX_MODEL_PROVIDER` selection.

## Current active path inventory

| Path | Caller / use | Transport | Classification | Decision |
| --- | --- | --- | --- | --- |
| Planner | `cmd/research` → orchestration → planner | Former LiteLLM-compatible chat endpoint | KEEP, consolidated | Uses `internal/modules/inference` through `llmcontext.InferenceProvider` |
| Assistant chat | `cmd/trader` → `internal/modules/chat` | Duplicated OpenAI-compatible chat client with Jax tool catalog | KEEP, consolidated | Uses the same inference transport; tool validation remains in the Jax harness |
| AI-shadow OpenAI | `cmd/ai-shadow-*`, HYP diagnostics | OpenAI Responses API with strict experiment budgets and provenance | SPECIALISED / JUSTIFIED | Retained; different contract and authorization boundary |
| AI-shadow Ollama | diagnostic benchmark and local shadow runs | Native Ollama `/api/chat` | SPECIALISED / JUSTIFIED | Retained; shadow-only policies/configuration remain separate |
| Memory embeddings | `cmd/research` memory proxy / `libs/pgmemory` | Embeddings API or deterministic local embedder | SPECIALISED / JUSTIFIED | Retained as an embedding-specific contract |
| LiteLLM gateway | Optional `config/litellm` stack | External gateway + Ollama/OpenAI routing | REMOVE | No active root runtime consumer or unique required capability |

No supported path calls Agent0, `/v1/plan`, `/v1/execute`, `/suggest`, or
`/chat` on an Agent0 service. No root Compose service or supported configuration
requires port 8093 or `AGENT0_SERVICE_URL`.

## LiteLLM value test

`LITELLM PROVIDES UNIQUE REQUIRED CAPABILITY = NO`.

Source inspection showed that LiteLLM supplied an OpenAI-compatible HTTP hop,
model aliases, and optional routing to Ollama/OpenAI. Jax already owns the
planner contract, request/result validation, deterministic policy, model route,
budget governor, usage logging, and orchestration audit. The active chat path
already implemented its own OpenAI-compatible request and tool-call decoding.
The optional LiteLLM Compose stack was not part of the root runtime and did not
provide a capability required by a supported Jax caller. Keeping it would
retain an unnecessary service, virtual-key configuration, and duplicate route.

## Jax-owned inference boundary

`internal/modules/inference` defines `Client`, bounded messages, tool
definitions, request/response usage, provider identity, explicit configuration,
and three provider modes:

- `deterministic` is the safe default and has no network client;
- `openai` uses a small Jax-owned OpenAI-compatible `/v1/chat/completions`
  adapter and requires explicit API configuration;
- `ollama` uses a small Jax-owned local `/api/chat` adapter and no credential.

`JAX_MODEL_PROVIDER`, `JAX_MODEL_BASE_URL`, `JAX_MODEL_MODEL`, and
`JAX_MODEL_API_KEY` are the operational model configuration. Existing
`OPENAI_API_KEY`, `OPENAI_BASE_URL`, and `OPENAI_MODEL` remain supported as
provider-specific fallback inputs for compatibility; they do not select a
networked provider. `JAX_AI_*` remains reserved for the specialized AI-shadow
contract. LiteLLM gateway variables are no longer read by supported runtime
code.

`llmcontext.InferenceProvider` adapts the shared transport to the existing
prompt, routing, budget, and usage service. `planner.ModelProvider` therefore
retains the Jax planner contract without any gateway dependency. Chat adapts the
same transport and retains tool calls as data for the existing Jax harness;
provider transport itself has no broker, execution, portfolio, or approval API.

## Configuration and safety

Provider selection is explicit and fail-closed. Unknown providers, missing
OpenAI configuration, invalid URLs, malformed responses, empty choices/content,
invalid requests, and provider HTTP failures return errors. A failed networked
provider does not silently become a deterministic plan. Deterministic mode
does not instantiate a network client.

OpenAI and Ollama responses retain provider/model identity, request ID where
available, and token usage. Planner validation still rejects unbounded or
authority-bearing actions. Chat tool calls remain subject to the Jax harness
registry, schema, permission, and execution boundaries.

Specialized AI-shadow OpenAI cost/accounting and authorization were not merged
into the operational transport because their experiment contract includes
distinct Responses API schemas, pricing schedules, holdouts, and authorization
controls. Memory embeddings likewise remain separate because they return
vectors rather than advisory completions.

The following invariants remain unchanged:

- `ALLOW_LIVE_TRADING=false`;
- `BROKER_EXECUTION_ALLOWED=false`;
- live worker disabled;
- maximum leverage `1x`;
- no approval, recommendation, candidate, ticket, paper/live order, trade,
  fill, or portfolio mutation was created by this cleanup;
- no hosted inference or paid provider call was made; spend `$0.00`.

## Files changed

- Added `internal/modules/inference/client.go` and contract tests.
- Added `internal/modules/llmcontext/inference_provider.go` and usage metadata
  fields.
- Updated planner orchestration, chat, Compose, and example configuration to
  use the shared Jax transport.
- Removed `internal/modules/llmcontext/litellm.go` and its tests.
- Removed the optional `config/litellm` Compose/configuration stack.
- Updated focused tests and historical/current commercial-readiness notes.

## Verification

Focused verification completed before the final repository-wide run:

- inference adapter tests: PASS;
- OpenAI fake-server request/response, usage, malformed-response and bounded
  request tests: PASS;
- Ollama fake-server adapter test: PASS;
- planner model-provider and orchestration configuration tests: PASS;
- chat configuration tests: PASS;
- existing context/cache/usage logger tests: PASS.

No hosted inference, broker call, or credential-bearing request is used by
tests. The final handover records the complete repository verification.

## Deferred work

CR-02C intentionally does not consolidate the specialized AI-shadow or memory
embedding provider contracts, does not perform broad vendor/data cleanup, and
does not decide whether any future provider should be added. These findings,
including any remaining OpenAI/Ollama configuration aliases and cost-policy
integration opportunities, are inputs to CR-02D if separately authorized.

Phase 13 remains `NOT STARTED / BLOCKED BY CLEANUP GATE`. Scientific status is
unchanged: `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE`.
