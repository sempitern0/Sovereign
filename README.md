# Sovereign

**Local-first, interoperable open-source suite for Windows, Linux and Android.**

Sovereign aims to prepare everyday devices with declarative catalogs, profiles and trusted package managers. Local apps and data must remain usable without Internet or a NAS. [Multisuite](https://github.com/sempitern0/Multisuite) is an **optional future connector**, not a dependency.

## Current state: H0 + H1 read-only foundation

- Go CLI: `suitectl doctor [--json]`, `catalog list`, `profiles list`, `plan --profile essential [--json]`, and `version`.
- Versioned YAML application catalog and profiles, JSON Schema v1 with semantic validation.
- LocalSend, VLC and LibreOffice registered as **pending**: package IDs and publishers still require independent verification.
- Dependency-aware, platform-specific deterministic planning with blocked/unverified status and explicit alternative choices.
- Windows/Linux CI, Go tests and negative catalog tests.
- `apply` and `verify` are **disabled** with a nonzero exit code. `plan` never installs or inspects the installed-package state.

**This is not yet an end-user installer.** No app is installed, no remote service contacted and no privileges requested by H0/H1.

## Development

Requires Go 1.23+ and Python 3.11+ for catalog validation:

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
```

The read-only developer bootstraps are `./bootstrap/setup.sh --json` and `./bootstrap/setup.ps1 -Json`. A release installer with prebuilt binaries will follow in later milestones.

## H1 plan states

- `pending_verification`: package identity/source has not been verified; cannot install.
- `unsupported`: no supported platform implementation.
- `provider_unavailable`: verified definition but local package manager missing; cannot install.
- `inspection_required`: verified definition with locally detected provider; does **not** mean installed or ready to install.
- `optional`: visible but deliberately not auto-selected.

A plan carries no private metadata and can be generated entirely offline. Cross-platform estimates cannot detect package managers on a different device.

## Architecture

- `cmd/suitectl`, `internal/catalog`, `internal/planner`, `internal/diagnostics`: CLI and testable read-only core.
- `catalog/apps`, `profiles`, `schemas`: declarative definitions and contracts.
- `tools/validate_catalog.py`: independent catalog validation.
- `bootstrap`: read-only entrypoints.
- `integrations/multisuite`: opt-in connector boundary for later work.
- `android`: guided onboarding planned; no silent APK installation.

The repository's [technical plan](PLAN_TECNICO_SUITE_LIBRE.md) is authoritative. Development instructions: [AGENTS.md](AGENTS.md). Security: [SECURITY.md](SECURITY.md). ADRs: [0001](docs/architecture/adr/0001-core-stack.md), [0002](docs/architecture/adr/0002-planning-contract.md).

## Delivery sequence

H0 + H1 read-only foundation → H2 consent-gated verified WinGet/Flatpak installers → H3 portable data → H4 guided Android → H5 optional Multisuite → H6 offline bundles → H7 signed releases.

MIT license; third-party apps retain their respective licenses.
