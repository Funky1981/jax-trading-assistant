# Phase 3 — Candidate Review and Hypothetical Outcomes

## Goal

Make the existing candidate, approval, paper-ticket and checkpoint records understandable without implying a real trade occurred.

## Candidate Review

Rename the user-facing page:

**Candidate Review**

## Required sections

### A. Why Jax considered this

Show:

- triggering evidence
- headline and source
- mapping reason
- evidence score
- limitations

### B. What supported or blocked it

Show:

- evidence score
- trust-gate result
- risk-review result
- chart confirmation
- sentiment, where genuinely available
- block reason
- expiry state

### C. Human decision

Show:

- approval state
- approver
- approval time
- decision reason
- explicit human/manual status
- no automatic approval language

### D. Hypothetical paper plan

Show only persisted values:

- hypothetical entry
- stop
- target
- quantity
- notional
- planned risk
- planned reward
- reward/risk
- leverage
- account-equity assumption
- paper-ticket ID
- paper-ticket state

Display a persistent warning:

> Hypothetical paper plan — not an order, fill or open position.

### E. Remove invented sizing

Remove the hardcoded `$100` risk-budget fallback.

When no persisted sizing evidence exists, show:

`No persisted sizing evidence available.`

Do not derive runtime financial values in the React page from hidden assumptions.

### F. Language correction

Remove or rewrite copy that tells the user to:

- place the trade in IB/TWS
- open an order ticket
- treat a candidate as executable

Use:

`Review the evidence and paper assumptions.`

## Outcomes

Create a dedicated Outcomes page or a clearly separated Candidate Review tab.

## Outcome checkpoint design

Each checkpoint card shows:

- duration: 1 hour, 1 day, 1 week
- tracking start
- due time
- status
- checkpoint price
- observation timestamp
- market-data source
- hypothetical return
- hypothetical P&L
- MFE
- MAE
- stop touched
- target touched
- first stop touch
- first target touch
- same-candle ambiguity
- missing-data reason
- created and updated timestamps

## Required labels

- `HYPOTHETICAL — NOT A FILL`
- `PENDING — NOT DUE`
- `COMPLETED`
- `MISSING MARKET DATA`
- `AMBIGUOUS SAME CANDLE`
- `TARGET TOUCHED`
- `STOP TOUCHED`

## Data semantics

The UI must distinguish:

- hypothetical entry versus fill
- checkpoint price versus execution price
- hypothetical P&L versus realised P&L
- completed versus pending
- missing data versus zero return
- expired candidate versus invalid retrospective outcome
- selected-journey records versus unrelated historical records

## Backend read models

Prefer read-only endpoints for:

- candidate review
- approval detail
- paper-ticket detail
- outcome checkpoints
- selected-journey execution-side counts

Do not expose mutation actions in these read models.

Do not make the frontend parse flexible metadata keys to determine financial meaning.

## Navigation

- Candidates list links to Candidate Review.
- Candidate Review links to Outcomes.
- Outcomes links back to Candidate Review.
- Evidence Inbox links to Candidate Review only when a candidate exists.
- Home shows recent hypothetical outcomes.

## Tests

- Existing approved QQQ candidate displays.
- Existing human approval displays.
- Existing paper ticket displays.
- Hypothetical entry is never called a fill.
- The `$100` fallback no longer exists.
- Missing sizing is explicit.
- Expiry displays but does not hide retrospective outcomes.
- Existing 1-hour and 1-day checkpoints display.
- One-week state displays accurately.
- Missing data is not displayed as zero.
- Same-candle ambiguity is explicit.
- Genuine market-data source is visible.
- No actual portfolio value is inferred.
- No order or execution button is added.
- Historical unrelated execution records are not attributed to the paper ticket.

## Runtime proof

Using existing local records, show:

- the approved QQQ candidate
- the human decision
- the paper-ticket ID
- the hypothetical assumptions
- completed 1-hour and 1-day results
- current 1-week status
- zero execution-side records attributable to this outcome tracker

## Exit criteria

Phase 3 is complete when a beginner can clearly explain that the displayed results are a retrospective hypothetical evaluation and not a real trade.
