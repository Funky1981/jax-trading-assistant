# Phase 2 — Evidence Inbox and Event Journey

## Goal

Replace the current Monitor Inbox audit console with a beginner-friendly daily evidence surface.

The page must answer:

1. What arrived?
2. Is it genuine?
3. What did Jax do with it?
4. Do I need to act?

## Rename

User-facing name:

**Evidence Inbox**

The existing route may remain `/monitor/inbox` initially to avoid breaking links.

## Required page structure

### A. Page introduction

Heading:

`Evidence Inbox`

Help text:

`Review genuine and test events received by Jax. Opening an event does not approve or place a trade.`

### B. Summary strip

Show:

- Genuine
- Synthetic tests
- Rejected
- Deduplicated
- Candidates created

Every count must come from real API or persisted data.

### C. Beginner-friendly filtering

Default filters:

- All
- Genuine
- Synthetic tests
- Rejected
- Candidate created

Advanced filters can sit behind `More filters`.

### D. Evidence list

Do not lead with a wide technical table.

Use a responsive list or compact table with only:

- status
- headline
- source
- published time
- what Jax did next

Recommended `What Jax did next` values:

- Research only
- Candidate created
- Rejected
- Duplicate ignored
- Awaiting processing
- Unknown

### E. Event detail

Selecting an event should show:

#### What happened

- headline
- summary
- source
- original article links

#### Provenance

- genuine or synthetic
- discovery method
- publication/event time
- collection time
- Jax receipt time
- deterministic analysis identity
- AI provider/model only if actually persisted
- explicit `No AI used` state

#### What Jax recognised

- event type
- severity
- confidence
- affected assets
- `Unknown assets` when no truthful mapping exists
- confidence reasons

#### What Jax did next

- accepted
- rejected
- deduplicated
- normalised
- research only
- candidate created

### F. Event journey timeline

Display stages in order:

1. Discovered
2. Collected
3. Delivered
4. Received by Jax
5. Validated
6. Normalised
7. Decision processed
8. Candidate created
9. Human approval
10. Paper ticket
11. Outcomes

Each stage shows:

- state
- timestamp
- record identifier
- short explanation

Missing stages must use truthful wording:

- Not run
- Not applicable
- Awaiting processing
- No candidate created
- Missing persisted evidence

### G. Audit details

Move these behind a collapsed section:

- source-event ID
- internal IDs
- raw payload
- complete provenance fields
- technical rejection metadata

Use heading:

`Audit details`

Do not display raw JSON by default.

## Backend read model

Prefer a narrow authenticated read endpoint shaped for the UI.

Suggested concepts:

- evidence list
- evidence detail
- event journey
- summary counts

The frontend should not reconstruct the complete journey by interpreting raw metadata variants.

Read models must distinguish:

- genuine versus synthetic
- duplicate delivery versus duplicate persisted record
- no candidate versus error
- missing value versus zero
- deterministic analysis versus AI analysis

## Empty and error states

### No evidence

`No evidence has arrived yet. Genuine and controlled test events will appear here after Jax receives them.`

### No genuine evidence

`No genuine evidence matches this filter.`

### API unavailable

`Jax could not load the Evidence Inbox. Your data has not been changed.`

### Safety unknown

`Jax cannot confirm runtime safety. Evidence remains read-only, but open System Safety before relying on this view.`

## Tests

- Six existing genuine events display as genuine.
- Genuine source URLs open safely.
- Publication, collection and receipt times are distinct.
- Deterministic analysis does not display as AI.
- No-model state says `No AI used`.
- Unknown assets display as `Unknown`.
- QQQ is not inserted into unrelated events.
- Rejected events show reasons.
- Deduplicated events are clear.
- No-candidate is valid.
- Candidate links only appear when a candidate exists.
- Raw payload is collapsed by default.
- Mobile layout has no primary-path horizontal overflow.
- Refreshing the list does not create records.
- No approval or execution action is available.

## Runtime proof

Use existing persisted genuine events.

The operator should be able to open one event and explain:

- where it came from
- whether it is genuine
- when it was published
- when Jax received it
- whether AI was used
- whether a candidate was created
- why no action is required

## Exit criteria

Phase 2 is complete when the Evidence Inbox is understandable without direct SQL, PowerShell output or raw JSON.

## Implementation and acceptance record

Status: **PROVEN — 2026-07-24**

- The compatible `/monitor/inbox` route now presents responsive evidence cards and a detail experience instead of a forced 980 px table.
- Summary counts, provenance, timestamps, analysis identity, candidate linkage, and journey stages come from authenticated persisted read models.
- The runtime contained 30 Inbox rows: 25 provenance-confirmed genuine, 2 synthetic, and 3 older rejected rows without enough linked raw provenance to classify. Those three display `Provenance unavailable` rather than being fabricated as genuine.
- Seven genuine records persisted as `new` are labelled `Awaiting processing`; the UI does not misstate them as completed research-only decisions.
- Persisted deterministic analysis is explicitly separated from AI analysis. Empty asset mappings display `Unknown assets`.
- Interactive Playwright checks against the existing runtime passed at 320, 768 and 1280 px with no document-level horizontal overflow.
- The previous baseline remains at `images/monitor-inbox.png`. Accepted after screenshots are stored under `images/acceptance/phase2-evidence-inbox-{320,768,1280}.png`.
- Phase 1 Home-to-Evidence-Inbox acceptance was repeated at the three viewports and the prior horizontal-scroll failure is resolved.
- No evidence, candidate, approval, paper-ticket, execution, order, trade, fill, or runtime-setting mutation was introduced.

### Accepted compact-list usability refinement

The original Phase 2 capability and persisted-data proof remain **PROVEN**. A later focused usability refinement removed the permanent detail pane and sticky empty placeholder, allowed evidence summaries to use the full page width, and added compact pagination with ten records by default and an optional twenty-record page size.

One record can be expanded inline at a time. Source and provenance, Analysis, Journey, and Audit remain collapsed until requested, with raw payload nested and collapsed inside Audit. Complete persisted technical evidence remains accessible through these disclosures. Genuine and synthetic provenance, deterministic versus AI analysis, unknown assets, candidate linkage, and valid no-candidate outcomes remain truthfully represented.

The operator supplied desktop and mobile persisted-runtime screenshots and accepted the compact-list direction. Automated responsive and scrolling checks passed at 320, 768 and 1280 px. Codex did not independently capture the full set of six originally proposed refinement screenshots, and no such screenshot files are claimed here.

The refinement is read-only. It added no approval, trading, execution, record mutation, or runtime-setting control. Phase 4 was not started as part of this work.
