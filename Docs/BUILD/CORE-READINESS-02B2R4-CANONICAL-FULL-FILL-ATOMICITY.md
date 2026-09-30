# CORE-READINESS-02B2R4 — Canonical Full-Fill Atomicity Gate

Status: **IMPLEMENTED / EXTERNAL REVIEW REQUIRED**.

Starting SHA: `555384f48b2d0f62d68e0fd2fb1a04efee5a9fbb`.

The corrected CORE-READINESS-02B2 → CORE-READINESS-02B2R3R1 chain is **GO /
EXTERNALLY REVIEWED** at that starting SHA. CORE-READINESS-02B remains
**BLOCKED / NOT EXECUTED**. Host clock synchronization is a separate blocker.
This package makes no hypothesis-readiness or trading-edge claim.

## Runtime boundary

The generic `PaperVenue` continues to support partial fills. The canonical
exploratory Working-Jax-v1 lifecycle currently represents one atomic entry fill
and one atomic exit fill; it does not aggregate multiple fills into a position
transition. Accordingly, the runtime checks unconsumed displayed quantity
before invoking the venue. Insufficient liquidity is a normal pending state:
the durable order remains pending, the entry queue is not consumed, and no
lifecycle or outcome is created. On a later sufficiently liquid observation,
the runtime requires exactly one fill and a FILLED order with zero remaining
quantity before persisting the lifecycle transition. Exit persistence also
requires the PAPER ledger to be flat for the closed instrument.

## Proof and scope

The guarded disposable-PostgreSQL OPS-01 runtime proof covers a 10-unit entry
with an initial 4-unit quote, process restart, later full liquidity, replay,
and an approved 10-unit exit with an initial 3-unit quote, restart, later full
liquidity, flat-ledger verification, outcome persistence, and replay. The direct
PaperVenue partial-fill regression remains unchanged.

The package does not execute CORE-READINESS-02B, access the normal Jax
database, consume the normal World Monitor cursor, make live Alpaca requests,
use IB Bridge, approve a genuine opportunity, create or activate a pilot,
select or tune a strategy, or enable broker/live execution.
