# Prompt 10 UAT Status - 2026-05-21

## Scope

This report tracks Prompt 10 acceptance items from:
- Docs/archive/programs/codex/jax-etf-news-research-build-pack/Docs/plans/etf-news-research-platform/15-codex-implementation-prompts.md

Prompt 10 requires proof of:
- ETF-only defaults
- event ingestion
- event study generation
- priced-in scoring
- evidence bundle
- candidate creation
- approval
- paper execution instruction
- broker paper mode
- no live trading
- post-trade memory/reflection

## Evidence Used

- Go tests run on 2026-05-21:
  - go test ./internal/modules/approvals ./cmd/trader ./internal/modules/candidates
  - go test ./cmd/research ./internal/modules/approvals ./internal/modules/tradingmodes ./libs/strategytypes ./internal/modules/etfnews ./cmd/trader
- Frontend test run on 2026-05-21:
  - npm test -- --run src/pages/Step9BeginnerPages.test.tsx
- Existing readiness artifacts in Docs/runs/etf-paper-pilot and Docs/UAT_PAPER_TRADING.md

## Prompt 10 Checklist Status

1. ETF-only defaults: PASS
- Evidence: existing ETF allowlist enforcement and passing backend suites.

2. Event ingestion: PASS
- Evidence: cmd/research backfill paths and tests in cmd/research.

3. Event study generation: PASS
- Evidence: event-study backfill route and schema migrations (000022/000023), tests passing.

4. Priced-in scoring: PASS
- Evidence: priced-in engine tests in cmd/research and schema fields.

5. Evidence bundle: PASS
- Evidence: research evidence bundle builder and persistence paths in cmd/research tests.

6. Candidate creation: PASS
- Evidence: internal/modules/candidates tests and watcher candidate flow in cmd/trader.

7. Approval: PASS
- Evidence: internal/modules/approvals tests and approval handlers.

8. Paper execution instruction: PASS
- Evidence: approval service creates execution instructions on approved decisions.

9. Broker paper mode: PARTIAL
- Evidence: paper-only constraints exist, but no fresh full-session runtime proof attached in this report.

10. No live trading: PASS
- Evidence: mobile approval path rejects live mode requests and gate checks enforce paper-only path.

11. Post-trade memory/reflection: PARTIAL
- Evidence: platform memory capabilities exist, but no fresh Prompt 10 run artifact in this report showing end-to-end post-trade reflection for ETF flow.

## New Work Completed In This Session

1. Runtime crash fix
- BeginnerUX provider now always provides context during initial mount.

2. Prompt 8 producer wiring
- Added idempotent mobile approval producer:
  - queues Telegram outbox notifications
  - creates one-time approval tokens
  - invoked after candidate qualification in watcher and manual refresh paths

3. Prompt 9 dedicated evidence screen
- Added dedicated route and page for candidate evidence:
  - /candidates/:candidateId/evidence
- Added approvals-page navigation button to evidence screen.

4. Step 9 targeted tests
- Added frontend tests for ETF universe, strategy cards, and research timeline pages.

## Remaining To Close Prompt 10 Fully

1. Run and archive a fresh full UAT session artifact that includes broker paper mode proof for this branch.
2. Attach explicit post-trade memory/reflection evidence from that same run.
3. Update signoff environment gates only after the above evidence is reviewed.

## Suggested Execution Commands

1. Backend UAT pipeline
- powershell -File scripts/uat-paper-trading.ps1 -Mode full

2. ETF pilot evidence report
- powershell -File scripts/etf-paper-pilot-evidence.ps1 -OperatorUATPassed

3. Archive generated outputs under Docs/runs/etf-paper-pilot with date/time stamp.
