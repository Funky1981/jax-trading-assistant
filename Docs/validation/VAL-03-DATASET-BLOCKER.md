# VAL-03 Dataset Blocker

## Status

`VAL-03 = DATASET_BLOCKED — CORPORATE ACTION PROVENANCE`.

The frozen VAL-02C manifest is verified at contract version `v1.3` with
SHA-256 `09738525839a49d7b5d725e52d0b8547e768ce1d82e212e63c110ae1b2d6f65e`.
The five frozen implementation identities also match the manifest. No VAL-03
performance calculation, formal OOS scoring, 2025 access, broker call, paid
provider call or forward-paper action occurred.

## Required contract that cannot currently be proved

The manifest requires raw/unadjusted Alpaca daily bars plus complete,
point-in-time effective handling of splits and dividends for the nine-ETF
basket from 2015-01-01 through 2024-12-31. Missing or ambiguous corporate
action provenance must cause abstention; retrospective adjustment is
prohibited.

## Repository evidence

The frozen Alpaca implementation at `libs/marketdata/alpaca_hardened.go`
implements the hardened historical-bars route only. It has no corporate-action
acquisition, storage or normalization path. The repository contains no
complete VAL-03 corporate-action dataset for the required basket and period.
Existing market-data documentation also identifies corporate-action semantics
as unresolved for this type of historical evaluation.

## Authoritative provider evidence

Alpaca's official corporate-actions documentation identifies the endpoint as
`GET https://data.alpaca.markets/v1/corporate-actions` and describes date
filters based on corporate-action processing dates. It also warns that Alpaca
does not guarantee corporate-action creation time and that actions may be
delayed before becoming available through the API:

- https://docs.alpaca.markets/us/v1.1/reference/corporateactions-1
- https://docs.alpaca.markets/us/reference/corporateactions-1

That limitation prevents the current API response from proving that a
corporate-action record was knowable at each historical strategy decision
timestamp. A current response keyed only by effective/process date is not
accepted as point-in-time evidence under the frozen contract.

## Consequence

The frozen raw-price indicator and return contract cannot be executed
scientifically without either:

1. an already-authorised and auditable point-in-time corporate-action dataset
   satisfying the manifest; or
2. a separately reviewed, pre-outcome contract that changes the strategy's
   corporate-action/knowability treatment.

No fallback provider, retrospective adjusted series, silent dividend omission,
or post-result repair is permitted. A bounded metadata-only Alpaca REST query
and a bounded Alpaca SSE replay test were performed during VAL-03A; neither
established the required historical point-in-time provenance.

## Current governance

- `VAL-02C = COMPLETE / GO`
- `VAL-03 = NOT STARTED / HARD-STOPPED: DATASET_BLOCKED — CORPORATE ACTION PROVENANCE`
- `FORWARD PAPER = NOT STARTED`
- `PHASE 13 = NOT STARTED`
- `2025 HOLDOUT = SEALED`
- `TRADING EDGE = NOT DEMONSTRATED`

This blocker is not a strategy-performance classification. No candidate
performance result exists from this VAL-03 attempt.

VAL-03A continued this hard stop and did not create a v1.4 manifest.

VAL-03B is the authorized continuation. It implemented and tested a v1.4
adjusted-bar contract, but its Alpaca preflight found that all requested
families begin on 2016-01-04 rather than the required 2015-01-01 warm-up.
VAL-03 therefore remains blocked before performance evaluation.

## VAL-03C continuation

The adjusted-bar contract is accepted; the standalone 2015 warm-up condition
was obsolete because 2015 was initialization-only. VAL-03C replaces it with a
per-instrument 200-valid-synchronized-session initialization boundary, without
changing the 2021–2022 validation, 2023–2024 formal OOS, sample floors or sealed
2025 holdout. The v1.5 manifest and readiness artifact record the correction.
VAL-03 performance remains not started.
