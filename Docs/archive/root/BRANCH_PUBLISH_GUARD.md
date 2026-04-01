# Branch Publish Guard

This repository now includes a branch publication guard to prevent "work only in container" incidents.

## Commands

- Check branch is already published and in sync:

```bash
make branch-guard
```

- Publish current branch to origin and verify sync:

```bash
make branch-publish
```

## Policy

Before starting or continuing implementation, run `make branch-publish` once and require:

1. `origin/<current-branch>` exists.
2. `origin/<current-branch>` SHA matches local `HEAD`.

If either condition fails, do not continue coding until fixed.

The guard is implemented in `scripts/branch-guard.sh`.
