# 11 — Testing, UAT, and Evidence

## Goal

Finish with a production-ready, testable paper system.

Do not rely on manual belief. The system must prove:

```text
research trigger -> event study -> swing thesis -> candidate -> approval -> paper execution -> revalidation -> reflection
```

## Test Layers

### Unit Tests

Required packages:

```text
internal/modules/etfuniverse
internal/modules/tradingmodes
internal/modules/airesearch
internal/modules/guardrails
internal/modules/confounders
cmd/research
cmd/trader
libs/strategytypes
```

Unit test themes:

```text
ETF allowlist
horizon policy
swing windows
priced-in scoring
confounder detection
AI JSON validation
guardrail evaluation
candidate eligibility
```

### Integration Tests

Required:

```text
migration apply
research trigger persist
event classification persist
event study persist
confounder persist
research thesis persist
candidate persist
approval persist
guardrail persist
paper execution instruction persist
revalidation persist
reflection persist
```

### E2E/UAT

Create:

```text
scripts/uat-etf-swing-research.ps1
scripts/etf-swing-evidence-report.ps1
```

UAT flow:

```text
1. Start local stack.
2. Confirm services healthy.
3. Confirm broker paper mode.
4. Confirm ETF universe and policy.
5. Ingest one World Monitor-style research trigger.
6. Ingest one calendar-style macro event.
7. Generate event study with intraday and swing windows.
8. Generate priced-in score.
9. Generate confounder result.
10. Generate swing thesis.
11. Convert eligible thesis to paper candidate.
12. Block candidate if guardrails fail.
13. Approve candidate after guardrails pass.
14. Create paper execution instruction.
15. Record simulated/paper submission evidence.
16. Run daily revalidation check.
17. Close/simulate outcome.
18. Record reflection.
19. Generate evidence report.
```

## UAT Evidence Files

Each run should write:

```text
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/session.md
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/session.json
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/service-health.json
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/broker-paper-mode.json
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/research-trigger.json
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/event-study.json
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/evidence-bundle.json
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/guardrails.json
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/candidate.json
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/approval.json
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/paper-execution.json
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/revalidation.json
Docs/runs/etf-swing-research/YYYYMMDD_HHMMSS/reflection.json
```

## Current Blockers To Carry Forward

Before swing signoff, prove these are closed:

```text
gofmt blocker fixed
service connectivity fixed
ETF catalog endpoint healthy
pilot-status endpoint healthy
testing-readiness endpoint healthy
fresh broker paper-mode proof captured
fresh post-trade reflection captured
no hardcoded stale_quote_pass/paper_mode_pass remains
```

## Release Gate

Release is blocked unless:

```text
all tests pass
all UAT evidence exists
paper mode proof is fresh
live mode is blocked
reflection exists
operator signoff exists
engineering signoff exists
risk signoff exists
```

## Commands

Expected commands:

```powershell
.\scripts\go-verify.ps1 -Mode quick
.\scripts\uat-paper-trading.ps1 -Mode full
.\scripts\uat-etf-swing-research.ps1 -Mode full
.\scripts\etf-paper-pilot-evidence.ps1 -OperatorUATPassed
.\scripts\etf-swing-evidence-report.ps1 -OperatorUATPassed
```

## No-Go Conditions

- Any live execution path is reachable.
- Any non-ETF candidate is created.
- Any leveraged/inverse/volatility ETF candidate is created.
- Any candidate lacks evidence bundle.
- Any paper execution lacks approval.
- Swing candidate lacks daily revalidation.
- Broker paper mode not proven.
- Post-trade reflection missing.
- Hardcoded guardrail pass detected.
