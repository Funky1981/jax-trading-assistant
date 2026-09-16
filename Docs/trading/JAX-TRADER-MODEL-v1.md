# JAX Trader Model v1

## EVENT-DRIVEN SHORT-HORIZON SWING TRADER

PAPER-01 is an exploratory paper-only learning loop. It is not evidence of a
trading edge, a formal forward paper, or authorization for live execution.

```text
EVENT/NEWS
  -> ISSUER/ASSET RESOLUTION
  -> EVIDENCE/CORROBORATION
  -> TRADE THESIS
  -> QUANT/TECHNICAL CONFIRMATION
  -> PORTFOLIO RISK
  -> HUMAN APPROVAL
  -> EXPLORATORY PAPER POSITION
  -> CONTINUOUS THESIS MONITORING
  -> EXIT
  -> OUTCOME/ATTRIBUTION/LEARNING
```

The event and causal mechanism are the catalyst. Quantitative and technical
indicators are supporting confirmation, risk sizing, and invalidation context;
`SMA20 > SMA50` alone cannot create an event-driven trade.

## Modes

`EXPLORATORY_PAPER` is prospective product learning. Real prospective event and
news evidence may be used, execution is paper-only, every new entry requires
human approval, and versioned rules may evolve between exploratory observations.
It does not prove an edge and may not be retrospectively relabelled as formal.

`FORMAL_FORWARD_PAPER` is a later, separately authorized scientific mode. It
requires a frozen policy, future-only run identity, preregistered gates, and no
tuning during the run. Exploratory records cannot populate that run.

## Horizon and state

The expected horizon is 1–5 trading sessions, usually 2–3. There is no
long-term hold. The initial rationale, stop, target, hard time exit, and policy
versions are frozen at entry. An explicit known session calendar is required;
unknown calendar state fails closed.

Thesis states are `VALID`, `STRENGTHENED`, `WEAKENED`, and `INVALIDATED`.
Operational states add `REVIEW_REQUIRED`, `EXIT_RECOMMENDED`,
`EXIT_AT_NEXT_TRADABLE_SESSION`, and `CLOSED`. Every transition is append-only
and retains its timestamp, evidence, reason, provenance, and policy version.

## Safety boundary

The loop uses the existing workflow, human approval, portfolio-risk, and
`internal/modules/papertrading` contracts. It requires `PAPER`,
`ExecutionAuthority=NONE`, `ALLOW_LIVE_TRADING=false`,
`BROKER_EXECUTION_ALLOWED=false`, `EXECUTION_ENABLED=false`, and maximum
leverage of 1x. There is no broker adapter, live order, or Phase-09 mutation.
