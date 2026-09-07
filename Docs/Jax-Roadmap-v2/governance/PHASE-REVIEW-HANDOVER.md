# Codex Phase Review Handover Template

Return the entire phase handover inside one Markdown code block at the end of an autonomous phase or earlier for a hard-stop external review.

## Identity
- Phase:
- Branch:
- Starting commit:
- Ending commit:
- Working-tree state:
- Upstream / ahead-behind:

## Work packages
For each package: ID/title, commit, internal result, capability delivered, important limitations.

## Architecture delivered
- New/changed boundaries:
- Contracts/versions:
- Persistence/migrations:
- Providers/external systems:
- Observability:
- Replay/determinism:

## Phase exit condition
Quote the exact roadmap exit condition.
State `DEMONSTRATED` or `NOT DEMONSTRATED` and provide reproducible proof.

## Verification
- Focused package tests:
- Full suite:
- Vet/lint:
- Golden/contract tests:
- Migration/integration tests:
- Live-source verification:
- Replay/backtest/evaluation evidence:
- Roadmap/manifest validation:

## Adversarial phase self-review
- Serious defects found:
- Corrections made:
- Remaining limitations:
- Accepted/non-blocking debt:
- Potential roadmap issues:

## Cost / external dependencies
- New paid services:
- Current spend:
- Credentials required:
- Deferred paid dependencies:
- Model escalations:

## Safety
- `ALLOW_LIVE_TRADING`:
- `BROKER_EXECUTION_ALLOWED`:
- Execution worker:
- Leverage:
- Recommendation execution authority:
- Real-money mutation:
- New trading-state mutation capability:

## Repository
- Commits:
- Migrations:
- Worktree:
- Unpushed work:
- User action required:

## Recommended technical-lead decision
Recommend one only:
- `GO PHASE NN`
- `CONDITIONAL GO PHASE NN`
- `NO-GO PHASE NN`
- `ROADMAP CHANGE`

Recommendation only. **STOP. Do not begin the next phase until external technical-lead review is complete.**
