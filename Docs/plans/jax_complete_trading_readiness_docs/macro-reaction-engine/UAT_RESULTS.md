# Macro Reaction Engine UAT Results

Status: deterministic Phase 8 fixtures added.

Coverage:

- Hot CPI confirms bearish QQQ and creates an awaiting-human-approval paper candidate.
- Cool CPI confirms bullish QQQ and creates an awaiting-human-approval paper candidate.
- Fed whipsaw rejects candidate creation.
- Already-priced-in event rejects candidate creation.
- Missing candles produce insufficient evidence and no candidate.
- Paper candidate generation does not persist, promote, or create broker orders in deterministic UAT.

Validation command:

```text
go test ./internal/modules/macroevents -run TestMacroReactionUAT -count=1
```

Live trading remains blocked by policy. The long proof rule in `08_BACKTESTING_AND_UAT.md` still applies before any live enablement.
