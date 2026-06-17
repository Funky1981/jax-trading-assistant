# 00 — Redesign Branch Review and Required Fixes

## Reviewed Branch

Treat `work` as the current redesign branch unless a separate branch is later created.

## Current Architecture Observed

The branch uses the correct high-level split:

```text
cmd/trader      = deterministic trading runtime + frontend API
cmd/research    = research, backfill, event study, memory tools
ib-bridge       = market/broker connectivity boundary
agent0-service  = assistant/planning boundary
frontend        = React dashboard
Postgres        = source of truth
```

This is the right foundation. Do not collapse research into trader or trader into research.

## Existing Good Work To Preserve

- ETF-only phase-one policy direction.
- Candidate and approval workflow.
- Event ingestion/backfill routes.
- Event-study schema and priced-in scoring foundation.
- Evidence bundle concept.
- Mobile approval producer direction.
- Candidate evidence page direction.
- Strategy mode/catalog direction.
- UAT paper-trading scripts and evidence folder convention.

## Blocker 1 — Event Study Bounds Are Time-Now Based

Current problem:

```go
from, to := studyBounds(windows)
```

`studyBounds` uses:

```go
now := time.Now().UTC()
return now.Add(-maxBefore), now.Add(maxAfter)
```

For historical event study, bounds must be derived from the earliest and latest selected event times, not the current clock.

### Required Fix

Change the event-study load flow to:

```text
1. Load selected events first, or load event min/max time first.
2. Compute bounds from event.EventTime across all selected events.
3. Expand by maxBefore/maxAfter from selected windows.
4. Load candles using those event-derived bounds.
```

### Acceptance Tests

- Event on `2024-01-10T14:30:00Z` with `+5d` window loads candles around January 2024, not current date.
- Two events with different dates load a combined envelope covering both.
- Test fails if `time.Now()` is used in event-study bounds.

## Blocker 2 — ETF Allowlist Duplication

Current code has a local `phaseOneETFs` map inside `cmd/research/backfill.go`.

This can drift from:

- instrument catalog
- ETF phase-one policy
- config defaults
- frontend universe
- strategy instances

### Required Fix

Create a central policy package, for example:

```text
internal/modules/etfuniverse
```

It should expose:

```go
func PhaseOne() Universe
func IsPhaseOneETF(symbol string) bool
func ValidatePhaseOne(symbols []string) ValidationResult
func PhaseOneSymbols() []string
```

Then replace all local allowlist maps with central calls.

### Acceptance Tests

- No duplicate `SPY`, `QQQ`, `TQQQ`, etc. allowlist maps outside the central package.
- `TQQQ`, inverse ETFs, volatility ETFs, and single-name stocks reject everywhere.
- Backfill, classifier, strategy generation, candidate creation, and manual order ticket use the same policy.

## Blocker 3 — Confounder Analysis Not Wired

Current event-study path builds evidence bundles with:

```go
buildResearchEvidenceBundle(event, symbol, wins, score, nil)
```

and writes event studies with:

```go
UpsertEventStudy(ctx, windowResults, scores, nil)
```

### Required Fix

Add a confounder finder before evidence bundle generation.

Minimum viable confounder rules:

```text
same_symbol_same_window
macro_event_same_day
rates_event_affects_equities
energy_event_affects_broad_market
market_wide_move_without_matching_theme
earnings_or_mega_cap_event_affects_sector_etf
```

### Acceptance Tests

- Two events affecting `QQQ` within a configured window produce a confounder record.
- Confounders appear in evidence bundle JSON.
- Candidate creation hard-rejects if a high-relevance confounder is unresolved.

## Blocker 4 — Hardcoded Guardrail Passes

Current evidence bundle guardrails hardcode:

```go
StaleQuotePass: true
PaperModePass: true
```

This is not production-ready.

### Required Fix

Evidence bundle generation must receive a real guardrail context:

```go
type GuardrailEvidence struct {
    AllowlistPass bool
    SpreadPass bool
    StaleQuotePass bool
    PaperModePass bool
    ApprovalRequired bool
    BrokerConnected bool
    MarketSession string
    QuoteTimestamp time.Time
    GuardrailFailures []string
}
```

For historical/offline research where runtime quote checks are not available, mark fields as:

```text
not_applicable_for_historical_research
```

Do not mark them as pass.

### Acceptance Tests

- Missing quote cannot produce `stale_quote_pass=true`.
- Paper mode proof must come from runtime config/IB bridge status, not a default boolean.
- Historical research bundle is allowed to say `runtime_guardrails_not_evaluated`, but candidate creation must evaluate them before paper submission.

## Blocker 5 — Swing Mode Not First-Class

Current strategy outputs and evidence assume `flatten_by_close=true`.

Swing mode must be separate, not a loosened intraday mode.

### Required Fix

Add explicit trading horizon:

```text
research_only
intraday_paper
swing_research
swing_paper
```

Swing candidates need:

```text
hold_period_target_days
max_hold_days
overnight_risk_allowed
weekend_hold_allowed
revalidation_schedule
thesis_invalidators
calendar_risk_check
position_size_modifier
```

### Acceptance Tests

- Intraday candidate requires `flatten_by_close=true`.
- Swing candidate requires `overnight_risk_allowed=true` and daily revalidation.
- Swing candidate cannot be submitted unless `swing_paper_enabled=true` and paper mode is proven.

## Blocker 6 — UAT Evidence Still Incomplete

Do not mark the release production-ready until these are captured fresh on the branch:

- Broker paper mode proof.
- Candidate -> approval -> paper execution proof.
- No live trading proof.
- Post-trade memory/reflection proof.
- Full UAT run artifact.
- Evidence report artifact.
- Connectivity/readiness blockers cleared.
- `gofmt` blocker cleared.

## Required Before New Swing Work Is Accepted

```text
[ ] gofmt passes
[ ] services are healthy
[ ] ETF catalog endpoint passes
[ ] pilot-status endpoint passes
[ ] testing-readiness endpoint passes
[ ] current intraday Prompt 10 evidence is either closed or explicitly marked as superseded by the new swing UAT
[ ] no hardcoded runtime guardrail passes remain
```
