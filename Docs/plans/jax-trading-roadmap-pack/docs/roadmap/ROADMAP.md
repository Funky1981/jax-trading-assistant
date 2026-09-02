# Jax Trading Roadmap

> **HISTORICAL / SUPPORTING ROADMAP MATERIAL — NOT CURRENT AUTHORITY**
>
> This document preserves the earlier safety, candidate, evidence, trust-gate,
> risk and paper-trading sequence. It is useful background, but it has been
> superseded as the current phase sequence. Use `Docs/ROADMAP.md` for current
> development status and authorization. Do not infer live-trading authorization
> from this historical plan.

Jax is not a magic AI trader. Jax is a production-grade trading decision system.

Core flow:

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

## Phase 0 — Safety Baseline

Goal: Make it impossible for Jax to accidentally behave like an unsafe auto-trader.

Required outcomes:

- No live execution by default.
- All trades must have an approval state.
- Every candidate trade must be logged.
- Every rejected trade must store a reject reason.
- All risk calculations must use decimal-safe money handling.
- Add a global kill switch.
- Add environment separation: research, paper, live.

Exit gate:

```text
Jax cannot place a live trade unless live mode, broker config, human approval, risk checks, and kill switch state all allow it.
```

## Phase 1 — Structured Trade Candidate Model

Goal: Stop vague AI opinions becoming trades.

A trade candidate must include:

- Asset/instrument.
- Direction: long/short/avoid.
- Setup type.
- Catalyst.
- Evidence summary.
- Entry price.
- Stop price.
- Target price.
- Risk/reward estimate.
- Confidence score.
- Invalidation point.
- Slippage allowance.
- Position size.
- Max expected loss.
- Human-readable explanation.

Exit gate:

```text
No candidate can enter approval unless entry, stop, target, reason, invalidation, and risk are known.
```

## Phase 2 — Evidence Scoring

Goal: Jax should score evidence, not guess direction.

Evidence categories:

- News/catalyst strength.
- Market reaction.
- Volume/liquidity confirmation.
- Sentiment/analyst context.
- Technical location.
- Macro/sector alignment.
- Contradictory evidence.
- Data quality.

Each evidence item should be stored as structured data with source, timestamp, confidence, and impact.

Exit gate:

```text
Every trade candidate has an evidence score and visible supporting/contradicting evidence.
```

## Phase 3 — Trust Gates / Strict Gatekeeper

Goal: Jax should reject most ideas.

Hard reject conditions:

- No clear catalyst.
- No stop price.
- No invalidation point.
- Risk/reward below configured minimum.
- Position size cannot be calculated.
- Data is stale or incomplete.
- Spread/slippage too high.
- Trade conflicts with max daily/weekly loss limits.
- Leverage requested during restricted phases.
- Human approval missing.

Exit gate:

```text
Jax can explain exactly why a trade passed or failed.
```

## Phase 4 — Risk Engine

Goal: Risk is calculated before trade approval, not after.

Risk engine must calculate:

- Account size.
- Max risk per trade.
- Entry price.
- Stop price.
- Stop distance.
- Slippage buffer.
- Realistic risk per unit.
- Position size.
- Max normal loss.
- Max slippage-adjusted loss.
- Daily loss exposure.
- Weekly loss exposure.
- Whether leverage is used.
- Margin required if leverage is used.

Beginner defaults:

```text
First live bank: £100–£250
Risk per trade: 1%
Leverage: disabled
Max trades per week: 1–3
Max daily loss: 1 losing trade
Max weekly loss: 3 losing trades
```

Exit gate:

```text
Jax refuses approval if the calculated loss is unknown or above the configured limit.
```

## Phase 5 — Research / Backtest Evidence

Goal: Validate ideas before paper trading.

Required outcomes:

- Replay historical events.
- Compare against buy-and-hold.
- Compare against do-nothing.
- Compare against a simple rule baseline.
- Track win rate, average win, average loss, expectancy, drawdown, and R multiple.
- Store failed assumptions.

Exit gate:

```text
A setup cannot move to paper trading unless it beats basic baselines and has acceptable drawdown.
```

## Phase 6 — Paper Approval Loop

Goal: Practise the full process without real money.

Required outcomes:

- Jax generates trade candidates.
- Human manually approves/rejects.
- Paper broker simulates orders.
- Stop loss and target behaviour are simulated.
- Slippage assumptions are recorded.
- Every trade has a journal entry.
- Every rejected trade is logged.

Exit gate:

```text
Jax has at least several months of paper-trade data and can show whether decisions have positive expectancy.
```

## Phase 7 — Shadow Mode

Goal: Let Jax observe live markets without placing trades.

Required outcomes:

- Jax watches live data.
- Jax produces candidates in real time.
- No real orders are sent.
- Compare suggested entries/exits against actual market movement.
- Record missed trades and rejected trades.

Exit gate:

```text
Jax performs sensibly in live market conditions while still unable to trade.
```

## Phase 8 — Tiny Live Activation

Goal: Test behaviour with real money, not make money quickly.

Rules:

```text
Starting bank: £100–£250
Risk per trade: 1%
Leverage: disabled
Approved setups only
Manual approval required
Max daily loss: 1 trade
Max weekly loss: 3 losing trades
```

Required outcomes:

- Real fills recorded.
- Slippage recorded.
- Broker reconciliation working.
- Human emotion/mistake log recorded.
- Compare real results to paper results.

Exit gate:

```text
Tiny live mode proves the system can be followed under real money without rule-breaking.
```

## Phase 9 — Execution Safety / Broker Reconciliation

Goal: Prevent live execution drift.

Required outcomes:

- Client order IDs.
- Order lifecycle state machine.
- Partial-fill handling.
- Cancel/readback verification.
- Broker position reconciliation.
- Broker cash reconciliation.
- Order retry rules.
- Emergency kill switch.
- Audit trail.

Exit gate:

```text
Jax always knows whether broker state matches internal state.
```

## Phase 10 — Journal Intelligence / Behaviour Review

Goal: Improve the operator and the strategy.

Required outcomes:

- Trade thesis.
- Screenshot/market context support if available.
- Jax recommendation.
- Human decision.
- Reason for override.
- Emotional state/mistake flag.
- Post-trade review.
- Strategy improvement note.

Exit gate:

```text
Every trade teaches Jax something useful or confirms that no change is needed.
```

## Phase 11 — Leverage Readiness

Goal: Only allow leverage when it solves a specific risk/capital problem.

Leverage remains disabled until:

- Setup has proven edge.
- Stop-loss behaviour is reliable.
- Slippage is measured.
- Fill quality is measured.
- Max drawdown is acceptable.
- Position sizing is stable.
- Live tiny-bank phase shows discipline.
- Broker reconciliation is reliable.
- Guaranteed/normal stop behaviour is understood.

Exit gate:

```text
Leverage can only be enabled per strategy, not globally, and only after explicit readiness review.
```

## Phase 12 — Scale-Up Gates

Goal: Increase size only when evidence justifies it.

Scale-up should be controlled by:

- Number of trades.
- Expectancy.
- Drawdown.
- Rule-following score.
- Slippage stability.
- Benchmark comparison.
- Strategy health.

Exit gate:

```text
Jax scales only after evidence, never after excitement.
```
