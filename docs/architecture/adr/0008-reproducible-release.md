# ADR-0008 — Reproducible, verifiable release artifacts

Status: accepted (2026-11-13), H4-1.

## Context

H4 must let a person use Sovereign without Git or Go (ADR-0005). PLAN §7 asks for checksums, provenance, a project signature and an SBOM, and warns that a SHA-256 published next to the file does not authenticate it on its own.

## Decision

1. **Vendored dependencies.** `vendor/` is committed (`gopkg.in/yaml.v3` only). Builds work with no network and an empty module cache. CI regenerates `vendor/` from the official proxy and fails on any difference, which ties it to the hashes in `go.sum`.
2. **Reproducible build** (`tools/release.py`, development/release-time Python as in ADR-0001):
   - `CGO_ENABLED=0 go build -mod=vendor -trimpath -buildvcs=false -ldflags "-s -w -buildid= -X main.version=<v> -X main.commit=<sha>"`;
   - targets `linux/amd64`, `linux/arm64`, `windows/amd64`, `windows/arm64`;
   - archives with sorted entries, fixed owner, fixed modes (0755 binary, 0644 docs) and timestamps from `SOURCE_DATE_EPOCH` (default: the commit time); gzip without file name or wall-clock time. Windows ships as `.zip`, Linux as `.tar.gz` (keeps the executable bit).
   - Output depends only on source, Go toolchain, version, commit and epoch. The release workflow builds twice and fails unless `SHA256SUMS` is identical.
3. **Artifacts per release**: one archive per target (binary, README, LICENSE, SECURITY, CHANGELOG), one SPDX 2.3 JSON SBOM per target generated from `go version -m` (main module, Go standard library, each dependency with its purl and the binary's SHA-256), and `SHA256SUMS`.
4. **Signed provenance.** On a `v*` tag, `actions/attest` signs a SLSA build provenance attestation over every file in `SHA256SUMS` with a short-lived Sigstore certificate tied to this repository's workflow. Users verify with `gh attestation verify <file> --repo sempitern0/Sovereign`. This, not the checksum alone, authenticates an artifact.
5. **Workflow hardening** (`.github/workflows/release.yml`):
   - actions pinned by commit SHA (inputs checked in each pinned `action.yml`);
   - `persist-credentials: false`;
   - least privilege per job: build is read-only and also runs on every PR; attest gets `id-token` and `attestations: write`; publish gets `contents: write` and runs only after attestation succeeds;
   - pre-release tags (`vX.Y.Z-rc.N`) publish as GitHub pre-releases.
6. `suitectl version [--json]` reports version, commit, Go version and platform, so a running binary can be matched to its release.

## Consequences

- Publishing is an explicit maintainer action: pushing a `v*` tag. Nothing is published from branches or PRs.
- H4-2 will switch the bootstrap from installing Go to downloading and verifying a release.
- Code signing with an OS-recognised certificate (Authenticode, which affects Windows SmartScreen) is out of scope; it needs a paid certificate and is tracked as a limitation.
