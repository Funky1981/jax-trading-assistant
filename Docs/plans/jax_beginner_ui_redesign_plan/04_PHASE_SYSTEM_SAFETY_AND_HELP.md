# Phase 4 — System Safety and Contextual Help

## Goal

Make safety state unmistakable and ensure help is available at the point of need throughout the primary workflow.

## System Safety page

The first viewport should show only the most important operator facts:

- Mode
- Live trading
- Execution
- Execution worker
- Broker execution
- Maximum leverage
- Broker orders
- Trades
- Fills
- Last successful status check

Recommended display:

| Field | Beginner label |
|---|---|
| Runtime mode | Mode |
| `ALLOW_LIVE_TRADING` | Live trading |
| `EXECUTION_ENABLED` | Execution |
| Worker state | Execution worker |
| Broker capability | Broker execution |
| Max leverage | Maximum leverage |

## Safe state

When all conditions are safe, show:

> Paper-safe mode is on. Live trading, broker execution and the execution worker are off.

## Unsafe or uncertain state

When any required value is unsafe or unavailable, show a prominent warning.

Example:

> Jax cannot confirm a paper-safe state. Do not rely on candidate or outcome pages until this is reviewed.

The UI remains read-only.

## Technical diagnostics

Move existing technical content under collapsed sections:

- Services
- Datasets
- Metrics
- Memory
- Events
- Logs
- Detailed health
- Historical execution records

Use:

`Technical diagnostics`

Do not remove this information.

## Historical versus selected-journey state

Clearly separate:

- global historical records
- records created by the selected evidence journey
- records created by the selected candidate
- records created by the selected paper ticket

Never say `No execution records exist` when historical rows are present.

Use:

`This journey created no execution records.`

## Help throughout the product

Every primary page must contain:

1. One-sentence purpose.
2. One-sentence safety boundary.
3. A `What does this mean?` control.
4. A link to the relevant Guide section.
5. A clear next step.

## Help patterns

### Tooltip

Use only for short definitions.

Example:

`MFE — the best hypothetical price movement seen during the checkpoint.`

### Popover

Use for one short explanatory block.

Example:

`Why is this research only?`

### Expandable help panel

Use for:

- decision methodology
- timestamp semantics
- provenance
- outcome calculation
- safety interpretation

### Guide link

Use:

`Learn how this page fits into the Jax workflow.`

## Glossary

Create one reusable glossary source rather than duplicating definitions.

Minimum terms:

- Evidence
- Genuine
- Synthetic test
- Deduplicated
- Candidate
- Approval
- Paper ticket
- Hypothetical entry
- Checkpoint
- MFE
- MAE
- Stop touched
- Target touched
- Same-candle ambiguity
- Runtime mode
- Execution
- Execution worker
- Broker execution
- Leverage
- Provenance
- Deterministic analysis
- AI analysis

## First-run tour

Maximum five steps:

1. Home and safety banner
2. Evidence Inbox
3. Event journey
4. Candidate Review and Outcomes
5. System Safety

Controls:

- Next
- Back
- Finish
- Do not show again
- Restart guide

Do not block use of the application behind the tour.

## Beginner and advanced display

Beginner display:

- plain-language summaries
- minimal columns
- help visible
- audit and diagnostics collapsed

Advanced display:

- additional provenance
- IDs
- technical statuses
- raw payload access
- diagnostic details

Safety language and hypothetical labels remain visible in both modes.

## Tests

- Safe state banner uses runtime data.
- Unknown safety state produces warning.
- Historical and selected-journey counts remain separate.
- Tooltips are keyboard accessible.
- Popovers return focus correctly.
- Guide links target the correct section.
- First-run tour can be dismissed and restarted.
- Help wording is consistent across pages.
- No help component contains an unsafe mutation.
- Screen readers announce status changes.
- Advanced mode does not hide safety labels.

## Exit criteria

Phase 4 is complete when a beginner can understand each primary page without needing a developer to explain its terminology.
