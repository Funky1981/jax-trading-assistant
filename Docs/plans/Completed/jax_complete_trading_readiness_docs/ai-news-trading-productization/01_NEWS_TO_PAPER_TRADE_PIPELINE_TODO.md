# News to Paper Trade Pipeline Todo

## Goal

Make the Jax Monitor -> Jax Trader path honest and beginner-usable. News must not become an actionable paper-trade approval until Jax has checked market data, chart confirmation, risk levels, and execution readiness. Every blocked step must be visible in the UI with plain-language reasons.

## Pipeline

- [x] Receive Jax World News Monitor trigger through `POST /api/v1/research/events/world-monitor`.
- [x] Validate trigger freshness, source URLs, confidence, ETF mapping, and forbidden direct trade-instruction language.
- [x] Show News Monitor connection and last received trigger in the UI.
- [x] Convert accepted trigger into a normalized research event.
- [x] Run scanner/promoter only in paper runtime with live trading disabled.
- [x] Check scanner symbol scope and minimum confidence.
- [x] Check latest quote exists before creating any candidate.
- [x] Check recent candles/chart confirmation before an opportunity can enter the approval queue.
- [x] Block unconfirmed news as `Needs chart confirmation`, not as an actionable approval.
- [x] Store chart confirmation evidence in candidate metadata.
- [x] Show chart-confirmation status on AI Trading opportunity cards.
- [x] Show the post-approval lifecycle clearly: approved -> paper instruction -> broker blocked/submitted/filled.
- [x] Disable misleading `Review order` paths when an approved ETF candidate cannot prefill/manual-submit.
- [x] Add a single operator test script that verifies: monitor trigger, chart gate, approval, execution instruction, and broker readiness state.

## Done Criteria

- News-only items never appear as approvable trades without chart confirmation.
- If chart data is missing or bearish, AI Trading shows the item as blocked/watch-only with the exact reason.
- If chart confirmation passes, the item can enter Approvals.
- Approving a candidate shows where it went next and why broker submission is blocked/submitted/filled.
- The beginner UI clearly separates simulated/test data, advisory AI suggestions, news-backed ideas, and broker-ready paper orders.
