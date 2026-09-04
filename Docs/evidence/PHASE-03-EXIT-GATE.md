# Phase 03 Exit-Gate Demonstration

## Gate requirement

Jax can construct a source-linked evidence packet for a representative US
equity/ETF using real market, company and macro evidence without relying on
model memory.

This closure is a bounded verification of that condition. WP-03.06 has already
received independent technical-lead `FINAL GO`; this document does not award a
Phase 03 decision.

## Configuration preflight

Using the existing ignored `.env` configuration mechanism for the live child
process, the non-secret preflight reported:

- `FINANCIAL_DATASETS_API_KEY`: PRESENT;
- `SEC_USER_AGENT`: PRESENT;
- `SEC_CONTACT`: PRESENT.

No credential values were emitted, logged, persisted, or committed.

## Representative entity

The candidate remains `AAPL / Apple Inc.` using the accepted canonical
identities:

- instrument: `ins_aapl_common` (`jax.instrument/v1`), ticker boundary `AAPL`;
- issuer: `iss_apple` (`jax.issuer/v1`), SEC CIK `0000320193`;
- instrument-to-issuer relationship: canonical issuer role `issuer`.

No new identity system was created.

## Demonstration contract and assertion

`libs/phase03gate` provides the smallest phase-gate-only contract,
`jax.phase03_exit_packet/v1`. It links canonical instrument and issuer
objects to provider-neutral evidence items. Each item carries its origin
classification, normalized contract reference, the accepted
`provider.RawPayloadRef`, temporal metadata, and canonical provenance.

`Packet.AssertRealExitCondition` fails closed unless the packet validates and
contains all three required families—`MARKET`, `COMPANY`, and
`MACRO_CONTEXT`—with origin `LIVE_ACQUIRED_NOW` or
`PREVIOUSLY_PERSISTED_REAL_SOURCE_EVIDENCE`. A `SYNTHETIC_FIXTURE` item can
prove deterministic contract behaviour but cannot satisfy the gate. Every item
also has to retain a valid raw source identity, content digest, acquisition
identity, and provenance input covering that digest.

The deterministic packet test proves stable IDs, canonical AAPL identity,
fixture rejection, missing-family rejection, and raw-source provenance
failure. The package is not the future Phase 06 research packet and contains no
recommendation, signal, valuation, or execution field.

## Real evidence matrix

| Family | Provider/source | Real vs fixture | Acquisition path | Normalized evidence | Raw payload / provenance | Temporal limitation | Result |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Market | Financial Datasets / `src_financial_datasets_historical_prices` | No accepted real item | Accepted WP-03.01 hardened daily-bars path was invoked with AAPL during this closure | None; live request failed authentication before normalization | No raw payload was retained; no real packet item exists | No market observation date can be asserted from this failed request | BLOCKED: `AUTHENTICATION`, provider path `market.bars` |
| Company | SEC EDGAR / `src_sec_submissions` | No accepted real item | Accepted SEC submissions path was invoked in the explicit live test with configured identity | None; provider operation returned `NOT_FOUND` before raw persistence | No raw payload was retained; no real packet item exists | No Apple filing/report/acceptance date is asserted by this closure | BLOCKED: SEC submissions operation `NOT_FOUND` |
| Macro/context | U.S. Treasury / `src_treasury_daily_par_yield_curve` | LIVE ACQUIRED NOW | Accepted WP-03.05 `AcquireYear` path, official XML, 2026 request | Normalized Treasury observations; live sample included `mobs_fb4286119f65131feaf49720`, 30 Yr, 2026-09-03, source value `5.25` | `rpa_treasury_live_smoke`; digest `0ad39aa28aa955d3cab2968cdb347fc2632dd05e52ca2f99aa8619ed42988e9f`; raw ref and normalized provenance retained in the run's memory store | Observation date is 2026-09-03; acquired 2026-09-04. This does not prove historical knowability | REAL MACRO EVIDENCE |
| Macro/context (optional) | Cboe / `src_cboe_vix_daily_history` | LIVE ACQUIRED NOW | Accepted WP-03.05 `AcquireHistory` path, official CSV | Normalized live sample `mobs_87f4a063fabf19b0dc036893`, 2026-09-03, source value `14.320000` | `rpa_cboe_live_smoke`; digest `8dba02f54d435d58249c57e45dc84842a0b1bb7f256af8708080072209337569`; raw ref and normalized provenance retained in the run's memory store | Observation date is separate from 2026-09-04 acquisition; no exchange-close timestamp was manufactured | REAL CONTEXT EVIDENCE |
| Release/calendar | Accepted release/calendar integration | UNAVAILABLE for this packet | No live release acquisition was needed or fabricated | None | None | Existing WP-03.04 BLS/live-access limitation remains | Not a blocker for the authoritative market/company/macro gate, but absent |
| Evidence quality | `libs/evidencequality`, approved Cboe VIX ↔ FRED `VIXCLS` mapping | SYNTHETIC FIXTURE | Deterministic unit tests only; no approved `FRED_API_KEY` was configured and no live FRED comparison was claimed | `AGREE`/divergence/missingness/semantic-negative controls are covered by WP-03.06 tests | Test provenance uses minimal synthetic raw references; no copyrighted FRED/Cboe raw data was copied into this document | Fixture dates do not establish live availability or historical knowability | Diagnostic framework verified; not a real packet family |

The Treasury and Cboe smoke runs were successful and retained exact raw bytes in
their existing in-memory `RawPayloadStore` for the run. They do not make the
AAPL packet complete: macro evidence alone cannot satisfy the market and
company requirements.

## Reproducible commands and results

Normal deterministic packet and contract checks:

```text
go test ./libs/phase03gate -count=1
```

Result: PASS. The fixture-backed packet test rejects fixture-only evidence and
proves the real-evidence assertion fails closed.

The bounded gate preflight itself is also reproducible:

```text
go build -o .runtime/phase03-exit-gate.exe ./cmd/phase03-exit-gate
& .runtime/phase03-exit-gate.exe
```

Result: exit code `2`, with JSON identifying canonical AAPL/Apple IDs and the
market and company blockers. It does not claim the separately acquired macro
smoke bytes as packet items without their raw references.

Accepted official keyless macro verification:

```text
$env:JAX_RUN_LIVE_WP0305='1'
go test ./libs/treasury ./libs/cboe -run 'Test(TreasuryLiveOfficialXMLSmoke|CboeLiveOfficialCSVSmoke)$' -count=1 -v
```

Result: PASS for both. The run logged the raw acquisition IDs, SHA-256
digests, normalized IDs, dates, source values, and acquisition times recorded
in the matrix above.

Explicit AAPL market live verification:

```text
$env:JAX_RUN_LIVE_PHASE03_GATE='1'
go test ./libs/marketdata -run TestPhase03ExitGateLiveAAPLMarketEvidence -count=1 -v
```

Result: FAIL CLOSED. The accepted provider path returned a non-retryable
`AUTHENTICATION` failure for provider `pvd_financial_datasets`, capability
`market.bars`, so no raw or normalized real market evidence was claimed.

Explicit Apple SEC live verification:

```text
$env:JAX_RUN_LIVE_PHASE03_GATE='1'
go test ./libs/sec -run TestPhase03ExitGateLiveAAPLCompanyEvidence -count=1 -v
```

Result: FAIL CLOSED. The configured accepted SEC submissions operation returned
`NOT_FOUND`; no SEC raw bytes were claimed. No identity or contact information
was invented.

The live tests are opt-in; the ordinary test suite does not depend on the
internet or credentials. The market test uses the existing approved local
configuration when an operator supplies it; the SEC test uses the existing
approved `SEC_USER_AGENT` and `SEC_CONTACT` configuration. Credentials are
never logged or persisted in packet provenance.

## Gate limitations

- No accepted real AAPL market raw acquisition was available or produced.
- No accepted real Apple SEC raw acquisition was available or produced; the
  configured submissions request returned `NOT_FOUND`.
- Therefore no packet satisfying the real-evidence assertion was constructed.
- Treasury/Cboe real evidence was acquired, but it is context only for this
  failed AAPL gate demonstration.
- The successful macro acquisitions were current closure-time observations;
  they do not establish that Jax knew those observations at an earlier
  historical decision time.
- `ObservationDate`, `ReportDate`, `FilingDate`, `AcceptanceDateTime`,
  `PublicationTime`, `PublicAvailabilityTime`, and `AcquiredAt` remain separate;
  absent source metadata is not fabricated.
- The Cboe/FRED diagnostic remains fixture-backed and preserves the accepted
  same-origin/redistribution lineage rather than claiming two independent
  origins. FRED raw data was not duplicated.

## Decision candidate

`PHASE 03 EXIT CONDITION NOT YET DEMONSTRATED`

Awaiting technical-lead Phase-03 gate review.
