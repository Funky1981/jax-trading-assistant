# Diagnostic-only OpenAI hosted A1 provider

This capability is isolated from the operational AI-shadow provider and from the accepted WP-00.02 evidence. The next experimental A1 cell explicitly selects `gpt-5.6-luna`; the already accepted `gpt-5.6-sol` selection remains supported. No arbitrary OpenAI model is accepted. This diagnostic does not add OpenAI to the normal Jax runtime, connect to Postgres, or grant any trading-state mutation.

This package authorizes implementation, mocked verification, and no-network preflight only. Do not run `--execute` without a later architecture decision that separately authorizes paid inference.

## API, model identity, and comparison contract

- Endpoint: OpenAI Responses API, `POST https://api.openai.com/v1/responses`.
- Authentication: bearer API key read only from `JAX_OPENAI_EXPERIMENT_API_KEY`.
- Allowed requested models: exact `gpt-5.6-luna` or exact `gpt-5.6-sol`.
- Luna returned identity: exact `gpt-5.6-luna`. No separately dated Luna snapshot is currently published, so Jax does not claim immutable weights or dated reproducibility.
- Sol returned identity: the accepted exact alias or an alias-prefixed dated identity remains supported.
- Provider evidence: requested and returned model identities, optional provider-returned `system_fingerprint`, response/request IDs, token/cache usage, cost by category, remaining budget, provider errors, timeouts, and budget state.
- Reasoning: explicit `none`, matching the non-thinking baseline posture.
- State: `store=false`.
- Structured outputs, tools, web search, file search, retrieval, and application-level response caching: not enabled. The unchanged v4 prompt, Jax parser, validator, resolver, and single corrective retry remain authoritative.
- Transport retries: none. Provider errors and timeouts are recorded; only the existing corrective contract retry is allowed.

The implementation follows the official [GPT-5.6 guidance](https://developers.openai.com/api/docs/guides/model-guidance?model=gpt-5.6), [GPT-5.6 Luna model contract](https://developers.openai.com/api/docs/models/gpt-5.6-luna), [Responses API reference](https://developers.openai.com/api/reference/resources/responses/methods/create), and dedicated [API pricing page](https://developers.openai.com/api/docs/pricing).

## Luna no-network configuration

Set every value in the current process. Use only a fake credential for implementation/preflight verification. Never place a real key in a command-line argument, repository file, diagnostic artifact, database, or captured log.

```powershell
$env:JAX_AI_SHADOW_ENABLED = "true"
$env:JAX_AI_PROVIDER = "openai"
$env:JAX_AI_MODEL = "gpt-5.6-luna"
$env:JAX_AI_TIMEOUT_SECONDS = "120"
$env:JAX_AI_MAX_EVENTS = "48"
$env:JAX_AI_DIAGNOSTIC_REPETITIONS = "1"
$env:JAX_AI_EXPERIMENT_ID = "A1"
$env:JAX_AI_REASONING_EFFORT = "none"
$env:JAX_AI_MAX_OUTPUT_TOKENS = "256"
$env:JAX_AI_EXPERIMENT_BUDGET_USD = "0.12"
$env:JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS = "0.20"
$env:JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS = "0.02"
$env:JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS = "0.25"
$env:JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS = "1.20"
$env:JAX_AI_HOSTED_INFERENCE_AUTHORIZED = "false"
$env:JAX_RUNTIME_MODE = "paper"
$env:ALLOW_LIVE_TRADING = "false"
$env:EXECUTION_ENABLED = "false"
$env:EXECUTION_INSTRUCTION_WORKER_ENABLED = "false"
$env:BROKER_EXECUTION_ALLOWED = "false"
$env:MAX_LEVERAGE = "1"
$env:JAX_OPENAI_EXPERIMENT_API_KEY = "sk-fake-preflight-only"
```

The four price inputs are execution-plan metadata, not permanent business-logic constants. The dedicated pricing page is authoritative for this experiment. Standard short-context Luna prices verified for this package are $0.20 input, $0.02 cached input, $0.25 cache writes, and $1.20 output per million tokens. The Luna model page may display older pricing; this discrepancy is documented and is not a blocker. Re-verify the dedicated pricing page immediately before any paid run.

## Conservative one-repetition ceiling

The largest rendered initial request across the exact frozen 48-case corpus is 2,334 UTF-8 bytes before the provider client's 1,024-token protocol allowance. A conservative corrective request is 2,419 bytes: it reserves four UTF-8 bytes per each of 256 possible output tokens in both the previous-response and validation-evidence fields.

The complete-run estimate assumes all 48 initial requests and all 48 authorized corrective retries, 256 output tokens on every request, 1,024 protocol tokens on every request, and prices every input token at the higher $0.25 cache-write rate. That bound is $0.110718. The proposed hard ceiling is therefore **$0.12**, the smallest two-decimal ceiling above the conservative full-run estimate. The loader rejects a higher Luna ceiling, and preflight rejects a selected execution shape whose conservative estimate exceeds the configured ceiling.

This ceiling is not spending authorization. Successful provider usage replaces each reservation with uncached-input, cached-input, cache-write, and output costs. Timeouts, unreadable responses, HTTP 408, and HTTP 5xx retain conservative ambiguous liability. Calculable cost plus ambiguous liability cannot authorize a request beyond the ceiling.

## No-network preflight

```powershell
go run ./cmd/ai-shadow-issuer-diagnostic --preflight
```

Preflight validates provider/model identity, explicit reasoning, the one-repetition 48-case execution shape, the frozen manifest byte hash and semantic fingerprint, every input fingerprint, v4 prompt/output versions, deterministic policy, API-key presence, budget/pricing inputs, isolated append-only evidence namespace, and paper/live safety. It reports `provider_contact=false` and `inference=false` and returns before provider construction, request construction, or HTTP execution.

## Future paid Luna execution — DO NOT RUN WITHOUT ARCHITECTURE APPROVAL

Immediately before an approved run, re-verify the dedicated pricing page, repeat preflight with authorization still false, then explicitly set:

```powershell
$env:JAX_AI_HOSTED_INFERENCE_AUTHORIZED = "true"
go run ./cmd/ai-shadow-issuer-diagnostic --execute
```

Do not use this command in the implementation/preflight package.
