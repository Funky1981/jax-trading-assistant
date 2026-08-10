# Diagnostic-only hosted A1 provider

This capability is isolated from the operational AI-shadow provider and from the accepted WP-00.02 evidence. It targets only the future `A1` issuer-diagnostic cell with `gpt-5.6-sol`. It does not add OpenAI to the normal Jax runtime, connect to Postgres, or grant any trading-state mutation.

WP-00.03B authorizes implementation and no-inference preflight only. Do not run `--execute` until WP-00.03C separately authorizes paid inference.

## API and comparison contract

- Endpoint: OpenAI Responses API, `POST https://api.openai.com/v1/responses`.
- Authentication: bearer API key read only from `JAX_OPENAI_EXPERIMENT_API_KEY`.
- Model: exact requested identifier `gpt-5.6-sol`.
- Reasoning: explicit `none`, matching the non-thinking baseline posture.
- State: `store=false`.
- Structured outputs: deliberately not enabled for initial A1. The unchanged v4 prompt requests JSON, then the existing Jax v4 parser, validator, resolver and single corrective retry remain authoritative. Provider-side JSON Schema is reserved for a separate conditional experiment because it would remove part of the output-contract nuisance measured in WP-00.02.
- Transport retries: none. Provider errors and timeouts are recorded; only the existing one corrective retry after a contract-invalid response is allowed.

The implementation follows the current official [GPT-5.6 Sol model contract](https://developers.openai.com/api/docs/models/gpt-5.6-sol), [Responses API guidance](https://developers.openai.com/api/docs/guides/migrate-to-responses), [authentication reference](https://developers.openai.com/api/reference/overview), [Structured Outputs guidance](https://developers.openai.com/api/docs/guides/structured-outputs), and [error guidance](https://developers.openai.com/api/docs/guides/error-codes).

## Required environment

Set every value in the current process. Never place the real API key in a command-line argument, repository file, diagnostic artifact, database, or captured log.

```powershell
$env:JAX_AI_SHADOW_ENABLED = "true"
$env:JAX_AI_PROVIDER = "openai"
$env:JAX_AI_MODEL = "gpt-5.6-sol"
$env:JAX_AI_TIMEOUT_SECONDS = "120"
$env:JAX_AI_MAX_EVENTS = "48"
$env:JAX_AI_EXPERIMENT_ID = "A1"
$env:JAX_AI_REASONING_EFFORT = "none"
$env:JAX_AI_MAX_OUTPUT_TOKENS = "256"
$env:JAX_AI_EXPERIMENT_BUDGET_USD = "1.00"
$env:JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS = "5.00"
$env:JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS = "0.50"
$env:JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS = "6.25"
$env:JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS = "30.00"
$env:JAX_AI_HOSTED_INFERENCE_AUTHORIZED = "false"
$env:JAX_RUNTIME_MODE = "paper"
$env:ALLOW_LIVE_TRADING = "false"
$env:EXECUTION_ENABLED = "false"
$env:EXECUTION_INSTRUCTION_WORKER_ENABLED = "false"
$env:BROKER_EXECUTION_ALLOWED = "false"
$env:MAX_LEVERAGE = "1"
$env:JAX_OPENAI_EXPERIMENT_API_KEY = Read-Host "Temporary OpenAI experiment API key"
```

The four price inputs are execution-plan metadata, not authoritative pricing embedded in business logic. The displayed `$5.00` input, `$0.50` cached-input, `$6.25` cache-write (1.25 times uncached input), and `$30.00` output rates per million tokens were verified for standard GPT-5.6 Sol pricing during WP-00.03B and must be re-verified immediately before any paid run.

## No-inference preflight

```powershell
go run ./cmd/ai-shadow-issuer-diagnostic --preflight
```

Preflight validates provider/model identity, the frozen 48-case manifest and byte hash, semantic and per-input fingerprints, v4 prompt/output versions, deterministic policy, three-repetition execution shape, API-key presence, budget/pricing inputs, the isolated hosted evidence namespace, and paper/live safety. It writes one append-only JSON record below `.runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-a1-v1/A1/preflight/` and returns before provider construction or HTTP execution.

## Future paid execution command — do not run in WP-00.03B

Only after WP-00.03C approval, re-verify pricing, repeat preflight, and explicitly set:

```powershell
$env:JAX_AI_HOSTED_INFERENCE_AUTHORIZED = "true"
go run ./cmd/ai-shadow-issuer-diagnostic --execute
```

The budget guard reserves a conservative per-request maximum before every request using a byte-count upper bound plus protocol overhead and the configured maximum output. It rejects an unaffordable request before HTTP. Successful usage replaces the reservation with calculable input/output cost; network timeouts, unreadable responses, HTTP 408, and HTTP 5xx outcomes retain their conservative reservation as ambiguous liability. Calculable cost plus ambiguous liability can never authorize another request beyond the configured ceiling.
