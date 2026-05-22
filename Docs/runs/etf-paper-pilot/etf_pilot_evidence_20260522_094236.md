# ETF Paper Pilot Evidence

- status: blocked
- generated: 2026-05-22T08:42:36.3215398Z
- api_base: http://localhost:8081
- pilot_symbol: SPY
- excluded_symbol: TQQQ

## Checks
- [WARN] auth/status: unable to determine auth mode: The underlying connection was closed: The connection was closed unexpectedly.
- [FAIL] etf/catalog: The underlying connection was closed: The connection was closed unexpectedly.
- [FAIL] etf/pilot-status: The underlying connection was closed: The connection was closed unexpectedly.
- [FAIL] etf/testing-readiness: The underlying connection was closed: The connection was closed unexpectedly.
- [FAIL] readiness/etf-section: etfPhase1Readiness missing
- [WARN] signoff/automated-validation: not provided
- [PASS] signoff/operator-uat: provided
- [WARN] signoff/paper-pilot: not provided
- [WARN] signoff/engineering: not provided
- [WARN] signoff/operations: not provided
- [WARN] signoff/trading-risk: not provided

## Sign-Off Environment

Set these only after this report and related UAT evidence are reviewed:

```powershell
$env:ETF_PHASE1_AUTOMATED_VALIDATION="passed"
$env:ETF_PHASE1_OPERATOR_UAT="passed"
$env:ETF_PHASE1_PAPER_PILOT_SIGNOFF="passed"
$env:ETF_PHASE1_ENGINEERING_SIGNOFF="true"
$env:ETF_PHASE1_OPERATIONS_SIGNOFF="true"
$env:ETF_PHASE1_TRADING_RISK_SIGNOFF="true"
```
