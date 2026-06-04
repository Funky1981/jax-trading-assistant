# 04 — Guardrails and Safety

## Core Rule

World Monitor is a radar. Jax is the evidence engine. The broker is only touched after human approval.

## Hard Safety Rules

- World Monitor cannot place trades.
- World Monitor cannot approve trades.
- World Monitor cannot create broker instructions.
- World Monitor cannot set runtime mode.
- World Monitor cannot enable live trading.
- World Monitor cannot bypass Jax approval flow.

## Jax Must Treat Every Trigger As Untrusted

Every incoming trigger starts as:

```text
unverified_external_research_trigger
```

It becomes useful only after Jax validates it.

## Required Jax Checks

Before a candidate trade can be offered:

1. Source count check.
2. Source quality check.
3. Timestamp freshness check.
4. Duplicate check.
5. ETF allowlist check.
6. Event-to-ETF mapping reason.
7. Historical reaction check.
8. Priced-in check.
9. Confounder check.
10. Risk guardrail check.
11. Paper-only mode check.
12. Human approval requirement.

## No-Go Conditions

Immediately reject or quarantine if:

- Trigger asks Jax to execute.
- Trigger contains no source URLs.
- Trigger is stale.
- Trigger maps only to non-ETF assets.
- Trigger maps to leveraged, inverse, volatility, or excluded ETFs.
- Trigger conflicts with Jax runtime mode.
- Source quality is unknown and no corroboration exists.
- The event is too vague to research.

## Audit Trail Requirements

Every accepted trigger must store:

- inbound payload
- received timestamp
- dedupe key
- validation result
- rejection reason, if rejected
- source URLs
- mapped ETFs
- research run id, if researched
- candidate id, if created

## Human Approval Boundary

Even if Jax creates a candidate, the result must remain:

```text
awaiting_human_approval
```

No automatic approval in phase 1.
