# Delivery Rules for Each Phase

## Work one phase at a time

Do not begin the next phase until:

- current scope is implemented
- relevant tests pass
- runtime proof is complete
- screenshots are reviewed
- the user has accepted the result
- the phase is committed

## Before making changes

Record:

- branch
- HEAD
- working-tree state
- untracked files
- upstream
- ahead/behind state

Do not:

- reset
- rebase
- clean
- stash
- discard
- overwrite
- amend prior commits
- switch branches
- push without explicit instruction

## Screenshot review

At the start of every phase:

1. Inspect all relevant screenshots in `images/`.
2. Identify the current usability problems.
3. Record which screenshots were reviewed.
4. Capture new screenshots after implementation at:
   - desktop
   - tablet
   - mobile

## Backend rules

- Prefer narrow authenticated read models.
- Do not expose database tables directly.
- Use parameterised SQL.
- Distinguish missing from zero.
- Distinguish hypothetical from actual.
- Distinguish selected journey from historical global state.
- Do not add mutation endpoints during this redesign.
- Do not weaken execution safety.

## Frontend rules

- No hardcoded runtime counts.
- No fabricated financial values.
- No placeholder production data.
- No forced asset mappings.
- No hidden fallback sizing.
- No raw-JSON-first page.
- No unexplained acronyms.
- No new trading actions.
- No live-order language.

## Documentation

Update:

- capability matrix
- user guide
- route/navigation documentation
- screenshots or design notes
- relevant API read-model documentation

Mark a UI capability proven only after real runtime browser proof.

## Commit strategy

Create one focused commit per phase.

Suggested commit messages:

- Phase 1: `feat: simplify beginner operator navigation`
- Phase 2: `feat: redesign evidence inbox journey`
- Phase 3: `feat: add beginner candidate and outcome review`
- Phase 4: `feat: simplify system safety and contextual help`

Do not combine unrelated cleanup.

## Required handover

Every phase handover must include:

### Local state confirmed

- branch
- starting HEAD
- ending HEAD
- working tree before and after
- ahead/behind state
- confirmation that no work was discarded

### Screenshots reviewed

- original screenshots reviewed
- new screenshots captured
- viewport sizes

### UI implemented

- pages
- routes
- components
- read models

### Beginner improvements

- jargon removed
- help added
- choices reduced
- advanced information hidden
- empty/error states added

### Data shown

- real persisted records
- runtime state
- provenance

### Removed assumptions

- hardcoded values
- fallback calculations
- fabricated labels
- inferred states

### Runtime proof

- what was observed in the browser
- which genuine records were visible

### Safety proof

- runtime mode
- live trading
- execution
- worker
- broker execution
- leverage
- approvals created
- instructions created
- orders created
- trades created
- fills created

### Verification performed

- passing checks
- relevant failures
- unrelated failures
- checks not run

### What was proven

Only demonstrated facts.

### What remains unproven

Do not claim later phases are complete.

### Single next action

Exactly one action.
