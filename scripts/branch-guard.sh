#!/usr/bin/env bash
set -euo pipefail

# Ensures the current branch exists on origin and is aligned with HEAD.
# Usage:
#   scripts/branch-guard.sh                # check only
#   scripts/branch-guard.sh --publish      # create/update origin/<branch>

REMOTE="${REMOTE:-origin}"
PUBLISH=false
if [[ "${1:-}" == "--publish" ]]; then
  PUBLISH=true
fi

branch=$(git rev-parse --abbrev-ref HEAD)
if [[ "$branch" == "HEAD" ]]; then
  echo "ERROR: detached HEAD; checkout a branch first" >&2
  exit 1
fi

if ! git remote get-url "$REMOTE" >/dev/null 2>&1; then
  echo "ERROR: remote '$REMOTE' is not configured" >&2
  exit 1
fi

# Refresh remote refs.
git fetch "$REMOTE" --prune >/dev/null 2>&1 || true

local_sha=$(git rev-parse HEAD)
remote_sha=""
if remote_line=$(git ls-remote --heads "$REMOTE" "$branch" 2>/dev/null); then
  remote_sha=$(awk '{print $1}' <<<"$remote_line")
fi

if [[ "$PUBLISH" == true ]]; then
  echo "Publishing $branch to $REMOTE..."
  git push -u "$REMOTE" "$branch"
  remote_sha=$(git ls-remote --heads "$REMOTE" "$branch" | awk '{print $1}')
fi

if [[ -z "$remote_sha" ]]; then
  echo "ERROR: $REMOTE/$branch does not exist."
  echo "Run: scripts/branch-guard.sh --publish"
  exit 2
fi

if [[ "$local_sha" != "$remote_sha" ]]; then
  echo "ERROR: local HEAD ($local_sha) != $REMOTE/$branch ($remote_sha)"
  echo "Run: git push $REMOTE $branch"
  exit 3
fi

echo "OK: $branch is published and in sync at $local_sha"
