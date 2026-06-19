# Paper Trading Decision Gate

## Purpose

This document defines how a Jax trade candidate becomes a paper-trading ticket.

Paper trading is the only execution path in the current roadmap.

## Required flow

```text
Event
  -> Decision Core
  -> Swing Brain
  -> Risk Veto
  -> Trade Candidate
  -> Human Approval
  -> Paper Ticket
  -> Paper Outcome Review
```

## Paper ticket schema

```json
{
  "paper_ticket_id": "pt_001",
  "decision_id": "dec_001",
  "event_id": "evt_001",
  "asset": "SHEL",
  "setup_family": "commodity_linked_equity_dislocation",
  "decision_evidence": {},
  "risk_assessment": {},
  "required_confirmations": [],
  "invalidation_conditions": [],
  "proposed_entry_zone": {
    "low": 0,
    "high": 0
  },
  "proposed_stop": 0,
  "proposed_target": 0,
  "risk_reward": 0,
  "max_paper_position_size": 0,
  "human_approval_status": "PENDING_REVIEW",
  "expires_at": "2026-06-19T16:30:00Z"
}
```

## Approval statuses

```text
PENDING_REVIEW
APPROVED_FOR_PAPER
REJECTED_BY_USER
DEFERRED
EXPIRED
```

## Gate rules

1. Only `TRADE_CANDIDATE` can create a paper ticket.
2. Risk veto must pass.
3. Human approval is required.
4. Ticket must include evidence.
5. Ticket must include invalidation.
6. Ticket must include risk/reward.
7. Ticket must include expiry.
8. Live trading remains disabled.
9. Expired candidates cannot be executed.
10. Paper trades must be reviewed.

## Hard reject conditions

Reject paper ticket creation if:

- decision is `NO_TRADE`
- decision is `WATCH`
- decision is `SETUP_FORMING`
- risk veto rejects
- no invalidation condition exists
- risk/reward < 2:1
- no evidence bundle exists
- event is stale
- human approval missing
- live execution requested

## Human review screen must show

- event summary
- decision
- primary reason
- supporting reasons
- affected asset
- setup family
- risk score
- conflict score
- edge score
- required confirmations
- invalidation conditions
- proposed entry/stop/target
- paper-only warning

## Current roadmap limitation

Paper trading does not imply live readiness.

A paper trade outcome can only move a setup family toward:

```text
PAPER_PROVEN
```

not:

```text
LIVE_READY
```
