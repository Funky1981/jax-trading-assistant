# Golden Event Cases

## Purpose

Golden event cases are executable examples of how Jax must behave.

They prevent drift.

If Jax cannot pass these cases, the decision system is not safe enough.

## Golden case format

Each case has:

```text
input.json
expected.json
```

Expected output must include:

- decision
- primary reason
- allowed actions
- forbidden actions
- review windows

## Required first case

### FTSE oil labour conflict

Input:

```text
FTSE falls because oil slumps, UK labour data is strong, and BoE decision is pending.
```

Expected:

```text
NO_TRADE
```

Reason:

```text
Conflicting macro drivers and no clean asset-specific edge.
```

Forbidden:

```text
execute_trade
create_live_order
auto_approve
```

Allowed:

```text
store_event
monitor
review_later
```

## Minimum 25-case set

1. FTSE down because oil falls, UK labour strong, BoE pending → NO_TRADE
2. Company beats earnings but cuts guidance → NO_TRADE or WATCH
3. Company beats earnings, raises guidance, volume confirms, sector strong → TRADE_CANDIDATE
4. Oil spikes after confirmed supply disruption, energy equities lag → WATCH or TRADE_CANDIDATE
5. Central bank decision tomorrow, market mixed → NO_TRADE
6. Stock gaps up 18% on rumour only → NO_TRADE
7. Stock pulls back to support after confirmed earnings beat → WATCH or TRADE_CANDIDATE
8. ETF breaks out but volume weak → WATCH
9. Index falls due to one heavyweight sector → NO_TRADE or WATCH
10. Strong macro data but currency/rates reaction contradicts → NO_TRADE
11. Profit warning from single company with sector peers unaffected → WATCH
12. Regulatory approval with clear revenue impact → TRADE_CANDIDATE
13. Regulatory headline with unclear impact → NO_TRADE
14. Merger rumour from weak source → NO_TRADE
15. Confirmed takeover offer → WATCH, not chase
16. Commodity fall hurts sector but stock already oversold → WATCH
17. Broad market risk-off day, no asset-specific catalyst → NO_TRADE
18. Earnings beat already fully priced in → NO_TRADE
19. High-quality setup but poor risk/reward → REJECTED_BY_RISK
20. Good catalyst but earnings tomorrow → WATCH or NO_TRADE
21. Good setup but spread too wide → REJECTED_BY_RISK
22. Good setup but stop cannot be defined → NO_TRADE
23. Good setup but existing correlated exposure too high → REJECTED_BY_RISK
24. Setup works in backtest but fails out-of-sample → RESEARCH_ONLY
25. Setup has promising backtest and paper evidence → PAPER_READY, not live

## Golden test rules

1. Tests must be deterministic.
2. No LLM call is required for golden pass/fail.
3. Volatile fields must be ignored or normalised.
4. FTSE/oil/labour must always reject.
5. NO_TRADE must be common.
6. Test failure blocks merge.
