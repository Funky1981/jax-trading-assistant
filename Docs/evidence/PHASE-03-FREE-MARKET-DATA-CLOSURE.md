# Phase 03 Free Development Market-Data Closure

## Roadmap adjustment

Financial Datasets remains an accepted WP-03.01 provider option. Its hardened
adapter is preserved and its request/header path was verified; the configured
external credential returned HTTP 401 during development verification. Jax
will not make Phase-03 development acceptance contingent on paying recurring
market-data fees. Financial Datasets is therefore recorded as:

`APPROVED PROVIDER — LIVE DEVELOPMENT ACCESS BLOCKED BY EXTERNAL CREDENTIAL / PAID ACCESS DECISION`

Alpaca Basic was evaluated as the bounded zero-cost development evidence
source. This is an additive source decision and does not rewrite WP-03.01
acceptance or approve Alpaca as a permanent production or serious-backtesting
provider.

## Alpaca evaluation

- Cost: Basic plan is documented by Alpaca as `$0/month`.
- Authentication: API key/secret pair, sent as `APCA-API-KEY-ID` and
  `APCA-API-SECRET-KEY`; values remain outside source and logs.
- Historical access: Alpaca documents US stock/ETF historical data and
  coverage from approximately 2016; the account accepted the bounded
  completed-date SIP and IEX requests used here.
- Rate limit: Alpaca documents approximately 200 historical API calls/minute
  for the Basic plan; this adapter additionally bounds each request to 1,000
  rows and 32 pages.
- Feed: the gate uses explicit `feed=sip`. SIP is consolidated US exchange
  coverage. Explicit `feed=iex` also succeeded in a control request, but IEX is
  a single-exchange feed and its volume is not whole-market volume.
- Adjustment: the gate uses explicit `adjustment=raw`, represented by Jax as
  `UNADJUSTED`. The adapter currently rejects implicit or other adjustment
  modes rather than comparing differently adjusted histories.
- Time: Alpaca daily timestamps are preserved as provider timestamps and mapped
  to the documented New York date boundary. Jax does not manufacture a close
  timestamp.
- Completeness: the adapter persists and reads back every page before parsing,
  validates token progression, rejects duplicate bar intervals and metrics,
  enforces ordering, bounds pages, and retains the requested feed/timeframe.
- Licensing/redistribution: the raw-retention policy remains replay-audit with
  redistribution not authorized. Production redistribution and commercial
  backtest licensing require a separate review of Alpaca terms and the chosen
  feed entitlement.
- Suitability: suitable for this development evidence demonstration. Not yet
  approved for production or serious backtesting without separate review of
  entitlements, feed completeness, corporate-action treatment, survivorship,
  historical availability, and licensing.

## Existing code assessment

The repository contained a legacy SDK-based Alpaca provider for quotes,
candles, trades and related DTOs. It did not satisfy the accepted WP-03.01
raw-first canonical evidence boundary. The new bounded path reuses the
existing `AlpacaProvider`, provider registry, operational executor,
`RawPayloadStore`, normalization pipeline, canonical AAPL identity, and
provider-neutral market contracts. It does not resurrect the legacy candle
path and does not create duplicate canonical contracts.

## Live verification

The direct historical controls returned HTTP 200 for both explicit `feed=sip`
and explicit `feed=iex` on completed AAPL daily-bar dates. The Phase-03 live
path selected SIP and produced:

- provider: `pvd_alpaca_market_data`;
- source: `src_alpaca_stock_bars_sip_unadjusted`;
- raw payload: `rpa_phase03_gate_market_alpaca_live`;
- digest: `710b0cdd1d718b08b8ebdc66bf6a4fb89db15fac60856388cd0454239012e7f5`;
- normalized close observation: `obs_f829e4bf6f53af2c9112e92b`;
- canonical instrument: `ins_aapl_common`;
- bar: 2026-09-04, open 328.305, high 328.93, low 317.86, close 319.97,
  volume 39,788,274;
- adjustment: raw/unadjusted;
- provenance: normalized observation lineage covers the exact raw digest and
  source acquisition.

The accepted SEC Apple evidence and Treasury evidence were acquired in the
same integrated gate process. Their raw IDs/digests and normalized lineage are
recorded in `PHASE-03-EXIT-GATE.md`.

## Packet result

The existing `jax.phase03_exit_packet/v1` packet was constructed for AAPL /
Apple Inc. with real market, company, and macro/context items. Its deterministic
ID was `p03_ca3d753b08cbbbda01b90605`. `Packet.AssertRealExitCondition` passed.
The evidence-quality diagnostic remains clearly fixture-backed and does not
claim live FRED data or two independent origins for Cboe/FRED.

## Limitations

This closure proves source-linked evidence capability for one completed AAPL
interval and one current Treasury acquisition. It does not prove historical
knowability, production suitability, full-market completeness for all feeds,
or serious-backtesting validity. It adds no recommendation, signal, valuation,
execution, approval, or Phase 04 capability.

## Status

`PHASE 03 EXIT CONDITION DEMONSTRATED`

Phase 03 remains `EXIT CONDITION DEMONSTRATED / awaiting technical-lead GO`.
Phase 04 remains `NOT STARTED`.
