#!/usr/bin/env bash
set -euo pipefail

fail=0

tracked_files="$(git ls-files)"

if printf '%s\n' "$tracked_files" | rg -q '(^|/)(bale_adapter\.py|run\.sh)$'; then
  echo "private Bale sidecar files must not be tracked" >&2
  fail=1
fi

# The scanner deliberately excludes itself and planning notes. Planning notes
# are not part of the public release surface and may contain test fixtures.
if git grep -nE \
  '(gh[pousr]_[A-Za-z0-9_]{20,}|xox[baprs]-[A-Za-z0-9-]{20,}|AIza[0-9A-Za-z_-]{20,}|sk-[A-Za-z0-9_-]{20,}|[0-9]{8,12}:[A-Za-z0-9_-]{30,})' \
  -- ':!scripts/check-public-repo.sh' ':!docs/superpowers/**' >/tmp/gapcode-connect-secret-scan.txt; then
  cat /tmp/gapcode-connect-secret-scan.txt >&2
  echo "possible credential material found in tracked public files" >&2
  fail=1
fi

if git grep -nEi \
  '(talking to (the )?(user|prompter)|human partner|ask the user to choose|interactive prompts to guide)' \
  -- ':!scripts/check-public-repo.sh' ':!docs/superpowers/**' >/tmp/gapcode-connect-tone-scan.txt; then
  cat /tmp/gapcode-connect-tone-scan.txt >&2
  echo "internal agent/prompter language found in public files" >&2
  fail=1
fi

rm -f /tmp/gapcode-connect-secret-scan.txt /tmp/gapcode-connect-tone-scan.txt

if [ "$fail" -ne 0 ]; then
  exit 1
fi

echo "public repository checks passed"
