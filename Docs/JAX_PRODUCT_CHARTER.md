# Jax Product Charter

## Product truth

Jax is an **EVENT-DRIVEN TRADING RESEARCH + DECISION SYSTEM**. It watches
events and evidence, resolves affected assets, forms and challenges causal
theses, rejects weak setups, requires human approval, starts with isolated
exploratory paper, and learns from every decision including `NO_TRADE`.

## Initial trader

The initial trader is the **EVENT-DRIVEN SHORT-HORIZON SWING TRADER**.

```text
event/news
→ asset resolution
→ evidence
→ causal thesis
→ quant/technical context
→ risk
→ human approval
→ exploratory paper
→ continuous thesis monitoring
→ exit
→ learning
```

The initial horizon is 1–5 trading sessions, typically 2–3, with a hard
maximum of five trading sessions. Events, news, and source-backed evidence are
the catalyst. Technicals support, confirm, or veto; they are not a sole
catalyst.

## Required behaviour

Jax must:

1. Ingest or receive a market event with provenance.
2. Resolve the issuer and affected instrument or remain unresolved.
3. Corroborate evidence and expose contradictions and unknowns.
4. Form a causal thesis with explicit invalidation conditions.
5. Use deterministic quant and risk context as supporting gates.
6. Default to `NO_TRADE`.
7. Require human approval before every exploratory paper entry.
8. Monitor relevant new evidence and exit deterministically or recommend exit.
9. Store decisions, provenance, policy versions, and outcomes.
10. Learn from watched, rejected, no-trade, and closed cases.

## Modes and evidence

`EXPLORATORY_PAPER` is product-learning mode. It is paper-only and may evolve
through prospective, versioned rules. It does not demonstrate an edge.

`FORMAL_FORWARD_PAPER` is a later scientific mode with a frozen policy,
future-only identity, preregistered gates, and separate authorization.

`DEMONSTRATED_EDGE` is an evidence conclusion, not a trading mode. Neither
exploratory paper profitability nor a paper label implies it.

## Non-negotiable boundaries

- `NO_TRADE` is the normal/default decision.
- Evidence comes before inference; no model-memory substitute for provenance.
- Human approval is mandatory for exploratory entries.
- Paper first; no live execution is currently authorized.
- No autonomous broker execution, live orders, or guaranteed profit claims.
- No day-trading or generic multi-week technical swing policy in the initial
  trader.

## Scientific position

`ma_crossover_v1` is closed after failed historical validation. Trading edge is
not demonstrated. PAPER-01 is implemented and requires external review;
PAPER-02 is not started or authorized; formal forward paper is not started;
Phase 13 is blocked.
