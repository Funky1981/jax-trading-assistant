# UI: How you inspect and verify decisions

Add these views (can be inside existing Research/Analysis/Testing pages):

## A) Run Detail (Analysis)
For a runId show:
- Config snapshot (instance config + parameters)
- Strategy evaluation summary
- Trade list
- Metrics summary
- **Decision Timeline** (audit events + AI decisions)

## B) AI Decision Inspector
For each AI decision:
- Purpose
- Inputs (raw + derived)
- Prompt template + version
- Output structured
- Confidence
- Acceptance record (accepted/rejected + reasons)
- Links to affected signals/trades

## C) Gate Status Inspector (Testing)
For each gate:
- pass/fail history
- last test run summary
- artifact download links

This is how you verify “every decision” without trusting memory or assumptions.

