#!/usr/bin/env bash
set -euo pipefail

TARGET_REPO_PATH="${1:-}"

if [ -z "$TARGET_REPO_PATH" ]; then
  echo "Usage: ./install-projectos.sh /path/to/repo"
  exit 1
fi

SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_DIR="$(cd "$TARGET_REPO_PATH" && pwd)"

echo "Installing ProjectOS into $TARGET_DIR"

for folder in project ai .github github-project-board; do
  if [ -d "$SOURCE_DIR/$folder" ]; then
    cp -R "$SOURCE_DIR/$folder" "$TARGET_DIR/"
    echo "Copied $folder"
  fi
done

echo "ProjectOS installed."
echo "Next: complete /project/brief.md and run /ai/commands/baseline-existing-project.md"
