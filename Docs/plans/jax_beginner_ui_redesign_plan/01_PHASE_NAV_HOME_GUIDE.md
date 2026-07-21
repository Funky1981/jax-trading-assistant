# Phase 1 — Navigation, Home and Guide

## Goal

Create a usable starting point before redesigning data-heavy pages.

At the end of this phase, a beginner should understand:

- what Jax is currently for
- whether the runtime is safe
- which page to open first
- where unfinished pages have moved
- how to get help

## Scope

### A. Simplify the application shell

Change the primary navigation to:

- Home
- Guide
- Evidence Inbox
- Candidates
- Outcomes
- System Safety

Add one collapsed **Review** section containing all other routes.

Preserve existing routes and deep links.

### B. Redesign Home

Home should answer:

1. **Is Jax safe?**
2. **What happened recently?**
3. **What needs my attention?**
4. **Where should I go next?**

Recommended layout:

#### Page header

- Heading: `Jax overview`
- Short explanation:
  `Review what Jax has received, what it decided and what happened to paper plans.`

#### Safety banner

Safe example:

> Paper-safe mode is on. Jax can collect evidence and track hypothetical plans, but it cannot place live orders.

Unsafe or unknown example:

> Jax cannot confirm its safety state. Open System Safety before continuing.

#### Four summary cards

- New evidence
- Candidates needing review
- Recent hypothetical outcomes
- System safety

Each card should have:

- one primary number or state
- one sentence explaining it
- one clear link

#### First-time action

- Primary: `Start the guide`
- Secondary: `Open Evidence Inbox`

### C. Redesign Guide

The Guide should be task-led, not architecture-led.

Recommended sections:

1. **What Jax does today**
2. **What Jax cannot do**
3. **Your current workflow**
4. **Key terms**
5. **Troubleshooting**
6. **Technical detail**, collapsed

Recommended task list:

| Task | Status example |
|---|---|
| Confirm paper-safe mode | Done / Needs attention |
| Review new evidence | Ready / No evidence |
| Understand a candidate | Waiting / Ready |
| Review hypothetical outcomes | Waiting / Ready |
| Check system safety | Available |

### D. Establish the help framework

Create reusable components:

- `PageIntro`
- `SafetyBanner`
- `HelpHint`
- `LearnMorePopover`
- `EmptyState`
- `StatusExplanation`
- `TechnicalDetailsDisclosure`
- `GlossaryTerm`
- `NextStepCard`

Rules:

- Tooltips contain one short sentence only.
- Rich explanations use a popover or expandable panel.
- Every primary page includes:
  - what this page is for
  - what the user can do here
  - what the page cannot do
- Every empty state explains why it is empty and what happens next.

## Help copy examples

### Evidence

`A stored news or research event that Jax can inspect.`

### Genuine

`Collected from a real external source.`

### Synthetic test

`Created for controlled testing. It is not live news.`

### Candidate

`A structured trade idea that still requires human judgement.`

### Paper plan

`A hypothetical trade plan. It is not an order or fill.`

### Deduplicated

`Jax had already stored this event, so it was not stored twice.`

## Accessibility requirements

- One clear page heading.
- Visible keyboard focus.
- Navigation usable by keyboard.
- Statuses use text and colour.
- Mobile layout works at 320 CSS pixels.
- Review section state is exposed with `aria-expanded`.
- Help controls have accessible names.
- Safety changes are announced as status messages.

## Tests

- Primary navigation contains exactly the approved pages.
- Review is collapsed by default.
- Existing routes still resolve.
- Home shows safe and unsafe states correctly.
- Home does not hardcode runtime values.
- Guide task states reflect real API data where available.
- Help controls work with keyboard and screen reader labels.
- Mobile navigation remains usable.
- No new mutation controls are introduced.

## Runtime proof

Using the local runtime:

- Home shows paper mode.
- Live trading is off.
- Execution is off.
- Execution worker is off.
- Maximum leverage is no greater than 1x.
- Evidence Inbox is reachable in one click.
- All old pages remain available under Review.

## Exit criteria

Phase 1 is complete only when the user can open Jax and clearly identify the next safe action without being told verbally.
