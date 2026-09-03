#!/usr/bin/env bash
set -euo pipefail

# GapCode Connect bootstrap installer.
#
# This script is the supported fresh-clone entry point. It installs Go with
# the host package manager only when Go is missing, then builds the connector
# with Go's toolchain selection.

go_version="1.25.0"
setup=0
force=0
output=""

usage() {
  cat <<'EOF'
GapCode Connect installer

Usage:
  ./install.sh [options]

Builds the connector from this checkout. If Go is not available, the script
offers to install it with the host package manager. Go then downloads the
project's required Go 1.25.0 toolchain through GOPROXY when necessary.

Options:
  --setup          run the interactive Telegram/Bale setup wizard after build
  --force          allow --setup to replace an existing private config
  --output PATH    write the binary to PATH (default: ./gapcode-connect)
  -h, --help       show this help

Examples:
  ./install.sh
  ./install.sh --setup
  ./install.sh --setup --force
EOF
}

die() {
  echo "install: $*" >&2
  exit 1
}

while (($# > 0)); do
  case "$1" in
    --setup)
      setup=1
      ;;
    --force)
      force=1
      ;;
    --output)
      (($# >= 2)) || die "--output requires a path"
      output="$2"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown option: $1 (use --help)"
      ;;
  esac
  shift
done

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
if [[ -z "$output" ]]; then
  output="$repo_root/gapcode-connect"
elif [[ "$output" != /* ]]; then
  output="$repo_root/$output"
fi

go_cmd="$(command -v go || true)"

install_go() {
  if [[ -n "$go_cmd" ]]; then
    return
  fi

  case "$(uname -s)" in
    Linux)
      if command -v apt-get >/dev/null 2>&1 && command -v sudo >/dev/null 2>&1; then
        echo "Go is not installed; installing it with apt..."
        sudo apt-get update
        sudo apt-get install -y golang-go
      else
        die "Go is missing. Install it with: sudo apt-get update && sudo apt-get install -y golang-go"
      fi
      ;;
    Darwin)
      if command -v brew >/dev/null 2>&1; then
        echo "Go is not installed; installing it with Homebrew..."
        brew install go
      else
        die "Go is missing. Install Homebrew or Go 1.25+, then rerun ./install.sh"
      fi
      ;;
    *)
      die "Go is missing. Install Go $go_version+ for your operating system, then rerun ./install.sh"
      ;;
  esac

  go_cmd="$(command -v go || true)"
  [[ -n "$go_cmd" ]] || die "Go installation finished, but the go command is still not on PATH"
}

install_go
[[ -x "$go_cmd" ]] || die "Go compiler was not found"

mkdir -p "$(dirname "$output")"
echo "Building GapCode Connect..."
(cd "$repo_root" && GOTOOLCHAIN="${GOTOOLCHAIN:-go${go_version}+auto}" GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}" "$go_cmd" build -tags no_googlechat -trimpath -o "$output" ./cmd/cc-connect)
chmod 0755 "$output"
echo "Built: $output"

gapcode_cmd="$(command -v gapcode || true)"
if [[ -z "$gapcode_cmd" ]]; then
  echo "Warning: the 'gapcode' command is not on PATH." >&2
  echo "Install and authenticate GapCode from https://gapgpt.app/gapcode before starting the connector." >&2
fi

if ((setup)); then
  [[ -n "$gapcode_cmd" ]] ||
    die "cannot start setup until the 'gapcode' command is available on PATH"
  setup_args=(setup)
  ((force)) && setup_args+=(--force)
  "$output" "${setup_args[@]}"
else
  echo
  echo "Next steps:"
  echo "  $output setup       # configure Telegram and/or Bale"
  echo "  $output              # start the connector after setup"
fi
