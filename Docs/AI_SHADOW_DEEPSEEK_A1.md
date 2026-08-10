# Diagnostic-only DeepSeek A1 provider

This WP-00.03C0 capability is a separate, narrow implementation of the existing Jax diagnostic `Provider` abstraction. The corrective execution gate adds only an explicit diagnostic repetition selector; it does not alter the accepted OpenAI Responses client, add DeepSeek to the operational AI-shadow runtime, connect to Postgres, or grant any trading-state mutation. This package authorizes implementation, mocked verification, and no-network preflight only. Do not run hosted inference until a later architecture decision explicitly authorizes WP-00.03C1.

## Official API contract verified for C0

- OpenAI-format base URL: `https://api.deepseek.com`.
- Endpoint: `POST /chat/completions`.
- Authentication: bearer API key read only from `JAX_DEEPSEEK_EXPERIMENT_API_KEY`.
- Requested and accepted returned model identifier: exact `deepseek-v4-pro`.
- Mode: explicit `thinking: {"type":"disabled"}`. DeepSeek currently defaults thinking to enabled, so omission is not acceptable for this cell.
- Structured output and tools: not enabled. The unchanged Jax v4 prompt, parser, validator, deterministic resolver, and one corrective retry remain authoritative.
- Response evidence: body response ID, header request ID where returned, exact returned model, required non-empty `system_fingerprint`, finish reason, prompt/cache-hit/cache-miss/completion/reasoning/total token usage, provider errors, and timeout/budget state.
- Transport retries: none. The only retry remains Jax's existing one corrective contract retry.

The implementation was checked against the official [Chat Completions contract](https://api-docs.deepseek.com/api/create-chat-completion), [Thinking Mode contract](https://api-docs.deepseek.com/guides/thinking_mode), [authentication contract](https://api-docs.deepseek.com/api/deepseek-api/), [error codes](https://api-docs.deepseek.com/quick_start/error_codes/), and [models and pricing](https://api-docs.deepseek.com/quick_start/pricing).

`deepseek-v4-pro` is a service model name, not a dated immutable snapshot identifier. Jax records the exact returned alias and `system_fingerprint`; it does not claim that the alias identifies immutable weights. A missing fingerprint, any returned model other than exact `deepseek-v4-pro`, non-zero reasoning tokens, or returned reasoning content fails and stops the experiment.

## No-network preflight

Set these values only in the current process. Never place the real key in a CLI argument, repository file, diagnostic artifact, database, or captured log.

```powershell
$env:JAX_AI_SHADOW_ENABLED = "true"
$env:JAX_AI_PROVIDER = "deepseek"
$env:JAX_AI_MODEL = "deepseek-v4-pro"
$env:JAX_AI_TIMEOUT_SECONDS = "120"
$env:JAX_AI_MAX_EVENTS = "48"
$env:JAX_AI_DIAGNOSTIC_REPETITIONS = "1"
$env:JAX_AI_EXPERIMENT_ID = "A1"
$env:JAX_AI_REASONING_EFFORT = "none"
$env:JAX_AI_THINKING_MODE = "disabled"
$env:JAX_AI_MAX_OUTPUT_TOKENS = "256"
$env:JAX_AI_EXPERIMENT_BUDGET_USD = "0.15"
$env:JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS = "0.435"
$env:JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS = "0.003625"
$env:JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS = "0.87"
$env:JAX_AI_HOSTED_INFERENCE_AUTHORIZED = "false"
$env:JAX_RUNTIME_MODE = "paper"
$env:ALLOW_LIVE_TRADING = "false"
$env:EXECUTION_ENABLED = "false"
$env:EXECUTION_INSTRUCTION_WORKER_ENABLED = "false"
$env:BROKER_EXECUTION_ALLOWED = "false"
$env:MAX_LEVERAGE = "1"
$env:JAX_DEEPSEEK_EXPERIMENT_API_KEY = Read-Host "Temporary DeepSeek experiment API key"
go run ./cmd/ai-shadow-issuer-diagnostic --preflight
```

Preflight validates provider/model/mode identity, API-key presence, the frozen 48-case manifest and byte hash, semantic and per-input fingerprints, v4 prompt/output versions, deterministic policy, execution-time budget/pricing inputs, isolated `deepseek-hosted-a1-v1/A1` evidence namespace, paper/live safety, false hosted-inference authorization, and prohibited database/trading mutation. It records the repetition environment, whether an override was supplied, requested and effective repetitions, cases per repetition, and total planned cases. With the setting above, the shape is one repetition and 48 planned cases. If `JAX_AI_DIAGNOSTIC_REPETITIONS` is absent, the frozen default remains three repetitions and 144 planned cases. Only explicit values `1` and `3` are accepted; malformed, zero, negative, or unsupported values fail closed.

The append-only JSON record is written below `.runtime/diagnostics/ai-shadow-issuer-hosted/deepseek-hosted-a1-v1/A1/preflight/`. Preflight returns before provider construction, request construction, HTTP transport, or inference.

The listed rates were verified from official documentation during C0. They remain execution-plan metadata and must be re-verified immediately before any later paid or granted-credit run.

## Proposed later C1 ceiling

The largest frozen initial request is 2,334 UTF-8 bytes before protocol overhead. DeepSeek's official approximation of 0.3 token per English character gives about 701 input tokens; allowing 256 output tokens at current cache-miss pricing is about $0.00053 per base request. The authorised C1 shape contains 48 base requests, approximately $0.025 before cache benefits. Allowing a conservative corrective-retry and estimation margin retains the proposed hard ceiling of **$0.15**. The frozen default remains three repetitions (144 base requests), and the implementation also imposes an absolute configuration cap of $0.25.

This is a ceiling, not spending authorization. Actual provider-reported cache-hit/cache-miss usage replaces each request reservation. Timeouts, unreadable responses, HTTP 408, and HTTP 5xx retain conservative ambiguous liability. Calculable cost plus ambiguous liability cannot authorize another request beyond the configured ceiling.

## Future C1 preflight command — hosted inference is not authorised here

After a separate WP-00.03C1 GO and re-verification of the official API/pricing contract, use the configuration above with the explicit one-repetition setting and authorization still false:

```powershell
$env:JAX_AI_DIAGNOSTIC_REPETITIONS = "1"
$env:JAX_AI_HOSTED_INFERENCE_AUTHORIZED = "false"
go run ./cmd/ai-shadow-issuer-diagnostic --preflight
```

This preflight records one repetition and 48 planned cases without constructing or contacting the provider. Do not use `--execute` in this corrective package.
