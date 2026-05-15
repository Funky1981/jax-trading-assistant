# ETF Paper Pilot Evidence

- status: awaiting_signoff
- generated: 2026-05-14T16:47:34.1674314Z
- api_base: http://localhost:8081
- pilot_symbol: SPY
- excluded_symbol: TQQQ

## Checks
- [PASS] auth/login: auth enabled; bootstrap login succeeded
- [PASS] etf/catalog: http://localhost:8081/api/v1/instruments/etfs
- [PASS] etf/pilot-status: http://localhost:8081/api/v1/trading/pilot-status
- [PASS] etf/testing-readiness: http://localhost:8081/api/v1/testing/readiness
- [PASS] catalog/version: phase1-2026-05-13
- [PASS] catalog/pilot-symbol: SPY present
- [PASS] catalog/excluded-symbol: TQQQ has exclusions
- [PASS] pilot/etf-phase1: enabled
- [PASS] pilot/workflow: candidate_approval_only
- [PASS] readiness/catalog-loaded: catalog loaded
- [PASS] readiness/status: not_ready
- [WARN] signoff/automated-validation: not provided
- [WARN] signoff/operator-uat: not provided
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
