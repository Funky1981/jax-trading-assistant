# Five Same-Day Strategies (Flat by Close)

All strategies:
- operate intraday
- have a defined entry window
- MUST flatten before close
- do not require constant monitoring

## Strategy 1: Same-Day Earnings Drift (Safest)
**Trigger:** earnings pre-market (or prior close)  
**Entry:** 10:00–11:00 ET  
**Exit:** flatten 15:30 ET  

**V1 direction rule**
- Long if earnings surprise positive AND guidance positive
- Short if earnings surprise negative AND guidance negative
- Otherwise: no trade

**V1 execution**
- Entry: limit near mid (or next bar open in backtest)
- Stop: ATR-based or morning low/high
- Target: fixed R multiple (e.g. 1.5R)

**Requires**
- earnings event source (`market.get_earnings`)
- intraday candles

## Strategy 2: Same-Day News Repricing (Low–Medium)
**Trigger:** material corporate news (not rumors)  
**Entry:** after consolidation (range tightens, volume cools)  
**Exit:** flatten 15:30 ET  

**V1 rule**
- Materiality filter must be deterministic (category allow-list)
- Enter only after N bars of consolidation after initial impulse

**Requires**
- structured news events to tickers
- consolidation detector on 1m/5m

## Strategy 3: Opening Range → Close Continuation (Medium)
**Trigger:** opening range breakout holds  
**Entry:** after first 30 minutes, breakout + hold for N bars  
**Exit:** flatten 15:30 ET  

**V1 rule**
- Opening range = first 30 mins
- Breakout must hold for 3 bars
- Volume must be > X * average

**Requires**
- opening range module
- volume normalization

## Strategy 4: Same-Day Panic Mean Reversion (Medium–High)
**Trigger:** intraday drop beyond threshold percentile  
**Entry:** late morning/early afternoon when selling exhausts  
**Exit:** flatten 15:30 ET  

**V1 rule**
- Drop > P95 of historical intraday moves for that symbol
- No additional negative event since initial drop
- Enter when price stabilizes (range contraction) near VWAP

**Requires**
- intraday volatility baseline
- event “silence” check

## Strategy 5: Same-Day Index Flow Continuation (Riskiest)
**Scope:** SPY/QQQ only (initially)  
**Trigger:** strong trend day + pullback  
**Entry:** pullback after trend confirmation  
**Exit:** flatten 15:30 ET  

**V1 rule**
- Trend day classifier: strong directional move + range expansion
- Entry on pullback to VWAP/EMA band
- Strict max trades/day = 1

**Requires**
- trend regime detector
- strict risk caps and kill switches
