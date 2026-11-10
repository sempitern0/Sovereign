#!/bin/sh
set -eu
# H0 entrypoint is read-only, never fetches remote scripts or installs packages.
if ! command -v go >/dev/null 2>&1; then
  printf '%s\n' 'Go 1.23+ is needed for H0 development. Release binaries are planned for H2.' >&2
  exit 3
fi
project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_dir"
exec go run ./cmd/suitectl doctor "$@"
