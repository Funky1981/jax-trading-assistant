# Add Override-Reason Collection and Calibration Reporting

## Summary

- Priority: P2
- Phase: Phase 3
- Estimate: 3-5d
- Outcome: trust and governance

## Objective

Capture why humans overrule sentiment-enriched Opportunities and expose aggregate reporting for calibration and governance.

## In-scope touchpoints

- Approvals
- Analytics
- Reporting views

## Implementation notes

- Add structured override reasons for approval rejection, deferral, and manual override flows.
- Include reasons such as weak sentiment evidence, stale sources, policy concern, risk concern, price/sentiment divergence, duplicate idea, and other with note.
- Record whether sentiment evidence was viewed before the decision.
- Add aggregate reporting for override rate, reason distribution, and sentiment-enriched approval decision time.
- Avoid exposing sensitive personal notes in broad reporting.

## Acceptance criteria

- Users can record why they overruled sentiment-enriched Opportunities.
- Reports show override reason distribution and calibration signals.
- Analytics includes `approval_override_reason_selected`.
- Override collection does not block urgent rejection or deferral flows.

## Suggested validation

- Run backend tests for reason persistence and report aggregation.
- Run frontend tests for approval reason UI and optional note handling.
- Verify analytics events do not include sensitive note text.

## Dependencies

- Builds on `14-approval-sentiment-evidence-pack.md` and `08-baseline-analytics-events.md`.
