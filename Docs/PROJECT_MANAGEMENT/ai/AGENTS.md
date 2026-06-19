# Codex Operating Rules

Codex must treat this repo as a managed project, not a loose coding sandbox.

This repo uses ProjectOS. The `/project` folder is the source of truth for roadmap, backlog, current focus, decisions, risks, and releases.

## Mandatory Reading Before Work

Before planning or implementing anything, read:

1. `/project/brief.md`
2. `/project/roadmap.md`
3. `/project/current-focus.md`
4. `/project/backlog.md`
5. `/project/decisions.md`
6. `/project/risks.md`

## Core Rules

1. Do not implement work that is not connected to the roadmap or backlog.
2. Do not make architecture changes without adding or updating a decision record.
3. Do not introduce new risks without updating `/project/risks.md`.
4. Do not mark work complete unless acceptance criteria are met.
5. Keep changes small and reviewable.
6. Prefer production-grade design over shortcuts.
7. If requirements are unclear, state assumptions clearly before coding.
8. If a task is too large, split it into smaller backlog items.
9. Update `/project/current-focus.md` after meaningful progress.
10. Update `/project/releases.md` for user-facing or operational changes.
11. Do not silently rewrite unrelated areas of the codebase.
12. Do not claim something is complete without evidence from tests, checks, or manual verification.

## Standard Workflow

For every feature:

```text
Read project state
→ Confirm roadmap/backlog link
→ Plan implementation
→ Identify risks
→ Implement focused changes
→ Test/review
→ Update ProjectOS docs
→ Summarise changes and remaining risks
```

## Planning Mode

When asked to plan, do not code.

Output:

```md
## Roadmap Link

## Backlog Item

## Goal

## Assumptions

## Implementation Plan

## Files Likely to Change

## Risks

## Acceptance Criteria

## Review Plan
```

## Implementation Mode

When asked to implement, only work from an approved plan or clear backlog item.

After implementation, output:

```md
## Implemented

## Files Changed

## Tests / Checks

## ProjectOS Updates

## Risks / Follow-ups
```

## Review Mode

When asked to review, be strict.

Check:

- Roadmap/backlog alignment
- Acceptance criteria
- Test evidence
- Architecture impact
- Risk impact
- Documentation updates
- Release-note impact

## Do Not

- Do not implement speculative features outside the roadmap.
- Do not add dependencies without explaining why.
- Do not make hidden architecture changes.
- Do not remove tests without justification.
- Do not leave untracked TODOs.
- Do not pretend something is production-ready if it has not been validated.
