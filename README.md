# Sovereign

**Local-first, interoperable open-source suite for Windows, Linux and Android.**

Sovereign aims to prepare everyday devices with declarative catalogs, profiles and trusted package managers. Local apps and data must remain usable without Internet or a NAS. [Multisuite](https://github.com/sempitern0/Multisuite) is an **optional future connector**, not a dependency.

## Current state: H0–H3 implemented, H4 release pipeline merged (publication pending)

Live status, limitations and next action: [docs/STATUS.md](docs/STATUS.md).

- Go CLI: `suitectl doctor [--json]`, `catalog list`, `profiles list`, `plan --profile essential [--json]`, and `version`.
- Versioned YAML application catalog and profiles, JSON Schema v1 with semantic validation.
- LocalSend, VLC and LibreOffice have reviewed Windows/WinGet and Linux/Flatpak identities; CoMaps (offline maps) has a reviewed Linux/Flathub identity and stays pending on Windows; Kiwix (offline Wikipedia) has a reviewed Windows/WinGet identity (Kiwix JS Electron, user scope) and stays pending on Linux. **Real-device first-install acceptance is still pending for VLC, LibreOffice, CoMaps and Kiwix**.
- Dependency-aware, platform-specific deterministic planning with blocked/unverified status and explicit alternative choices.
- Windows/Linux CI, Go tests and negative catalog tests.
- `apply --app <id> --yes` and `verify --app <id>` work on the host OS for allowlisted apps only (LocalSend, VLC, LibreOffice; CoMaps on Linux; Kiwix on Windows). Setup apps not yet verified on the current platform are skipped and reported as "not yet available". Other apps and Android remain blocked; `plan` is read-only.
- The catalog, profiles and the reviewed install allowlist are embedded in the binary, so `suitectl` works from any directory without a checkout. `--root <checkout>` reads a development copy of the catalog and profiles; the allowlist is always the embedded one ([ADR-0004](docs/architecture/adr/0004-embedded-catalog-and-allowlist.md)).
- `apply --profile essential` installs everything the profile still needs with **one** consent covering the plan it prints first, keeps going after a failure, skips applications whose dependencies failed and ends with a per-app summary (exit 1 if incomplete; re-run to finish). Results go to a small local journal without personal data, and a lock prevents two runs at once ([ADR-0007](docs/architecture/adr/0007-profile-apply-and-journal.md)).
- `status [--profile essential] [--json]` compares, per application, what the profile wants, what the package manager reports now and the last result in the journal, and points out drift (for example an app removed outside Sovereign). It is read-only.
- Blocked plan entries include a `next_step` with the exact action to take (for example, how to install a missing package manager).

`apply` invokes the native package manager with user consent and can install LocalSend; real-device testing was confirmed by the maintainer on 2026-11-11. All other commands are read-only. There is no published release yet: run from a clone with the bootstrap below; the release pipeline is prepared for a future version tag.

## Privacy and knowledge offline (initial integration)

- Kiwix and CoMaps are catalog entries; they are installable only once their package is verified and allowlisted. Wikipedia ZIM libraries and offline regional maps require independent downloads, storage and integrity verification. See [Offline library](docs/OFFLINE_LIBRARY.md).
- Sovereign only catalogs free/open-source software. WinHance (PolyForm Shield, not OSI) was removed from the catalog.
- Windows can use `sovereign.ps1 privacy audit` (or the new wizard menu option) for a **read-only** inventory of selected privacy settings and services. No process, service, telemetry policy, scheduled task, Defender or Windows Update component is disabled. See [Windows privacy](docs/WINDOWS_PRIVACY.md).
- A future opt-in privacy plan will require safe rollback and explicit approval for each compatible setting.

## Una sola preparación

Sovereign está diseñado para uso personal en equipos **nuevos o existentes**. No hay perfiles entre los que elegir: el asistente prepara una única configuración, comprueba las aplicaciones existentes y solo instala las que falten.

```powershell
# Windows: ejecutar desde la carpeta del proyecto
.\sovereign.ps1
# Alternativa sin menú: consultar los cambios necesarios
.\sovereign.ps1 setup
# Después de revisar, autorizar
.\sovereign.ps1 setup --yes
```

```sh
# Linux
sh sovereign.sh
sh sovereign.sh setup
sh sovereign.sh setup --yes
```

La primera opción del asistente es **Preparar este equipo**. Muestra el plan antes de cambiar nada, permite cancelar y exige escribir `SI` para instalar. Si está todo listo, repetir la preparación no vuelve a instalar aplicaciones. `status` presenta un resumen del entorno. El resto de funciones —aplicaciones adicionales, privacidad Windows, mapas y Wikipedia— permanecen disponibles como opciones separadas, sin instalar elementos todavía no aprobados.

La clave interna `essential` se conserva solo como identificador estable de estado y para compatibilidad con la CLI y el journal existente. `--profile essential` sigue funcionando en scripts antiguos, pero **no hay selección de perfiles en la experiencia del usuario**.

Sovereign desde código fuente prepara Go si falta; no instala Git. Una futura release binaria permitirá ejecutar sin ninguno de ellos.

## Arranque simplificado

Desde una copia del repositorio, usar `sovereign.ps1` (Windows) o `sovereign.sh` (Linux) sin argumentos. Los lanzadores reutilizan Go o preparan una versión verificada en la carpeta del usuario si hace falta; no requieren instalar Go manualmente.

- `setup`: vista previa y comprobación, sin cambios.
- `setup --yes`: preparación automática de la configuración aprobada; instala solo lo que falta, con las autorizaciones necesarias del gestor de paquetes.
- `status`: salud del entorno.
- `privacy audit`: auditoría de privacidad Windows, sin modificar servicios.
- `--verbose` y `--json`: opciones técnicas para detalles y automatización (en comandos compatibles).

**Offline real:** el lector Kiwix, los mapas y sus archivos de datos todavía necesitan un flujo de instalación y descarga verificado; no se presentan como disponibles hasta completarlo. Las futuras releases precompiladas no requerirán Git ni Go.

## Download a release

**Release status (2026-11-19):** the reproducible build pipeline is merged, but no version has been published yet. Use the source checkout instructions below until a signed release is published. Do not treat the examples below as existing downloads.

Each tagged release ships `suitectl` for Windows and Linux (x86-64 and ARM64) with no runtime to install: `.zip` for Windows, `.tar.gz` for Linux, an SPDX SBOM per archive and `SHA256SUMS`. Builds are reproducible and their provenance is signed by this repository's release workflow ([ADR-0008](docs/architecture/adr/0008-reproducible-release.md)).

Verify **before** running. The checksum detects a damaged download; the signed attestation proves it was built here (a checksum downloaded from the same place does not authenticate the file on its own):

```bash
# Linux
sha256sum -c SHA256SUMS --ignore-missing
gh attestation verify suitectl_<version>_linux_amd64.tar.gz --repo sempitern0/Sovereign
tar -xzf suitectl_<version>_linux_amd64.tar.gz && ./suitectl_<version>_linux_amd64/suitectl version
```

```powershell
# Windows (PowerShell)
$file = "suitectl_<version>_windows_amd64.zip"
$expected = (Select-String -Path SHA256SUMS -Pattern ([regex]::Escape($file))).Line.Split(" ")[0]
if ((Get-FileHash $file -Algorithm SHA256).Hash.ToLower() -ne $expected) { throw "Checksum mismatch" }
gh attestation verify $file --repo sempitern0/Sovereign
Expand-Archive $file; .\suitectl_<version>_windows_amd64\suitectl_<version>_windows_amd64\suitectl.exe version
```

`gh` is the GitHub CLI. The binaries are not signed with a Windows code-signing certificate, so SmartScreen may warn on first run.

## Development

The bootstrap scripts automatically reuse Go 1.23+ or download the SHA256-pinned Go 1.27.3 archive into the **current user's** application data directory when missing. No administrator rights or system-wide installation are required. On Windows, the verified per-user toolchain is added to the **user PATH** by default (disable with `-NoPath`). Reopen already-running terminals to refresh their PATH; a child script cannot change its parent's environment. Python 3.11+ is needed only for development-time catalog validation:

```bash
# Linux, from a clone
./bootstrap/setup.sh --json
# Optional: --offline forbids toolchain downloads; dependencies must already be cached
```

```powershell
# Windows, from a clone
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

## Installing applications

Installable applications are those in the reviewed [`catalog/install-allowlist.yaml`](catalog/install-allowlist.yaml) compiled into the binary: **LocalSend** (Windows and Linux, accepted on real devices) and, with real-device acceptance still pending, **VLC** (Windows in machine scope with an administrator prompt; Linux from Flathub's community build) and **LibreOffice** (Windows in machine scope, WinGet may add the Visual C++ Redistributable; Linux from Flathub). The plan shows each package's install scope and provenance before you consent ([ADR-0006](docs/architecture/adr/0006-install-scope-and-provenance.md)). `apply --app localsend` first checks installed state: if already installed, returns success without consent or changes. If missing, explicit `--yes` is required before installation. An ambiguous inspection always fails closed. `verify` is read-only and returns nonzero if not installed or if inspection is ambiguous. A second successful `apply` leaves the existing installation unchanged. On Linux, configure the official **Flathub user remote** yourself first (no repository additions or privileged operations by Sovereign); if it is missing, `apply` prints the exact command: `flatpak remote-add --user --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo`. On Windows, WinGet uses the **winget** source, the exact identifier, user scope and the no-upgrade option. In a terminal, the package manager runs attached to it: you see its progress and answer its agreement prompts yourself. Without a terminal (scripts, CI, or input/output redirected to `/dev/null`/`NUL`), or with `--non-interactive`, output is captured and WinGet runs with `--disable-interactivity`, so it fails rather than waiting on an invisible prompt. Inspection and installation have timeouts (2 and 30 minutes). Neither mode uses a shell or downloads a remote script.

Sources and operating checklist: [LocalSend runbook](docs/H2_LOCALSEND.md). CI simulates the package managers; each newly allowlisted application is accepted on a real test device before merge.

The default `plan` on the current operating system overlays **read-only** local state for allowlisted applications: `installed`, `not_installed`, or `inspection_failed`. Cross-platform plans and `--no-inspect` retain the deterministic H1-only result. The plan never invokes installation.

## Estados de planificación (uso avanzado)

- `pending_verification`: package identity/source has not been verified; cannot install.
- `unsupported`: no supported platform implementation.
- `provider_unavailable`: verified definition but local package manager missing; cannot install.
- `inspection_required`: verified definition with locally detected provider; no inspection performed (cross-platform, `--no-inspect`, or unsupported H2 adapter).
- `installed` / `not_installed`: current device checked through the local package manager (read-only).
- `inspection_failed`: ambiguous provider output or inspection error; no installation is authorized.
- `optional`: visible but deliberately not auto-selected.

A plan carries no private metadata and can be generated entirely offline. Cross-platform estimates cannot detect package managers on a different device.

## Architecture

- `cmd/suitectl`: CLI. `internal/catalog`, `internal/planner`, `internal/diagnostics`: read-only core. `internal/install`: consent-gated, allowlisted package-manager adapter (WinGet, Flatpak).
- `embed.go`: catalog, profiles and install allowlist compiled into the binary.
- `catalog/apps`, `catalog/install-allowlist.yaml`, `profiles`, `schemas`: declarative definitions and contracts.
- `tools/validate_catalog.py`: independent catalog validation.
- `bootstrap`: per-user, SHA-256-verified Go toolchain setup and `doctor`; it installs no applications (replaced by a verified release download in H4).
- `integrations/multisuite`: design notes for the optional connector (H8); no code yet.
- `android`: guided onboarding (H6); no silent APK installation.

The repository's [technical plan](PLAN_TECNICO_SUITE_LIBRE.md) is authoritative. Development instructions: [AGENTS.md](AGENTS.md) (Claude Code: [CLAUDE.md](CLAUDE.md)). Security: [SECURITY.md](SECURITY.md). ADRs: [0001](docs/architecture/adr/0001-core-stack.md), [0002](docs/architecture/adr/0002-planning-contract.md), [0003](docs/architecture/adr/0003-first-provider.md), [0004](docs/architecture/adr/0004-embedded-catalog-and-allowlist.md), [0005](docs/architecture/adr/0005-roadmap-release-first.md), [0006](docs/architecture/adr/0006-install-scope-and-provenance.md), [0007](docs/architecture/adr/0007-profile-apply-and-journal.md), [0008](docs/architecture/adr/0008-reproducible-release.md).

## Delivery sequence

H0–H2 foundation, planning and first verified installer (done) → **H3 installable essential profile** → H4 first distributable release (no Git or Go needed) → H5 portable data → H6 guided Android → H7 offline kits → H8 optional Multisuite connector → H9 UX and user guides. Order and rationale: [ADR-0005](docs/architecture/adr/0005-roadmap-release-first.md).

Dependencies are vendored: `go build ./cmd/suitectl` works offline. Release artifacts locally: `python3 tools/release.py --version 0.0.0-dev --out dist` (see ADR-0008); publishing happens only from a `v*` tag in CI.

MIT license; third-party apps retain their respective licenses.
