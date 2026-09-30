# CORE-READINESS-02B2R4R1 — Single-Instrument Lifecycle Exclusivity

Status: **IMPLEMENTATION VALIDATED / EXTERNAL REVIEW REQUIRED**.

Starting SHA: `cf41fdd2090899dc1727f8704397575264ff56cd`.

CORE-READINESS-02B2R3R1 remains **GO / EXTERNALLY REVIEWED**. CORE-READINESS-
02B2R4 implemented canonical full-fill atomicity and is superseded for final
acceptance by this package's additional exclusivity boundary. CORE-READINESS-02B
remains **BLOCKED / NOT EXECUTED**; host clock synchronization remains separate.
This work makes no hypothesis-readiness or trading-edge claim.

## Working Jax v1 accounting invariant

For one server-configured PAPER account and canonical instrument, the canonical
handoff permits at most one pending approved entry, pending canonical order,
open exploratory lifecycle, or non-flat ledger exposure. Approval acquires a
PostgreSQL session advisory lock over the account/instrument pair before opening
its serializable transaction, then checks durable queues, orders, lifecycles,
and ledger events. An exact candidate replay remains idempotent. A different
candidate for the occupied instrument receives a canonical conflict; other
instruments and other PAPER accounts remain independent.

The exploratory runtime repeats the durable-state check before entry economics
and rejects a non-flat same-instrument account ledger. Before an approved exit,
the account's instrument quantity must equal that lifecycle's entry-fill
quantity within the canonical tolerance. Outcome closure still requires a
complete fill and a flat instrument ledger. A durably closed lifecycle with a
zero ledger quantity releases the instrument for a later candidate.

This is a v1 accounting invariant, not a strategy rule. Multi-lot aggregation
and multiple simultaneous same-instrument exploratory lifecycles remain
unsupported. Generic PaperVenue partial-fill capability remains available;
the canonical exploratory lifecycle still requires atomic full entry and exit
fills.

## Validation boundary

The guarded disposable PostgreSQL tests cover same-account same-instrument
approval serialization, one winning approved queue, runtime durable exposure
checks, pending-order detection, different-instrument and cross-account
independence, and release after a complete close. Existing OPS-01 PostgreSQL
coverage continues to prove pending entry/exit restart, full-fill completion,
ledger reconciliation, and idempotent replay.

No normal Jax database, normal World Monitor cursor, live Alpaca request,
IB Bridge, genuine opportunity, pilot activation, strategy selection, broker
execution, or live trading is used or authorized by this package.
