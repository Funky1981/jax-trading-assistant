# Testing and Acceptance Plan

## Test philosophy

The UI must be proven with the existing local persisted runtime data, not only mocked fixtures.

Fixtures remain useful for isolated component testing but must be explicitly labelled as fixtures.

## Automated test layers

### Unit and component tests

Cover:

- safety banner
- status badge
- empty state
- help tooltip
- help popover
- technical disclosure
- evidence row
- event timeline
- candidate summary
- checkpoint card
- historical versus selected-journey counts

### Accessibility tests

Run automated accessibility checks on:

- Home
- Guide
- Evidence Inbox
- Candidate Review
- Outcomes
- System Safety

Check:

- heading order
- accessible names
- focus visibility
- keyboard navigation
- status announcements
- colour-independent meaning
- dialog/popover focus management

### Responsive tests

Capture and verify:

- 320 px
- 768 px
- 1280 px

Core reading paths must not require horizontal scrolling.

Wide audit tables may scroll only inside an explicitly advanced technical container.

### End-to-end tests

Required flows:

1. First-time Guide flow
2. Home to Evidence Inbox
3. Genuine evidence detail
4. No-candidate evidence
5. Candidate Review
6. Candidate to Outcomes
7. Pending checkpoint
8. Missing-data checkpoint
9. Safe System Safety state
10. Unsafe/unknown safety state
11. Review section route access

## Real-data runtime acceptance

### Home

- Shows paper-safe state from the API.
- Links to Evidence Inbox.
- Does not show fabricated counts.

### Evidence Inbox

- Shows existing genuine events.
- Shows genuine sources.
- Shows unknown assets honestly.
- Shows no false AI usage.
- Shows no duplicate rows after refresh.

### Candidate Review

- Shows the existing QQQ proof candidate.
- Shows the human approval.
- Shows no hardcoded sizing fallback.

### Outcomes

- Shows existing 1-hour and 1-day checkpoints.
- Shows the current 1-week state.
- Labels all results hypothetical.

### System Safety

- Shows paper mode.
- Shows live trading off.
- Shows execution off.
- Shows worker off.
- Shows leverage no greater than 1x.
- Separates global historical rows from selected-journey rows.

## User acceptance questions

At the end of each phase, ask only questions relevant to that phase.

### Phase 1

- Do you know which page to open first?
- Can you find the pages moved into Review?
- Is the safety state obvious?

### Phase 2

- Can you explain what one genuine event is?
- Can you tell what Jax did with it?
- Is any field still too technical?

### Phase 3

- Is it clear that no real trade occurred?
- Can you understand the approval and paper plan?
- Can you understand the outcome checkpoints?

### Phase 4

- Can you understand the main terms without external help?
- Can you confirm Jax is safe?
- Are the technical details available without dominating the page?

## Regression requirements

Existing capabilities must remain intact:

- authentication
- protected routes
- World Monitor ingestion
- event persistence
- deduplication
- candidate evidence
- human approval records
- paper-ticket records
- outcome tracking
- historical instruction quarantine
- execution worker safety
- genuine candle provenance

## Failure handling

Do not mark a phase complete when:

- a relevant changed-path test fails
- the frontend build fails
- primary pages cannot load local persisted data
- safety state is inferred rather than retrieved
- mobile layout is unusable
- a hypothetical value is presented as actual

Known unrelated failures must be documented separately rather than hidden.
