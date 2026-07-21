# Product Direction and Information Architecture

## Product position

For the current development stage, Jax is an **operator evidence system**, not a live trading terminal.

The frontend should prioritise:

- evidence review
- provenance
- decision transparency
- hypothetical paper outcomes
- runtime safety
- beginner guidance

It should not prioritise:

- live order placement
- advanced strategy configuration
- legacy experiments
- technical diagnostics
- feature discovery across unfinished modules

## Recommended primary navigation

The primary navigation should contain only:

1. **Home**
   - Is Jax safe?
   - What happened recently?
   - What needs attention?

2. **Guide**
   - What Jax currently does
   - What it cannot do
   - The current operator workflow
   - Contextual help and glossary

3. **Evidence Inbox**
   - Genuine and synthetic events
   - Source provenance
   - Jax processing status
   - Candidate linkage

4. **Candidates**
   - Candidate review
   - evidence
   - approval state
   - paper-plan assumptions

5. **Outcomes**
   - hypothetical checkpoint results
   - market-data provenance
   - pending and missing-data states

6. **System Safety**
   - paper/live state
   - execution state
   - worker state
   - leverage
   - execution-side counts

## Review section

Create one collapsed section called:

**Review**

Move all currently non-essential pages into it, including:

- AI Trading
- Manual Trading
- Swing Trading
- Research
- Macro Events
- Analysis
- Notifications
- Settings
- Testing
- E2E Tests
- Portfolio
- Blotter
- Assistant
- legacy equity routes
- legacy ETF routes
- specialist guides
- admin and QA pages

The routes remain available. They are not deleted.

## Navigation behaviour

- The Review section is collapsed by default.
- Primary items remain visible at all times.
- Active route state must be obvious.
- Labels use plain English.
- Icons support labels but never replace them.
- Mobile navigation must expose the same hierarchy.
- A user should reach Evidence Inbox from Home in one click.
- A user should reach Help from every primary page.

## Beginner mode

Beginner mode should become the default operator experience.

Beginner mode should:

- show the simplified primary navigation
- include inline explanations
- hide advanced details behind disclosures
- replace unexplained jargon with plain language
- show safe next steps
- show explicit empty and waiting states

Advanced mode may expose more technical details, but must not introduce unsafe actions.

## Product vocabulary

Use these terms consistently:

| Avoid | Use |
|---|---|
| Monitor payload | Evidence |
| Monitor Inbox | Evidence Inbox |
| Candidate Evidence | Candidate Review |
| Execution overview | System Safety |
| Order instruction | Paper plan, unless it is a genuine execution record |
| Result | Hypothetical outcome |
| Unknown/empty shown as `-` | Unknown, Not supplied, Not applicable |
| Legacy | Review, unless describing an explicitly archived implementation |

## Non-negotiable safety language

The UI must never imply that:

- a hypothetical entry is a fill
- hypothetical P&L is realised P&L
- a paper ticket is an order
- a candidate is approved automatically
- zero journey records means the entire database is empty
- an unknown value is zero
- deterministic analysis is AI analysis

## Progressive disclosure

Each page should show only what the operator needs first.

Advanced details should sit behind:

- `Show details`
- `How Jax decided this`
- `Audit details`
- `Technical diagnostics`
- `Raw payload`

Raw JSON must never be the primary content surface.

## Phase boundary

Do not redesign every existing page.

A page should be redesigned only when:

1. it becomes part of the active workflow;
2. its data is genuine and available;
3. the operator has a clear task to complete on it;
4. it passes the promotion checklist in `05_REVIEW_SECTION_AND_PAGE_PROMOTION.md`.
