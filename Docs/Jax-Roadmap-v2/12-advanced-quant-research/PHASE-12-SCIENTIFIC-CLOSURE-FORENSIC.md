# Phase 12 Scientific Closure Forensic

## API cost reconciliation — mandatory closure item

This addendum preserves the historical internal figure and records the cost
accounting defect investigation. No additional hosted inference was run.

### Evidence and frozen pricing

- Hypothesis: `HYP-EVENT-001A`.
- Model identity: `gpt-5.6-luna`.
- Persisted successful provider-response records: 939 unique events / request
  records, plus retry metadata indicating 3 additional provider calls.
- Persisted usage totals: 12,721,590 input tokens, 99,989 cached input tokens,
  and 126,229 output tokens. All persisted records report zero cache-write
  tokens; the historical response records did not retain a cache-write field.
- Maximum persisted request input: 243,715 tokens. Requests above the
  long-context threshold of 272,000: **none**. Consequently there are no
  request/event identities or surcharges to list for that category.
- Frozen pricing schedule: `openai:gpt-5.6-luna:standard-text:v1`, checked
  2026-09-09: input `$0.20`/million, cached input `$0.02`/million, cache write
  `$0.25`/million (1.25x uncached input), output `$1.20`/million. The provider
  long-context rule is request-specific: input above 272,000 applies 2x input
  and 1.5x output to the full request. See the [official GPT-5.6 Luna pricing
  page](https://developers.openai.com/api/docs/models/gpt-5.6-luna).

### Reconciliation results

| Measure | USD | Meaning |
|---|---:|---|
| `ORIGINAL INTERNAL COST ESTIMATE` | `$2.677034` | Historical sum stored on the 939 final result records; it used the old per-request calculator and did not include missing retry usage. |
| Corrected persisted-response total | `$2.678539` | Recomputed per request with component-level micro-USD ceiling, cached input, cache-write field when present, and request-level long-context logic. It covers only the 939 persisted successful responses. |
| User-reported provider/dashboard total | `$3.340000` | Default project / All APIs / 2026-09-09 UTC, supplied by the user; no local provider export was available to partition it. |
| Difference from corrected persisted-response total | `$0.661461` | Unresolved: missing retry usage and/or unrelated project API usage. |

The exact full HYP-EVENT provider-accounted total cannot be established from
the retained artifacts because the three retry calls have no persisted provider
usage records. No evidence permits assigning the entire `$3.34` dashboard
amount to HYP-EVENT, and no unrelated call has been identified locally.

### Root cause and corrective implementation

The HYP classifier persisted usage only on the final structured-output-valid
result. A provider response that consumed tokens and was later rejected by
the validator could increment the retry counter without retaining its request
usage. Its old calculator also lacked an explicit cache-write field, pricing
schedule identity, and reusable request-level surcharge accounting.

The corrective implementation adds the reusable `aishadow` OpenAI request
calculator and makes it authoritative for successful provider responses. It
accounts separately for uncached input, cached input, cache writes, output,
and a request-specific long-context surcharge. The classifier now persists a
provider-usage audit record immediately after every successful provider
response, before structured-output validation, so retry calls are retained.
Result records also carry the pricing schedule and request breakdown for
future runs. No historical result was rewritten and no new API call was made.

### Regression coverage

`internal/modules/aishadow/openai_cost_test.go` covers ordinary uncached
requests, cached and cache-write input, retries in aggregate accounting, the
threshold immediately below, exactly at, and above 272,000 tokens, mixed
normal/surcharged requests, aggregate equality to individual request sums,
and usage reconciliation failure. Existing classifier and diagnostic tests
also pass.

### Required closure status

**API COST ACCOUNTING: IMPLEMENTATION CORRECTED; HISTORICAL BILLING NOT RECONCILABLE**

The reusable calculator defect is fixed and regression-tested. The corrected
retained-response total is `$2.678539`; it is not presented as the complete
experiment bill. Exact historical provider billing cannot be reconstructed
from the retained artifacts because three retry calls lack usage records and
the dashboard total cannot be partitioned to HYP-EVENT. The approved
disposition is to preserve that limitation rather than invent a reconciliation
or run additional inference. The original `$2.677034` value is preserved as
`ORIGINAL INTERNAL COST ESTIMATE`.
