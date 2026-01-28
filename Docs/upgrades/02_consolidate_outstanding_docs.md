# Consolidate Outstanding Changes Documentation

There are two nearly identical pull requests (#4 and #5) that add an `OUTSTANDING_CHANGES.md` to document a change in file permissions for a CLI script. Keeping multiple branches with the same change creates confusion and review overhead.

## Why it matters

Maintaining a single source of truth helps reviewers understand what needs to be merged and prevents duplicate documentation from cluttering the repository. Only one version of `OUTSTANDING_CHANGES.md` should be merged.

## Tasks

1. **Review the duplicate PRs**
   - Both PR #4 and PR #5 add the same `OUTSTANDING_CHANGES.md` documenting the change to `services/hindsight/hindsight-control-plane/bin/cli.js` (mode changed to `100755`).
   - Confirm that there are no other files modified in these branches.

2. **Choose one branch to merge**
   - Pick either `codex/document-outstanding-changes-in-markdown-files` or `codex/document-outstanding-changes-in-markdown-files-4fco4h` as the source of truth.
   - Close the redundant PR to avoid duplicate merges.

3. **Merge the chosen PR**
   - Merge the selected branch into `main`.
   - Ensure that `OUTSTANDING_CHANGES.md` lives at the repo root and clearly lists any review‑relevant changes on non‑source files (e.g. file mode updates).

4. **Communicate the process**
   - Update internal documentation to describe how `OUTSTANDING_CHANGES.md` should be used: it is intended as a summary of file‑mode and other “hidden” changes that may not be obvious in the diff.
   - Encourage future contributors to document such changes in this file to aid reviewers.