#!/bin/sh
# Single entry point; Go is downloaded only if absent or incompatible.
set -eu
dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
exec sh "$dir/bootstrap/setup.sh" "$@"
