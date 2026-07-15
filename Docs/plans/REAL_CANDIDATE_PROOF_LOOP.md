# Real Candidate Proof Loop

## Question

Can Jax produce one real reviewable paper candidate from real input data?

This loop calls the existing World Monitor promotion endpoint. It does not insert fixtures, approve a candidate, create an execution instruction, call a broker order endpoint, enable live trading, or permit leverage.

## Minimum services

- Postgres with migrations through version 46.
- `jax-trader` API on `http://localhost:8081`.
- IB Bridge market ingester running long enough to put usable rows in `quotes` and at least 20 candles for a promoted ETF in `candles`.
- Jax World News Monitor integration with at least one normalized inbox row at confidence `>= 0.55` and a mapped ETF.

IB Bridge is an input adapter only in this proof. Broker execution services are not required and should remain disabled.

## Run

```powershell
$env:DATABASE_URL="postgres://jax:jax@localhost:5432/jax?sslmode=disable"
./scripts/migrate.ps1 version
./scripts/run-real-candidate-proof.ps1
```

If the API is elsewhere:

```powershell
./scripts/run-real-candidate-proof.ps1 -ApiBase http://localhost:8081 -OutputDir Docs/runs/real-candidate-proof
```

The script requires `psql` on `PATH`. When API authentication is enabled it reads `AUTH_BOOTSTRAP_USERNAME` and `AUTH_BOOTSTRAP_PASSWORD` from the environment or `.env`.

## Successful output

Success is a report with `status: candidate_produced`, one or more real World Monitor candidates, populated entry/stop/target, and either:

- `candidate_status: awaiting_approval`, which is a reviewable paper-only candidate awaiting evidence/risk/human review; or
- a paper ticket status, possible only after the existing independent human approval path has completed.

Safety must show zero execution instructions, zero unsafe tickets, and zero leveraged candidates. Reports are written as timestamped Markdown and JSON under `Docs/runs/real-candidate-proof/`.

## Expected failure output

The script exits non-zero with `status: blocked` when Postgres is unreachable, migration version 46 is not clean, quotes/candles are empty, no inbox row is promotable, the API promotion fails, or a safety invariant fails. It exits non-zero with `status: no_candidate` when processing completes but nothing reviewable is produced. A promoted row can legitimately produce `candidate_status: blocked`; its reject reason explains missing chart confirmation or structural/gate readiness. `paper_ticket_status: not_created` is expected before human approval and must not be bypassed by this harness.

Common fixes:

- No quotes: start/fix the IB Bridge market ingester and confirm a positive quote for the mapped ETF.
- No/insufficient candles: backfill real IB candles; promotion requires at least 20 recent closes for the mapped ETF.
- No promotable inbox rows: run the real World Monitor ingestion; ensure normalization, ETF mapping, and confidence threshold pass.
- No strategy instance: enable an existing ETF paper strategy that contains the mapped symbol.
- Candidate blocked as structurally incomplete: complete the existing promoter's structured candidate/evidence/risk integration; do not fabricate proof data or bypass gates.

## Collect 10 real candidates

Run the proof after ten distinct real, normalized World Monitor events arrive. The existing same-session strategy/symbol duplicate guard may skip repeats, so collect across distinct mapped ETFs or sessions. Keep every generated JSON/Markdown report, including blocked and skipped results. Do not change thresholds simply to reach ten.

Maintain a small outcome log keyed by `candidate_id` with the quote observed at:

- 1 hour after creation;
- 1 day after creation;
- 1 week after creation.

For each horizon record timestamp, market-data source, observed price, return from entry, whether stop or target was crossed, and any data gap. Use real stored quotes/candles and never rewrite the original proof report.

## What this proves

- Real World Monitor inbox data can reach the existing promoter.
- The promoter consumes stored real quotes and candles.
- Jax can persist and expose a paper-only candidate for review, including block reasons.
- Existing paper-ticket records, when present after human approval, remain review-only and safety constrained.

## What this does not prove

- Profitability, statistical edge, execution quality, or live readiness.
- That every event becomes a candidate.
- Broker connectivity for orders, order placement, fills, or portfolio accounting.
- That human approval may be skipped.
- That a paper ticket should exist before evidence, gates, risk review, and explicit human approval pass.
- Any support for live trading, leverage, or automatic execution.
