# Paper Review UX

## Purpose

Make persisted paper tickets easy to review without turning the page into an execution surface. The panel should help a beginner understand the trade idea, evidence, risk, and review state before recording a safe review action.

## What Changed

- Replaced the raw paper-ticket row with grouped review cards.
- Added plain-language copy: "Paper review only", "Trade idea", "Why this exists", "Risk summary", "Entry / Stop / Target", "Worst planned loss with slippage", "Reward:risk", "Review status", and "Internal notes".
- Displayed saved internal review notes from the paper-ticket read model.
- Added a safe internal-note action alongside mark-reviewed and cancel actions.
- Added friendlier loading, error, empty, reviewed, and cancelled states.

## UI Sections

- Trade idea: symbol, direction, setup type, review status, and catalyst summary.
- Evidence: evidence status, gate status, approval status, and warnings.
- Risk summary: entry, stop, target, position size, max normal loss, slippage-adjusted worst planned loss, reward:risk, risk status, and blockers.
- Review actions: mark reviewed, cancel paper ticket, and add internal note.
- Notes: saved internal notes or an empty notes message.

## Safe Wording

- Paper review only
- Trade idea
- Why this exists
- Evidence
- Risk summary
- Entry / Stop / Target
- Worst planned loss with slippage
- Reward:risk
- Review status
- Internal notes

## Forbidden Wording/Actions

Do not add any paper-ticket UI action or label that suggests:

- execute
- place order
- broker
- live
- leverage
- trade now
- auto trade

Do not expose execution-control DTO fields in the UI, including broker execution, instruction-created, live-trading, or leverage flags.

## Screenshot Checklist / Manual Test Checklist

- Open the approvals page with at least one created paper ticket.
- Confirm every paper-ticket card says "Paper review only".
- Confirm the card shows symbol, direction, setup type, catalyst summary, evidence status, gate status, risk status, approval status, entry, stop, target, position size, max normal loss, slippage-adjusted worst planned loss, reward:risk, warnings, and notes.
- Confirm reviewed and cancelled tickets are visually/status distinguishable.
- Confirm only safe actions are visible: Mark reviewed, Cancel paper ticket, Add internal note.
- Confirm no execution, place-order, broker, live, leverage, trade-now, or auto-trade action appears in the paper-ticket panel.
- Confirm the empty state says no paper tickets need review.
- Confirm loading and backend error states are readable.

## Deferred UX Improvements

- Add filters for created, reviewed, and cancelled paper tickets.
- Add a compact/mobile-first card variant after real operator feedback.
- Add better plain-English translations for backend status codes.
- Add a read-only evidence detail drawer for paper tickets without adding any execution actions.
