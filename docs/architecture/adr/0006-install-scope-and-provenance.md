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

## Consequences

- VLC is allowlisted on Windows (machine scope, upstream binaries) and Linux (user scope, community build). Real-device acceptance is still required before merge.
- The invariant "user scope only" becomes "user scope unless the allowlist entry declares machine scope, which requires an attended terminal and the OS's own approval".
- Future apps must record both fields from evidence; unknown means not verified.
