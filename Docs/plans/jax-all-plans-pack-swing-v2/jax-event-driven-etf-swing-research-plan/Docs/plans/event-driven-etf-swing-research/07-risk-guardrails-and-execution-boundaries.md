# 07 — Risk Guardrails and Execution Boundaries

## Goal

Make the system safe, deterministic, and auditable.

## Authority Model

```text
World Monitor / news feeds = awareness
AI provider = summarisation and research assistance
cmd/research = evidence engine
cmd/trader = candidate and guardrail engine
human = approval authority
ib-bridge = paper broker boundary
```

## Hard Rule

```text
Only cmd/trader can create paper execution instructions.
Only after guardrails pass.
Only after human approval.
```

## Guardrail Evaluation Contract

```json
{
  "evaluationScope": "candidate_submission",
  "passed": false,
  "hardReject": true,
  "checks": {
    "etfAllowlist": "pass",
    "paperMode": "pass",
    "liveTradingBlocked": "pass",
    "approvalRequired": "pass",
    "quoteFreshness": "fail",
    "spread": "not_evaluated",
    "marketSession": "pass",
    "stopLoss": "pass",
    "horizonPolicy": "pass",
    "confounders": "pass",
    "calendarRisk": "warn"
  },
  "failures": ["quote_stale"],
  "brokerMode": "paper",
  "marketSession": "regular",
  "quoteTimestamp": "2026-06-14T14:35:00Z"
}
```

## Guardrail States

Use explicit values:

```text
pass
fail
warn
not_applicable
not_evaluated
```

Do not use `true` where proof is absent.

## Universal Hard Rejects

Reject candidate before approval if:

```text
single-name stock
non-allowlisted ETF
leveraged ETF
inverse ETF
volatility ETF
options/futures/crypto
live mode enabled
broker mode unknown
missing evidence bundle
missing stop-loss
missing horizon policy
AI output tries to place order
manual order bypass attempted
```

## Swing-Specific Hard Rejects

Reject swing candidate if:

```text
overnight risk not explicitly allowed
max hold days missing
max hold days > 10 in phase 1
no daily revalidation schedule
weekend hold allowed without explicit operator flag
high-impact calendar event within forbidden window
unresolved high-relevance confounder
historical sample count below minimum
```

## Intraday-Specific Hard Rejects

Reject intraday candidate if:

```text
flatten-by-close false
outside RTH
quote stale
spread too wide
same-session expiry missing
target-like move already happened
```

## Execution Boundary Tests

- AI response cannot create execution instruction.
- Research endpoint cannot create execution instruction.
- Candidate cannot create execution instruction without approval.
- Approval cannot create execution instruction if guardrails fail.
- Live broker mode cannot create instruction in phase 1.
- Manual ETF order ticket must route through candidate approval workflow.

## Paper Mode Proof

Paper mode must be proven from at least one runtime source:

```text
ib-bridge /health or /mode endpoint
trader runtime config
pilot-status endpoint
UAT evidence artifact
```

The evidence must be captured in UAT and persisted.

## Audit Trail

Every candidate should link:

```text
source_event_id
research_thesis_id
evidence_bundle_id
guardrail_evaluation_id
approval_id
execution_instruction_id
paper_order_id
reflection_id
```
