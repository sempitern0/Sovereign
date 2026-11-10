#!/bin/sh
# Sovereign developer bootstrap: use a compatible Go or install an official,
# SHA256-pinned toolchain into this user's data directory (no root/sudo).
set -eu

go_version='1.27.3'
offline=0
json=0
for option in "$@"; do
  case "$option" in
    --json) json=1 ;;
    --offline) offline=1 ;;
    *) printf 'Unknown bootstrap option: %s\n' "$option" >&2; exit 2 ;;
  esac
done

if [ "$(uname -s)" != 'Linux' ]; then
  printf '%s\n' 'This bootstrap supports Linux only; use setup.ps1 on Windows.' >&2
  exit 3
fi

compatible_go() {
  version_text=$(GOTOOLCHAIN=local "$1" version 2>/dev/null) || return 1
  minor=$(printf '%s\n' "$version_text" | sed -n 's/^go version go1\.\([0-9][0-9]*\)\(\.[0-9][0-9]*\)\{0,1\} linux\/.*$/\1/p')
  [ -n "$minor" ] && [ "$minor" -ge 23 ]
}

go_bin=''
if command -v go >/dev/null 2>&1; then
  candidate=$(command -v go)
  if compatible_go "$candidate"; then
    go_bin=$candidate
  else
    printf '%s\n' 'Existing Go is incompatible with go.mod; using a local Go toolchain.' >&2
  fi
fi

if [ -z "$go_bin" ]; then
  case "$(uname -m)" in
    x86_64|amd64)
      go_arch=amd64
      expected_sha256='65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40' ;;
    aarch64|arm64)
      go_arch=arm64
      expected_sha256='a06e99309dbcbdca1a8ccd2282c3ee146e55930886e26b4f1193269587165f49' ;;
    *) printf '%s\n' 'Unsupported Linux architecture for the pinned Go bootstrap.' >&2; exit 3 ;;
  esac

  data_home="${XDG_DATA_HOME:-$HOME/.local/share}"
  case "$data_home" in
    /*) ;;
    *) printf '%s\n' 'XDG_DATA_HOME must be an absolute path.' >&2; exit 3 ;;
  esac
  toolchains="$data_home/sovereign/toolchains"
  install_dir="$toolchains/go$go_version"
  go_bin="$install_dir/bin/go"

  if [ -e "$install_dir" ] && [ ! -x "$go_bin" ]; then
    printf '%s\n' 'A broken Go toolchain already exists; refusing to overwrite it.' >&2
    exit 3
  fi

  if [ ! -x "$go_bin" ]; then
    if [ "$offline" -eq 1 ]; then
      printf '%s\n' 'Go is unavailable and --offline forbids downloading it.' >&2
      exit 3
    fi
    if ! command -v tar >/dev/null 2>&1; then
      printf '%s\n' 'tar is needed to extract the verified Go archive.' >&2
      exit 3
    fi
    if ! command -v sha256sum >/dev/null 2>&1 && ! command -v openssl >/dev/null 2>&1; then
      printf '%s\n' 'A SHA256 verifier (sha256sum or openssl) is required.' >&2
      exit 3
    fi
    umask 077
    mkdir -p "$toolchains"
    staging=$(mktemp -d "$toolchains/.go$go_version.XXXXXXXX") || exit 3
    trap 'rm -rf "$staging"' 0
    trap 'exit 130' 1 2 3 15
    archive="$staging/go.tar.gz"
    url="https://go.dev/dl/go$go_version.linux-$go_arch.tar.gz"
    printf 'Downloading Go %s for linux/%s from go.dev (verified SHA256)...\n' "$go_version" "$go_arch" >&2

    if command -v curl >/dev/null 2>&1; then
      curl --fail --location --silent --show-error --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 20 --max-time 600 --output "$archive" "$url"
    elif command -v wget >/dev/null 2>&1; then
      wget --https-only --quiet --output-document="$archive" "$url"
    else
      printf '%s\n' 'curl or wget is required to download Go securely.' >&2
      exit 3
    fi

    if command -v sha256sum >/dev/null 2>&1; then
      actual_sha256=$(sha256sum "$archive" | cut -d ' ' -f 1)
    else
      actual_sha256=$(openssl dgst -sha256 "$archive" | awk '{print $NF}')
    fi
    if [ "$actual_sha256" != "$expected_sha256" ]; then
      printf '%s\n' 'Go archive SHA256 mismatch. Nothing was installed.' >&2
      exit 4
    fi

    tar -xzf "$archive" -C "$staging"
    if [ ! -x "$staging/go/bin/go" ] || [ -e "$install_dir" ]; then
      printf '%s\n' 'Go archive invalid or destination was created concurrently.' >&2
      exit 4
    fi
    # GNU/BusyBox mv -T avoids accidentally nesting inside a concurrent directory.
    mv -T "$staging/go" "$install_dir"
    printf 'Go %s installed locally at %s\n' "$go_version" "$install_dir" >&2
  fi

  case "$(GOTOOLCHAIN=local "$go_bin" version 2>/dev/null)" in
    "go version go$go_version linux/"*) ;;
    *) printf '%s\n' 'Cached Go toolchain does not match the pinned version.' >&2; exit 4 ;;
  esac
fi

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_dir"
if [ "$json" -eq 1 ]; then
  if [ "$offline" -eq 1 ]; then
    exec env GOTOOLCHAIN=local GOPROXY=off "$go_bin" run ./cmd/suitectl doctor --json
  fi
  exec env GOTOOLCHAIN=local "$go_bin" run ./cmd/suitectl doctor --json
fi
if [ "$offline" -eq 1 ]; then
  exec env GOTOOLCHAIN=local GOPROXY=off "$go_bin" run ./cmd/suitectl doctor
fi
exec env GOTOOLCHAIN=local "$go_bin" run ./cmd/suitectl doctor
