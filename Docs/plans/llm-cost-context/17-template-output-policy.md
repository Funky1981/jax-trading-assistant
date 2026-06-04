# 16 — Template Output Policy

## Purpose

Reduce output tokens and keep messages consistent.

## Rule

Use templates for repeated user-facing outputs.

The model should only fill the parts requiring language judgement.

## Approval Template

```text
ETF: {{symbol}}
Strategy: {{strategy_name}}
Action: {{paper_action}}
Confidence: {{confidence}}

Why:
{{model_reason_max_80_words}}

Priced-in:
{{priced_in_verdict}} — {{priced_in_reason_max_40_words}}

Risk:
Entry {{entry}}, stop {{stop_loss}}, target {{take_profit}}, risk {{risk_amount}}.

Expires:
{{expires_at}}

Decision:
Approve / Reject / Snooze
```

## Reject Template

```text
No trade.

Reason:
{{deterministic_reject_reason}}

Evidence:
{{key_metric_1}}
{{key_metric_2}}
{{key_metric_3}}
```

## Daily Digest Template

```text
ETF News Digest

Top themes:
{{themes}}

Possible watchlist:
{{watchlist}}

No-trade reasons:
{{no_trade_summary}}

System status:
{{provider_status}}
```

## Output Limits

```text
approval summary: max 180 words
reject reason: max 80 words
post-trade reflection: max 300 words
daily digest: max 500 words
strategy explanation: max 250 words
```

## Acceptance Criteria

- Templates exist in repo.
- Model output max length is enforced.
- Approval summaries stay short.
- Reject summaries can be deterministic with no model call.
