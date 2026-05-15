# Copilot Prompt — Implement ETF News Strategy Foundation

Use this prompt after reviewing the strategy pack.

```text
Implement the foundation needed to support the ETF news strategy pack for paper trading only.

Do not enable live trading.

Use the strategy docs in Docs/plans or the provided ETF news strategy pack as requirements.

Hard requirements:
1. Add or verify a first-class ETF instrument model.
2. Add a phase-1 ETF allowlist:
   - SPY
   - QQQ
   - DIA
   - IWM
   - XLK
   - XLF
   - XLE
   - SMH
   - SOXX
   - TLT
   - GLD
3. Explicitly block:
   - leveraged ETFs
   - inverse ETFs
   - volatility ETFs
   - options
4. Add pre-trade gates:
   - allowlist
   - paper mode
   - quote freshness
   - spread
   - session
   - stop-loss
   - take-profit or flatten-by-close
   - human approval
5. Add strategy definitions for:
   - ETF_NEWS_001_MARKET_PANIC_REVERSAL
   - ETF_NEWS_002_SECTOR_MOMENTUM
   - ETF_NEWS_003_RATES_BONDS_ROTATION
6. Do not submit orders directly from strategy code.
7. Route all candidates through the existing approval flow.
8. Add tests for rejected unsafe paths.
9. Add UAT checklist for SPY and QQQ paper trading.

Output:
- files changed
- summary
- tests run
- remaining blockers
```
