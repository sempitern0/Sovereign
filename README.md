# Sovereign

**Local-first, interoperable open-source suite for Windows, Linux and Android.**

Sovereign aims to prepare everyday devices with declarative catalogs, profiles and trusted package managers. Local apps and data must remain usable without Internet or a NAS. [Multisuite](https://github.com/sempitern0/Multisuite) is an **optional future connector**, not a dependency.

## Current state: H0 + H1 + first H2 installation slice

- Go CLI: `suitectl doctor [--json]`, `catalog list`, `profiles list`, `plan --profile essential [--json]`, and `version`.
- Versioned YAML application catalog and profiles, JSON Schema v1 with semantic validation.
- LocalSend verified for Windows/WinGet and Linux/Flatpak with pinned package IDs and official source references; VLC and LibreOffice remain pending.
- Dependency-aware, platform-specific deterministic planning with blocked/unverified status and explicit alternative choices.
- Windows/Linux CI, Go tests and negative catalog tests.
- `apply --app localsend --yes` and `verify --app localsend` are supported on the host OS only. Other apps and Android remain blocked; `plan` is unchanged and read-only.
- The catalog, profiles and the reviewed install allowlist are embedded in the binary, so `suitectl` works from any directory without a checkout. `--root <checkout>` reads a development copy of the catalog and profiles; the allowlist is always the embedded one ([ADR-0004](docs/architecture/adr/0004-embedded-catalog-and-allowlist.md)).
- Blocked plan entries include a `next_step` with the exact action to take (for example, how to install a missing package manager).

**H2 is experimental**: apply invokes the native package manager with user consent and can install LocalSend. All other H0/H1 commands remain read-only. No release bundles or real-device QA have been completed.

## Development

The bootstrap scripts automatically reuse Go 1.23+ or download the SHA256-pinned Go 1.27.3 archive into the **current user's** application data directory when missing. No administrator rights or system-wide installation are required. On Windows, the verified per-user toolchain is added to the **user PATH** by default (disable with `-NoPath`). Reopen already-running terminals to refresh their PATH; a child script cannot change its parent's environment. Python 3.11+ is needed only for development-time catalog validation:

```bash
# Linux, from a clone of the H2 branch
./bootstrap/setup.sh --json
# Optional: --offline forbids toolchain downloads; dependencies must already be cached
```

```powershell
# Windows, from a clone of the H2 branch
.\\bootstrap\\setup.ps1 -Json
# Optional: -Offline forbids toolchain downloads; dependencies must already be cached
```

The bootstrap currently runs **doctor** and does not itself install applications. It requires an Internet connection for first-time toolchain/dependency downloads. Sovereign release binaries will not require Go.

For developers, equivalent direct commands:

```bash
git clone https://github.com/sempitern0/Sovereign.git
cd Sovereign
python3 -m pip install -r tools/requirements.txt
python3 tools/validate_catalog.py
python3 -m unittest discover -s tests -v
go test ./...
go run ./cmd/suitectl doctor --json
go run ./cmd/suitectl profiles list
go run ./cmd/suitectl catalog list --platform windows
go run ./cmd/suitectl plan --profile essential --platform linux --json
# Test catalog edits without rebuilding: read the checkout instead of the embedded copy
go run ./cmd/suitectl plan --profile essential --root . --json
# Only after reviewing the plan, sources and local prerequisites:
go run ./cmd/suitectl verify --app localsend
go run ./cmd/suitectl apply --app localsend --yes
```

Bootstrap security: Go 1.27.3 archive hashes are pinned from [the official Go release list](https://go.dev/dl/) for Windows/Linux x86-64 and ARM64. The bootstrap downloads HTTPS archives, validates SHA-256, extracts into a per-user directory and executes without privilege elevation. Windows updates only the user PATH, never the machine PATH; Linux does not change PATH. See [the bootstrap runbook](docs/BOOTSTRAP_GO.md). A prebuilt release installer will follow in later milestones.

## H2 first installation

Only **LocalSend** is allowlisted, in the reviewed [`catalog/install-allowlist.yaml`](catalog/install-allowlist.yaml) compiled into the binary. `apply --app localsend` first checks installed state: if already installed, returns success without consent or changes. If missing, explicit `--yes` is required before installation. An ambiguous inspection always fails closed. `verify` is read-only and returns nonzero if not installed or if inspection is ambiguous. A second successful `apply` leaves the existing installation unchanged. On Linux, configure the official **Flathub user remote** yourself first (no repository additions or privileged operations by Sovereign); if it is missing, `apply` prints the exact command: `flatpak remote-add --user --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo`. On Windows, WinGet uses the **winget** source, the exact identifier, user scope and the no-upgrade option. In a terminal, the package manager runs attached to it: you see its progress and answer its agreement prompts yourself. Without a terminal (scripts, CI, or input/output redirected to `/dev/null`/`NUL`), or with `--non-interactive`, output is captured and WinGet runs with `--disable-interactivity`, so it fails rather than waiting on an invisible prompt. Inspection and installation have timeouts (2 and 30 minutes). Neither mode uses a shell or downloads a remote script.

Sources and operating checklist: [H2 LocalSend runbook](docs/H2_LOCALSEND.md). The current smoke tests simulate the package managers; run actual installations only on a disposable or test device before approving H2 for general use.

The default `plan` on the current operating system overlays **read-only** local state for allowlisted applications: `installed`, `not_installed`, or `inspection_failed`. Cross-platform plans and `--no-inspect` retain the deterministic H1-only result. The plan never invokes installation.

## H1 plan states

- `pending_verification`: package identity/source has not been verified; cannot install.
- `unsupported`: no supported platform implementation.
- `provider_unavailable`: verified definition but local package manager missing; cannot install.
- `inspection_required`: verified definition with locally detected provider; no inspection performed (cross-platform, `--no-inspect`, or unsupported H2 adapter).
- `installed` / `not_installed`: current device checked through the local package manager (read-only).
- `inspection_failed`: ambiguous provider output or inspection error; no installation is authorized.
- `optional`: visible but deliberately not auto-selected.

A plan carries no private metadata and can be generated entirely offline. Cross-platform estimates cannot detect package managers on a different device.

## Architecture

- `cmd/suitectl`, `internal/catalog`, `internal/planner`, `internal/diagnostics`, `internal/install`: CLI and testable read-only core.
- `catalog/apps`, `profiles`, `schemas`: declarative definitions and contracts.
- `tools/validate_catalog.py`: independent catalog validation.
- `bootstrap`: read-only entrypoints.
- `integrations/multisuite`: opt-in connector boundary for later work.
- `android`: guided onboarding planned; no silent APK installation.

The repository's [technical plan](PLAN_TECNICO_SUITE_LIBRE.md) is authoritative. Development instructions: [AGENTS.md](AGENTS.md). Security: [SECURITY.md](SECURITY.md). ADRs: [0001](docs/architecture/adr/0001-core-stack.md), [0002](docs/architecture/adr/0002-planning-contract.md), [0003](docs/architecture/adr/0003-first-provider.md), [0004](docs/architecture/adr/0004-embedded-catalog-and-allowlist.md).

## Delivery sequence

H0 + H1 read-only foundation → H2 first verified LocalSend adapter (in review) → H3 portable data → H4 guided Android → H5 optional Multisuite → H6 offline bundles → H7 signed releases.

MIT license; third-party apps retain their respective licenses.
