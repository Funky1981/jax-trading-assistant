# 08 — Backtesting and UAT

## Goal

Prove the engine before trusting it.

No live trading until there is a long paper-trading proof period.

## Backtest scope

Phase 1 backtests:

```text
US CPI
US NFP/jobs
FOMC rate decision
FOMC statement days
```

ETFs:

```text
QQQ
SPY
TLT
IWM
XLK
SMH
SOXX
XLF
GLD
```

## Backtest windows

```text
event day intraday
1 hour post-release
end of session
next session open
next session close
5 trading days
```

## Metrics

```text
win rate
average R
median R
max drawdown
false confirmation rate
missed winner rate
late/chase rejection quality
confounder rejection quality
priced-in rejection quality
```

## UAT scenarios

### Scenario 1 — Hot CPI

Expected:

```text
Jax classifies hawkish_rates
maps QQQ/SPY/TLT
checks chart reaction
requires QQQ/TLT confirmation
creates candidate only if not extended
```

### Scenario 2 — Cool CPI

Expected:

```text
Jax classifies dovish_rates
maps QQQ/SPY/TLT
checks chart reaction
candidate long QQQ/TLT only if confirmed
```

### Scenario 3 — Strong jobs

Expected:

```text
Jax classifies hawkish_rates/growth_strong
checks unemployment/wages if available
maps QQQ/SPY/TLT/IWM
requires chart confirmation
```

### Scenario 4 — Fed press conference

Expected:

```text
Jax delays confirmation window
does not trade first fake move
requires post-Powell direction confirmation
```

### Scenario 5 — Noisy/unclear event

Expected:

```text
Jax creates no candidate
stores evidence
explains no-trade
```

## Codex task

```text
Add backtest and UAT fixtures for macro reaction engine.

Use deterministic fixture candles/events first.
Do not require paid providers for tests.
Add live-provider tests separately and keep them optional.
```

## Tests

```text
fixture hot CPI confirms bearish QQQ
fixture cool CPI confirms bullish QQQ
fixture whipsaw Fed event rejects
fixture already-priced-in event rejects
fixture missing candles rejects
fixture candidate cannot become order without approval
```

## Acceptance criteria

```text
all deterministic tests pass
UAT checklist documented
paper-only run completed
no live broker writes during tests
results stored for review
```

## Long proof rule

Do not enable live trading until:

```text
minimum 6 months paper trading
minimum 100 macro event evaluations
documented positive expectancy
operator review shows no major safety failures
```
