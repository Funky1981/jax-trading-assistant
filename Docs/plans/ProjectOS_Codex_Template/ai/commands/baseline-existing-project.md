# Command: Baseline Existing Project

Use this when adding ProjectOS to an existing repo.

## Goal

Create a current-state baseline without trying to perfectly reconstruct history.

## Steps

1. Inspect the repo structure.
2. Read README and existing docs.
3. Identify stack, architecture, entry points, database, tests, and deployment setup.
4. Identify what appears complete.
5. Identify what appears incomplete.
6. Identify risks and unknowns.
7. Create/update:
   - `/project/brief.md`
   - `/project/roadmap.md`
   - `/project/backlog.md`
   - `/project/current-focus.md`
   - `/project/decisions.md`
   - `/project/risks.md`
   - `/project/releases.md`

## Output

```md
## Current State

## Confirmed Stack

## Working Areas

## Incomplete Areas

## Risks

## Recommended Roadmap Phase

## Next 3 Actions
```

## Rules

- Do not guess beyond evidence.
- Mark unknowns clearly.
- Prefer current-state truth over old intentions.
