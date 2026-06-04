# 03 — Model Routing Policy

Initial local-only routing:

```json
{
  "event_classification": "local-small",
  "etf_mapping": "local-small",
  "historical_summary": "local-small",
  "evidence_bundle_summary": "local-small",
  "approval_mobile_summary": "local-small",
  "priced_in_explanation": "local-small",
  "complex_conflicting_news_review": "disabled",
  "post_trade_reflection": "local-small"
}
```

Paid later:

```json
{
  "evidence_bundle_summary": "paid-cheap",
  "approval_mobile_summary": "paid-cheap",
  "complex_conflicting_news_review": "paid-strong"
}
```

Escalation does not force trade. If uncertain: no trade.
