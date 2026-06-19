# Phase 7 Contract: Post Decision Review

## Objective

Make Jax learn from every decision, including no-trades.

## Delivers

- Decision log
- Review schedule
- Outcome review schema
- No-trade review
- Paper-trade review
- Memory tagging

## Explicitly does not deliver

- Autonomous strategy mutation
- Live trading
- Automatic promotion to live
- New trading brains

## User-facing capability made testable

Jax can show whether a past no-trade or paper trade decision was correct.

## Acceptance tests

- NO_TRADE schedules 1d/1w/1m review.
- WATCH schedules review.
- Paper trade schedules review.
- Review records lesson.
- Review can flag missed opportunity.
- Review can flag avoided loss.

## Required evidence

- Review tests.
- Example reviewed decision.
- Capability matrix update.

## What Jax still cannot do afterwards

- Trade live.
- Autonomously change strategy rules without approval.
