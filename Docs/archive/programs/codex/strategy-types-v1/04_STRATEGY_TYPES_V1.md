# 04_IMPLEMENTATION: Five same-day strategy types (v1 logic)

Important:
- These are **baseline, testable** v1 implementations.
- They intentionally avoid magic/AI. They are rule-based.
- They require intraday candles; if you only have daily candles today, these types can still exist but should return a clear error until intraday candles are available.

Common input contract:
- timeframe: 1m (or 5m) candles for the session day
- session timezone: America/New_York
- entryWindow: provided by instance (not hardcoded in type)
- flattenTime: provided by instance

### Type 1: same_day_earnings_drift_v1
Parameters:
- entryDelayMins (int)
- minGapPct (float)
- minVolumeMultiple (float)
Logic:
- Wait entryDelayMins after open.
- If open gap magnitude >= minGapPct and volume multiple >= threshold, follow direction.
- Exit at flatten time or stop/target if implemented.

### Type 2: same_day_news_repricing_v1
Parameters:
- entryDelayMins (int)
- consolidationBars (int)
- maxRangePct (float)
Requires: `needsNews=true`
Logic:
- After news event timestamp, wait entryDelayMins.
- Detect consolidation: last N bars range <= maxRangePct.
- Breakout from consolidation -> enter in breakout direction.
- Exit at flatten.

### Type 3: opening_range_to_close_v1
Parameters:
- openingRangeMins (int)
- holdBars (int)
- minVolumeMultiple (float)
Logic:
- Build high/low for first openingRangeMins.
- Enter on break + hold for holdBars.
- Exit at flatten.

### Type 4: panic_reversion_v1
Parameters:
- dropPctThreshold (float)
- vwapDistancePct (float)
- minTimeAfterOpenMins (int)
Logic:
- If intraday drop from open exceeds threshold, look for exhaustion.
- Enter mean-reversion when price reclaims VWAP (or moves within vwapDistance).
- Exit at flatten.

### Type 5: index_flow_v1
Scope: SPY/QQQ only (enforced in Validate)
Parameters:
- trendStrengthPct (float)
- pullbackPct (float)
Logic:
- Confirm strong trend (range expansion + direction).
- Enter on pullback within pullbackPct.
- Exit at flatten.

Acceptance criteria:
- Each type compiles.
- Each type validates parameters.
- Each type returns either signals or a clear “missing required inputs” error.
