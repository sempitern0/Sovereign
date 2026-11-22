# ADR-0006 — Install scope and package provenance

Status: accepted (2026-11-11), decided by the maintainer during H3-1 (VLC). Refines ADR-0003/0004.

## Context

- Until H2 every installation was user scope (`winget --scope user`, `flatpak --user`).
- VLC's WinGet manifest (3.0.24) offers full installers only in **machine** scope (`elevatesSelf`). With `--scope user` WinGet can only fall back to a portable zip, without Start menu entries or file associations.
- VLC on Flathub is a **community package**: Flathub labels it "not verified by, affiliated with, or supported by VideoLAN", although it builds from the official `download.videolan.org` tarball with a pinned SHA-256.
- LocalSend's WinGet and Flathub packages both install binaries published by the LocalSend project (pinned hashes in the channel manifests).

## Decision

1. Every verified catalog platform entry and every allowlist entry declares:
   - `scope`: `user` or `machine`. `machine` is allowed only on Windows/WinGet. Flatpak stays `user` (a system install would need root).
   - `provenance`: `upstream` when the channel installs binaries published by the upstream project; `community` when a third party builds or maintains the package. The value is decided from the channel manifest, not from badges.
2. Both fields are part of the field-by-field match between catalog and allowlist (Go and Python CI).
3. Machine scope is **never silent**: the plan and the consent refusal state that Windows will ask for administrator approval, and `apply` installs a machine-scope package only from an attended interactive terminal. Unattended or `--non-interactive` runs fail before calling the package manager. The administrator prompt itself belongs to Windows; Sovereign does not elevate.
4. Community provenance is shown in the plan and on consent refusal ("community-built package, not endorsed by the upstream project"). It is accepted per application by the maintainer and recorded in the catalog notes.
5. **Declared provider dependencies** (added in H3-2, LibreOffice). When a package manifest declares other packages that the provider installs on its own (WinGet `PackageDependencies`), the entry lists them in `provider_dependencies`, matched field by field between catalog and allowlist. The plan and consent refusal show them, narrowed to the device's architecture when planning for the current device. Sovereign does not install them separately and does not pass `--skip-dependencies`. Accepting them is a per-app maintainer decision, like machine scope.

## Consequences

- VLC is allowlisted on Windows (machine scope, upstream binaries) and Linux (user scope, community build). Real-device acceptance is still required before merge.
- LibreOffice is allowlisted on Windows (machine scope, upstream MSI, declared Microsoft Visual C++ Redistributable dependency) and Linux (user scope, Flathub build maintained by LibreOffice developers from the official tag, `upstream`).
- The invariant "user scope only" becomes "user scope unless the allowlist entry declares machine scope, which requires an attended terminal and the OS's own approval".
- Future apps must record both fields from evidence; unknown means not verified.

## Amendment 2026-11-22 — unspecified scope, community by default, acceptance after merge

Maintainer decision to keep the project moving without weakening consent:

- `scope: unspecified` (Windows/WinGet only) records that the manifest declares no `Scope`. Sovereign passes no `--scope`, lets WinGet choose the installer and treats the package like `machine`: the plan warns that Windows *may* ask for administrator approval and `apply` requires an attended terminal. First use: CoMaps on Windows.
- Community provenance and declared provider dependencies no longer need a per-app decision; they are always shown as notices before consent.
- Unmaintained packages (no upstream update for over a year or an end-of-life runtime) still need an explicit maintainer decision recorded in the catalog notes. First use: Kiwix Desktop on Flathub.
- Several applications may be allowlisted in one PR. Real-device acceptance may follow the merge; until it is recorded in STATUS the app is *pending real acceptance*, and a failure reverts it or sets the platform back to `pending`.
