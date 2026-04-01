# Definition of Done: Paper-Trading / Backtesting Ready

Jax is finished for this phase only when every item below is true.

## A. Research truth path
- [ ] no-fake-data policy is implemented in code
- [ ] fake backtest paths are blocked from research truth path
- [ ] provenance is stored on runs and artifacts
- [ ] deterministic backtests run through current research runtime
- [ ] backtest results persist cleanly

## B. Data and strategy model
- [ ] strategy types metadata exists
- [ ] strategy instances CRUD exists
- [ ] enabled instances are loaded from DB/API model
- [ ] instance config is validated
- [ ] dataset / data source provenance is available on runs

## C. Always-on trade discovery
- [ ] trader runtime continuously scans enabled instances
- [ ] candidate trades are created
- [ ] duplicate candidate suppression exists
- [ ] blocked candidates are recorded with reasons

## D. Approval and paper execution
- [ ] approval queue exists
- [ ] approve / reject / snooze / reanalyze actions exist
- [ ] approved candidates become execution instructions
- [ ] paper fills/status are recorded
- [ ] flatten-by-close proof exists for same-day instances
- [ ] no paper execution bypasses approval

## E. Operator UX
- [ ] `/research` page exists and is wired
- [ ] `/analysis` page exists and is wired
- [ ] `/testing` page exists and is wired
- [ ] `/approvals` page exists and is wired
- [ ] `/assistant` page or panel exists and is wired

## F. AI and explanation
- [ ] AI influence is schema-logged
- [ ] AI decisions are replayable
- [ ] assistant cannot execute trades
- [ ] assistant can explain blockers and scenarios

## G. Trust
- [ ] data quality gate exists
- [ ] deterministic replay gate exists
- [ ] provenance gate exists
- [ ] approval flow gate exists
- [ ] execution integration gate exists
- [ ] flatten proof gate exists
- [ ] paper P/L reconciliation gate exists

## H. Sign-off
- [ ] 20+ paper sessions with no critical unresolved failures
- [ ] scoreboard shows no P0 missing blockers
