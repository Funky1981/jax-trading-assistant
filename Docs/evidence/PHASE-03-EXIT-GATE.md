# Phase 03 Exit-Gate Demonstration

## Gate requirement

Jax can construct a source-linked evidence packet for a representative US
equity/ETF using real market, company and macro evidence without relying on
model memory.

This document records the bounded 2026-09-07 demonstration. WP-03.06 is
already `COMPLETE / GO`; this document does not award the Phase 03 decision.

## Representative entity

- Instrument: `ins_aapl_common`, AAPL, Apple Inc. common stock.
- Issuer: `iss_apple`, Apple Inc., SEC CIK `0000320193`.
- The instrument-to-issuer link is the accepted canonical issuer role.

## Real evidence matrix

| Family | Provider/source | Real vs fixture | Raw payload | Digest | Normalized evidence | Result |
| --- | --- | --- | --- | --- | --- | --- |
| Market | Alpaca Stock Market Data API / `src_alpaca_stock_bars_sip_unadjusted` | `LIVE_ACQUIRED_NOW` | `rpa_phase03_gate_market_alpaca_live` | `710b0cdd1d718b08b8ebdc66bf6a4fb89db15fac60856388cd0454239012e7f5` | `obs_f829e4bf6f53af2c9112e92b`, AAPL daily bar, 2026-09-04: O 328.305 / H 328.93 / L 317.86 / C 319.97 / V 39,788,274 | PASS |
| Company | SEC EDGAR submissions / `src_sec_submissions` | `LIVE_ACQUIRED_NOW` | `rpa_phase03_gate_sec_live` | `7c67278a0c4009db2cfd62ac23fbdb16d935867ad44fdfde9eaef77837386440` | filing `evd_862e273f3c7a6afe34947f4b`, issuer `iss_apple`, CIK `0000320193` | PASS |
| Macro/context | U.S. Treasury Daily Par Yield Curve / `src_treasury_daily_par_yield_curve` | `LIVE_ACQUIRED_NOW` | `rpa_phase03_gate_treasury_live` | `a80704b25ed50f21426206eb9d4bf9f34360f6e3dd302a90dd91b3c2cc74ccc2` | Treasury 10-year macro evidence lineage `evd_65b77552184e067e63c2118a` | PASS |
| Release/calendar | Accepted WP-03.04 integration | `UNAVAILABLE` | None | None | Not required by the authoritative market/company/macro gate | NOT A BLOCKER |
| Evidence quality | WP-03.06 Cboe VIX ↔ FRED `VIXCLS` mapping | `SYNTHETIC_FIXTURE` | Minimal test references only | N/A | Deterministic diagnostics remain fixture-backed; same-origin redistribution is preserved | PASS, FIXTURE-BACKED |

Each live item was persisted to `RawPayloadStore` before parsing and retains
provider/source identity, raw acquisition ID, SHA-256 digest, acquisition time,
and normalized provenance. The integrated harness used one process-local memory
store for the demonstration; it did not copy provider payload bytes into this
document.

## Packet and assertion

The demonstration constructs the existing `jax.phase03_exit_packet/v1` packet:

- packet ID: `p03_ca3d753b08cbbbda01b90605`;
- canonical instrument: `ins_aapl_common`;
- canonical issuer: `iss_apple`;
- real families: MARKET, COMPANY, MACRO_CONTEXT;
- diagnostic ID: `eqc_vix_fred_fixture_v1`, explicitly fixture-backed.

`Packet.AssertRealExitCondition` passed. It remains fail-closed: a fixture
cannot satisfy a required real family, and every real item must carry a valid
raw source identity, digest, and provenance covering that acquisition.

## Reproducible execution

The normal suite remains non-live. The bounded live command is:

```text
$env:JAX_RUN_LIVE_PHASE03_GATE='1'
go test ./libs/phase03gate -run TestPhase03ExitGateLiveAAPLPacket -count=1 -v
```

Observed result:

```text
PHASE 03 EXIT CONDITION DEMONSTRATED
PASS
```

The source-specific smoke tests are also retained:

```text
go test ./libs/marketdata -run TestPhase03ExitGateLiveAAPLMarketEvidence -count=1 -v
go test ./libs/sec -run TestPhase03ExitGateLiveAAPLCompanyEvidence -count=1 -v
$env:JAX_RUN_LIVE_WP0305='1'
go test ./libs/treasury ./libs/cboe -run 'Test(TreasuryLiveOfficialXMLSmoke|CboeLiveOfficialCSVSmoke)$' -count=1 -v
```

Credentials are read from the existing process configuration only. Values are
never printed, persisted, or included in packet provenance.

## Temporal and source limitations

The Alpaca timestamp is retained as a provider bar timestamp at the documented
New York date boundary; it is not converted into an invented exchange-close
timestamp. Observation/report/filing/acceptance/public-availability/acquisition
boundaries remain separate. Current acquisition of historical data does not
prove historical knowability at an earlier decision time.

The SEC filing retains filing date and acceptance time; public availability is
not fabricated. Treasury observation date and acquisition time remain separate.
Release/calendar evidence was not fabricated. FRED was not required for this
gate and no live FRED comparison is claimed.

## Decision candidate

`PHASE 03 EXIT CONDITION DEMONSTRATED`

Awaiting technical-lead Phase-03 GO review. Phase 04 remains not started.
