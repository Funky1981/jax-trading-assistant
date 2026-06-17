# 07 — Technical/Fundamental Backtesting and UAT

## Goal

Prove the technical and fundamental engines before trusting them.

No live trading from this layer.

## Backtest questions

```text
Does technical confirmation improve outcomes?
Does fundamental scoring filter weak events?
Do hard vetoes reduce bad trades?
Does waiting for confirmation reduce Fed fakeouts?
Does relative strength improve ETF choice?
Does confounder detection prevent misleading trades?
```

## Fixture scenarios

### Hot CPI bearish confirmation

```text
CPI hotter than expected
QQQ breaks pre-event low
TLT sells off
SPY weak
candidate allowed
```

### Hot CPI no confirmation

```text
CPI hotter than expected
QQQ initially drops then reclaims VWAP
TLT rallies
candidate rejected
```

### Strong jobs hawkish

```text
NFP beat
unemployment stable
wages firm
QQQ down
TLT down
candidate allowed or watch depending extension
```

### Fed whipsaw

```text
statement hawkish
first move down
Powell reverses tone
QQQ reclaims range
candidate rejected
```

### Mega-cap confounder

```text
CPI neutral
Nvidia crashes on separate headline
QQQ down
Jax must not blame CPI
candidate rejected or reassigned to different event
```

## Metrics

```text
candidate win rate
average R
median R
false positive rate
false rejection rate
watch-only outcome quality
hard veto usefulness
technical score correlation with outcome
fundamental score correlation with outcome
```

## UAT checklist

```text
[x] Hot CPI fixture creates bearish TA/FA alignment
[x] Cool CPI fixture creates bullish TA/FA alignment
[x] Fed whipsaw fixture rejects
[x] Confounder fixture rejects
[x] Missing data fixture rejects
[x] High score with no stop rejects
[x] LLM summary cannot override veto
[x] Candidate remains paper-only
[x] Human approval required
[x] No broker order created
```

## Completion evidence

Closed on 2026-06-17 against deterministic automated coverage:

- `internal/modules/macroevents/uat_test.go`
  - `TestMacroReactionUATDeterministicFixtures` covers hot CPI bearish alignment, cool CPI bullish alignment, Fed whipsaw rejection, confounder rejection, missing-candle rejection, and paper-only non-persisted candidate handling.
  - `TestMacroReactionUATCandidateCannotBecomeOrderWithoutSeparateApproval` covers human approval required and no broker order creation.
- `internal/modules/macroevents/candidate_test.go`
  - `TestGenerateCandidateMissingStopBlocksCandidate` covers high score/no-stop rejection.
- `internal/modules/macroevents/review_test.go`
  - `TestEvaluateMultiAnalystReviewLLMSummaryCannotOverrideVeto` covers LLM summary override prevention.

Verification command:

```powershell
go test ./internal/modules/macroevents -count=1
```

## Codex task

```text
Add deterministic TA/FA test fixtures and backtest harness.

Use fixture candles and fixture macro events first.
No paid providers required.
No broker writes.
```

## Acceptance criteria

```text
fixture tests pass
backtest output includes score vs outcome
bad data fails safely
no live execution path added
UAT checklist committed
```
