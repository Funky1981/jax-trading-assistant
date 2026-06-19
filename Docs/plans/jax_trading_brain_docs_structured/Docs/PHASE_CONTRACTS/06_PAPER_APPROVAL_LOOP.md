# Phase 6 Contract: Paper Approval Loop

## Objective

Wire trade candidates into a human-approved paper trading flow.

## Delivers

- Paper ticket schema
- Candidate-to-paper-ticket conversion
- Human approval statuses
- Paper-only enforcement
- Ticket validation tests

## Explicitly does not deliver

- Live trading
- Auto approval
- Auto execution
- Day trading
- Long-term portfolio execution

## User-facing capability made testable

A high-quality candidate can become a paper ticket requiring human approval.

## Acceptance tests

- NO_TRADE cannot create ticket.
- WATCH cannot create ticket.
- Risk-rejected candidate cannot create ticket.
- Human approval required.
- Expired ticket cannot execute.
- Live execution remains blocked.

## Required evidence

- Ticket tests.
- Approval tests.
- Capability matrix update.

## What Jax still cannot do afterwards

- Trade live.
- Learn from paper outcomes unless review phase is complete.
