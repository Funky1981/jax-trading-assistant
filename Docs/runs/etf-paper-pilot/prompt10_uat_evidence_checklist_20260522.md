# Prompt 10 UAT Evidence Checklist - 2026-05-22

Use this checklist to capture signoff-quality evidence for Prompt 10 acceptance.

## Session Metadata

- [ ] Operator name recorded
- [x] Branch and commit hash recorded (branch: work, commit: ade1b37)
- [x] Runtime mode confirmed as paper (requested in script execution)
- [ ] Session start/end timestamps recorded
- [x] Evidence folder created under Docs/runs/etf-paper-pilot

## 1. ETF-Only Defaults

- [ ] Capture config/runtime proof showing ETF-only universe
- [ ] Capture rejection proof for non-ETF symbol path
- [ ] Save screenshot or log excerpt with timestamp

## 2. Event Ingestion

- [ ] Run ingestion/backfill command and save terminal output
- [ ] Confirm events persisted (count or sample rows)
- [ ] Archive resulting artifact/report files

## 3. Event Study Generation

- [ ] Generate event-study output for at least one ETF event
- [ ] Save event-study artifact path(s)
- [ ] Confirm output includes expected windows/metrics

## 4. Priced-In Scoring

- [ ] Capture priced-in score and explanation in evidence output
- [ ] Confirm score is associated with the tested candidate
- [ ] Save artifact path and screenshot/log proof

## 5. Evidence Bundle

- [ ] Generate candidate evidence bundle
- [ ] Confirm bundle includes news context and confounders
- [ ] Archive bundle path in this checklist

## 6. Candidate Creation

- [ ] Confirm candidate is created from ETF workflow
- [ ] Save candidate id
- [ ] Capture create-time payload/log evidence

## 7. Approval

- [ ] Record human decision action and actor
- [ ] Capture approval artifact (id, decided_at, notes)
- [ ] Confirm audit trail entry exists

## 8. Paper Execution Instruction

- [ ] Confirm execution instruction row created after approval
- [ ] Capture instruction id and status
- [ ] Attach evidence showing approved -> instruction linkage

## 9. Broker Paper Mode (Required Fresh Proof)

- [ ] Capture proof broker is connected in paper mode
- [ ] Capture one full candidate -> approval -> paper submission flow
- [ ] Save broker-facing confirmation evidence (logs/screenshots)

Status note (2026-05-22): current run is blocked by connectivity/readiness failures; no fresh broker paper-mode proof captured yet.

## 10. No Live Trading

- [ ] Capture explicit proof live trading path is blocked
- [ ] Capture mobile approval live-mode rejection proof (if exercised)
- [ ] Confirm no live execution records were created

## 11. Post-Trade Memory/Reflection (Required Fresh Proof)

- [ ] Record post-trade reflection entry for the same UAT session
- [ ] Include rationale, outcome, and next action
- [ ] Link reflection artifact to candidate/trade id

Status note (2026-05-22): no fresh post-trade reflection artifact captured in this run.

## Mobile Approval Producer Checks

- [ ] Trigger re-qualify path from UI for blocked candidate
- [ ] Confirm candidate transitions to awaiting_approval
- [ ] Confirm mobile approval token row created (unused, non-expired)
- [ ] Confirm notification_outbox row created with pending/sent status

## Candidate Evidence Page Checks

- [ ] Open /candidates/{candidateId}/evidence
- [ ] Open /etf/candidates/{candidateId}/evidence
- [ ] Open /equity-alpha/candidates/{candidateId}/evidence
- [ ] Confirm evidence summary renders without errors

## Artifacts Index

- [x] Terminal transcript path: execution_subagent run on 2026-05-22 for scripts/uat-paper-trading.ps1 and scripts/etf-paper-pilot-evidence.ps1
- [ ] Screenshots folder:
- [x] Generated reports folder: Docs/runs/etf-paper-pilot
- [ ] Candidate IDs tested:
- [ ] Approval IDs tested:
- [ ] Trade IDs tested:

Generated evidence artifacts (fresh):
- Docs/runs/etf-paper-pilot/etf_pilot_evidence_20260522_094236.md
- Docs/runs/etf-paper-pilot/etf_pilot_evidence_20260522_094236.json

Run outcome summary (2026-05-22):
- uat-paper-trading.ps1 -Mode full: blocked by service connectivity + formatting gate failure (`gofmt would modify: internal/modules/tradingmodule/module.go`).
- etf-paper-pilot-evidence.ps1 -OperatorUATPassed: blocked (`etf/catalog`, `etf/pilot-status`, `etf/testing-readiness`, `readiness/etf-section`).

## Recommended Commands

- powershell -File scripts/uat-paper-trading.ps1 -Mode full
- powershell -File scripts/etf-paper-pilot-evidence.ps1 -OperatorUATPassed

## Signoff

- [ ] Reviewer confirms all required items complete
- [ ] Reviewer confirms broker paper mode and post-trade reflection evidence are fresh
- [ ] Prompt 10 marked ready for final signoff
