#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
installer="$repo_root/install.sh"

test -x "$installer"
bash -n "$installer"

help_output="$(bash "$installer" --help)"
grep -q -- "--setup" <<<"$help_output"
grep -q -- "--force" <<<"$help_output"
grep -q -- "Go 1.25.0" <<<"$help_output"

grep -q -- "apt-get install -y golang-go" "$installer"
grep -q -- "GOTOOLCHAIN" "$installer"
grep -q -- "GOPROXY" "$installer"
grep -q -- "gapcode" "$installer"

printf 'installer tests passed\n'
