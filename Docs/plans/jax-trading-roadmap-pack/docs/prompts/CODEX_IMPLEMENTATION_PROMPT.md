# Codex Implementation Prompt — Jax Roadmap Pack

You are working on `Funky1981/jax-trading-assistant`.

Jax is a production-grade trading decision system, not a magic AI trading bot.

Target flow:

```text
market/news/event data
→ structured signal extraction
→ evidence scoring
→ trust gates
→ risk engine
→ human approval
→ execution adapter
→ journal/memory
→ post-trade review
→ strategy improvement
```

## Current Priority

Implement the beginner-safe roadmap foundations:

1. Structured trade candidate model.
2. Risk and slippage calculation.
3. Trust gate / strict gatekeeper.
4. Paper approval loop.
5. Journal and review records.

## Non-Negotiable Rules

- Do not add autonomous live trading.
- Do not enable leverage by default.
- Do not place live broker orders without explicit human approval.
- Do not allow a trade candidate to pass without entry, stop, target, invalidation, position size, and max loss.
- Do not calculate risk using floats for money.
- Do not hide rejected trades; rejected trades are valuable training data.

## Required Domain Objects

Create or align existing models for:

- TradeCandidate
- EvidenceItem
- RiskPlan
- GateDecision
- HumanApproval
- PaperOrder
- ExecutionRecord
- TradeJournalEntry
- TradeReview
- StrategyHealth

## Required Risk Calculation

```text
risk_budget = account_size * risk_percent
stop_distance = abs(entry_price - stop_price)
realistic_risk_per_unit = stop_distance + slippage_allowance
position_size = floor(risk_budget / realistic_risk_per_unit)
max_slippage_adjusted_loss = position_size * realistic_risk_per_unit
```

## Hard Reject Conditions

Reject candidate if:

- catalyst is missing or unclear
- entry price is missing
- stop price is missing
- target price is missing
- invalidation point is missing
- slippage allowance is missing
- position size cannot be calculated
- max loss exceeds configured risk budget
- leverage is requested while leverage is disabled
- data is stale
- human approval is missing for paper/live execution

## Implementation Shape

Prefer small, reviewable changes.

Suggested folders:

```text
internal/domain/trading/
internal/domain/risk/
internal/domain/gates/
internal/domain/journal/
internal/application/candidates/
internal/application/approval/
internal/application/papertrading/
internal/application/review/
docs/roadmap/
docs/risk/
docs/templates/
```

## Tests Required

Add tests for:

- position sizing
- slippage-adjusted risk
- rejecting missing stop loss
- rejecting missing invalidation point
- rejecting leverage while disabled
- rejecting max loss above risk budget
- approving a valid paper candidate
- journal entry creation after simulated trade close

## Acceptance Criteria

The system should be able to:

1. Create a trade candidate.
2. Attach evidence.
3. Calculate risk with slippage buffer.
4. Reject unsafe candidates.
5. Move safe candidates to human approval.
6. Simulate paper execution.
7. Record result and review.
8. Produce a clear explanation for every pass/fail decision.
