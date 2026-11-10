# Sovereign

**Local-first, interoperable open-source suite for Windows, Linux and Android.**

Sovereign aims to prepare everyday devices with declarative catalogs, profiles and trusted package managers. Local apps and data must remain usable without Internet or a NAS. [Multisuite](https://github.com/sempitern0/Multisuite) is an **optional future connector**, not a dependency.

## Current state: H0 foundation

- Go CLI: `suitectl doctor [--json]` and `suitectl version` (read-only).
- Versioned YAML application catalog and profiles, JSON Schema v1 with semantic validation.
- LocalSend, VLC and LibreOffice registered as **pending**: package IDs and publishers still require independent verification.
- Windows/Linux CI, Go tests and negative catalog tests.
- `plan`, `apply` and `verify` deliberately return a nonzero exit code without modifying the device.

**This is not yet an end-user installer.** No app is installed, no remote service called and no privileges requested by H0.

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
```

The read-only developer bootstraps are `./bootstrap/setup.sh --json` and `./bootstrap/setup.ps1 -Json`. A release installer with prebuilt binaries will follow in later milestones.

## Architecture

- `cmd/suitectl`, `internal`: CLI and testable core.
- `catalog/apps`, `profiles`, `schemas`: declarative definitions and contracts.
- `tools/validate_catalog.py`: validation before any catalog entry is trusted.
- `bootstrap`: H0 read-only entrypoints.
- `integrations/multisuite`: opt-in connector boundary for later work.
- `android`: guided onboarding planned; no silent APK installation.

The repository's [technical plan](PLAN_TECNICO_SUITE_LIBRE.md) is authoritative. Development instructions: [AGENTS.md](AGENTS.md). Security: [SECURITY.md](SECURITY.md).

## Delivery sequence

H0 foundation → H1 deterministic read-only planner → H2 consent-gated verified WinGet/Flatpak installers → H3 portable data → H4 guided Android → H5 optional Multisuite → H6 offline bundles → H7 signed releases.

MIT license; third-party apps retain their respective licenses.
