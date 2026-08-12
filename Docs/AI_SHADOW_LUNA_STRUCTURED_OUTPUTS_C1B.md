# WP-00.03C1B — Luna Structured Outputs cell

## Experimental identity

- Cell: `WP-00.03C1B`
- Evidence namespace: `openai-hosted-c1b-structured-outputs-v1/WP-00.03C1B`
- Provider/model: `openai` / `gpt-5.6-luna`
- Contract mode: `openai-responses-json-schema-strict`
- Frozen prompt: `ai-shadow-prompt-v4-issuer-resolution`
- Canonical contract: `ai-shadow-output-v4-issuer-resolution`
- Repetitions/cases: `1 × 48`

This is a new provider-enforced experimental cell. It is not a repetition of the unconstrained OpenAI A1 cell in `openai-hosted-a1-v1/A1`.

## Request mapping

The existing system and user messages remain unchanged. The OpenAI Responses request adds only the provider enforcement mapping below:

```json
{
  "model": "gpt-5.6-luna",
  "input": [
    {"role": "system", "content": "<frozen v4 system prompt>"},
    {"role": "user", "content": "<frozen v4 event or corrective prompt>"}
  ],
  "reasoning": {"effort": "none"},
  "max_output_tokens": 256,
  "store": false,
  "text": {
    "format": {
      "type": "json_schema",
      "name": "jax_ai_shadow_output_v4_issuer_resolution",
      "strict": true,
      "schema": "<the request's ProviderRequest.Schema object>"
    }
  }
}
```

No tools, web search, retrieval, or additional model instructions are sent. The `schema` value is not copied or redefined: it is the same `ProviderRequest.Schema` constructed by `OutputSchema`, including the resolver-derived proxy vocabulary. Both initial and corrective requests use that path.

## Compatibility determination

The canonical root is an object, every one of its ten properties is required, and `additionalProperties` is false. Its property nodes use strings, arrays, enums, `minLength`, `maxLength`, `maxItems`, and string `items`. These constructs are supported for non-fine-tuned Structured Outputs models under the current OpenAI contract. The root is not `anyOf`, and the schema contains no unsupported composition, reference, conditional, numeric, or pattern keywords. No semantic wire conversion is required.

The adapter validates this supported subset before any HTTP call. A missing schema, non-object root, non-required property, open additional properties, or unsupported keyword/type fails closed. Mocked tests serialize the request and compare its decoded wire schema semantically with the canonical `ProviderRequest.Schema`.

Official contract references checked for this package:

- <https://developers.openai.com/api/docs/guides/structured-outputs>
- <https://developers.openai.com/api/docs/models/gpt-5.6-luna>

## Isolation and safety

`JAX_AI_OUTPUT_CONTRACT_MODE=openai-responses-json-schema-strict` is required together with cell `WP-00.03C1B` and model `gpt-5.6-luna`. The existing A1 experiment permits only the legacy `prompt-only` mode, so it cannot silently acquire Structured Outputs. Other experiment IDs, provider/model combinations, or enforcement modes fail closed.

The C1B preflight requires one explicit repetition and retains the existing 48-case order and single corrective retry. It performs no provider call or inference and records database and trading mutation as false. Paid execution remains separately guarded by `JAX_AI_HOSTED_INFERENCE_AUTHORIZED`.

## Conservative budget proposal

At implementation review, the supplied planning rates are USD 1.00/M uncached input, USD 0.10/M cached input, USD 1.25/M cache writes, and USD 6.00/M output. These values are runtime inputs, not permanent source-code truth, and must be reverified immediately before any paid execution.

The offline estimator serializes every frozen initial request including the Structured Outputs schema. It also prices 48 worst-case corrective requests, each containing bounded prior-response and validation evidence, with 256 output tokens per request. It assumes no cache hits and applies the higher cache-write input rate to every estimated input token. Its deliberately conservative byte-as-token allowance plus a 1,024-token per-request margin gives:

- largest initial wire request: 3,760 bytes
- conservative corrective wire request: 3,804 bytes
- maximum requests: 96
- estimated maximum: USD 0.721729
- proposed hard ceiling: USD 0.75

This ceiling is a cell guard, not a forecast of actual spend or a reusable price assertion.
