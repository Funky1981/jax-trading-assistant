# CORE-READINESS-02B2 — Alpaca Quote Temporal Semantics Correction

Status: **IMPLEMENTED / VALIDATION IN PROGRESS / EXTERNAL REVIEW REQUIRED**.
This package corrects quote/trade temporal provenance only. It does not execute
CORE-READINESS-02B, create a candidate or pilot, or make an economic claim.

## Provenance contract

- `marketdata.Quote.Timestamp` is the provider timestamp for the bid/ask quote.
- `marketdata.Quote.TradeTimestamp` is the provider timestamp for the latest
  trade whose price is exposed as `Quote.Price`.
- Alpaca requires both `LatestQuote` and `LatestTrade`; it never substitutes a
  midpoint or one record's timestamp for the other.
- `quotes.timestamp` stores the bid/ask quote timestamp;
  `quotes.last_trade_timestamp` stores the latest-trade timestamp;
  `quotes.received_at` is captured after the provider response is returned and
  decoded. Request-start and response-receipt times remain distinct in ingest
  diagnostics. Existing rows receive no guessed trade timestamp and remain
  unchanged by migration 000080.
- `EconomicObservation.ProviderAt`, `LastProviderAt`, and `ReceivedAt` retain
  those distinct times without clamping.

## Temporal rules

The shared quote validator allows provider timestamps to exceed local
`ReceivedAt` by at most 250 ms. Larger differences fail with
`PROVIDER_CLOCK_SKEW_EXCEEDED`. The allowance is quote-only and does not
rewrite any timestamp.

`AvailableAt = max(ReceivedAt, ProviderAt, LastProviderAt)`. Economic use is
permitted only when `asOf >= AvailableAt`. Both quote and trade ages must be no
greater than the configured quote maximum age (60 seconds for the 02B proof).
Thus a 77 ms cross-clock difference can pass after all three timestamps, but
the same observation is rejected at an earlier `asOf`; stale quote or stale
trade independently rejects the pair.

Canonical economic loads fail closed when `last_trade_timestamp` is absent.
Legacy rows remain stored and are not backfilled. The canonical PAPER tick
carries quote time, trade time, receipt time, and its economic `asOf` through
PaperVenue validation.

## Candle boundary

The 02B1 candle contract is unchanged: completion must be at or before `asOf`,
ingestion must be at or after completion, and no partial or future candle is
eligible. The quote skew allowance is not applied to candles.

## Validation and evidence

Deterministic tests cover Alpaca timestamp binding, 77 ms acceptance only after
availability, 251 ms rejection, independent quote/trade freshness, and
PaperVenue causal validation. Disposable PostgreSQL tests exercise the
production canonical loader and verify persistence/read-back of all three
timestamps, while proving that legacy rows without trade provenance stay NULL
and cannot be loaded economically.

The prior proof's measured +77 ms result does not, by itself, establish whether
its comparison used request-start or response-complete time unless the original
measurement log records both. This package captures the two separately going
forward; absent contemporaneous prior evidence, the old measurement basis is
reported as unknown rather than inferred.

CORE-READINESS-02B remains **BLOCKED / NOT YET EXECUTED** until this correction
receives external review and the separate proof is authorized. The World
Monitor cursor remains untouched by this package. No normal database or 02B
proof database is used; no broker call or IB Bridge use is authorized.
