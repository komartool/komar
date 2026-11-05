#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

allowed='^\s*//(go:|nolint|\s*\+build)'

found=$(git ls-files --cached --others --exclude-standard -- '*.go' '*.java' '*.kts' \
  | while read -r f; do [ -f "$f" ] && echo "$f"; done \
  | xargs -r grep -nHE '(^|[^:"])//|/\*' 2>/dev/null \
  | grep -vE "^[^:]+:[0-9]+:${allowed}" \
  | grep -vE '"[^"]*(//|/\*)[^"]*"' || true)

if [ -n "$found" ]; then
  echo "comments are not allowed (AGENTS.md rule 1):"
  echo "$found"
  exit 1
fi
