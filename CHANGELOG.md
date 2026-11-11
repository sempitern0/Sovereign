# Changelog

## Unreleased — H3

- H3-3: `apply --profile` with a single consent over the printed plan, topological order, continuation after failures, `blocked` dependents and a per-app summary (exit 1 if incomplete). New `internal/state`: atomic per-user journal with error classes only (no provider output or paths), corrupt-journal preservation and an apply lock with dead-process detection; `apply --app` uses both (ADR-0007).
- H3-2: verify and allowlist LibreOffice (WinGet `TheDocumentFoundation.LibreOffice`, machine scope, declared Microsoft Visual C++ Redistributable dependency; Flathub `org.libreoffice.LibreOffice`, upstream-maintained). New optional `provider_dependencies`, matched between catalog and allowlist and shown before consent, narrowed to the device architecture. Tests needing an unapproved app now use a fictional pending one.
- H3-1: verify and allowlist VLC (WinGet `VideoLAN.VLC` in machine scope; Flathub `org.videolan.VLC`, community build). Add required `scope` and `provenance` to verified catalog and allowlist entries; machine scope only from an attended terminal; plan and consent refusal show administrator and community notices (ADR-0006).

## H0 + H1 + H2 LocalSend
- Go read-only CLI diagnostic, catalog/profile discovery and deterministic profile planning.
- Strict YAML decoder, cross-reference and cycle checks, dependency ordering, optional alternatives.
- JSON v1 plans, blocked/unverified status, mock device/provider tests and invalid-catalog tests.
- Declarative manifests, schema contracts, CI and security documentation.
- Fix Windows Go CLI discoverability by idempotently adding the verified Go directory to HKCU user PATH, with `-NoPath` opt-out, process activation and tests.
- Per-user self-bootstrapping Go 1.27.3 for Linux and Windows x86-64/ARM64 using SHA256-pinned official archives, offline fail-closed mode and no permanent PATH changes.
- Docs: reorder milestones so an installable profile (H3) and a distributable release (H4) come before data, Android, offline kits and the optional Multisuite connector (H8), see ADR-0005; add `docs/STATUS.md` as the single status/next-action record and `CLAUDE.md`; restructure AGENTS.md with a per-task reading map, app checklist and definition of done; align plan, security, recovery, Android and Multisuite docs with the current code and Multisuite's real state.
- Embed catalog, profiles and install allowlist in the binary so `suitectl` runs outside a checkout; `--root` reads a development copy (ADR-0004).
- Replace LocalSend-specific adapter constants with a reviewed, embedded allowlist and generic WinGet/Flatpak handling; Go and Python CI cross-check it against verified catalog entries.
- Show package-manager progress and prompts on a real terminal (OS-level detection, so `/dev/null`/`NUL` count as unattended); disable WinGet prompts when unattended or with `apply --non-interactive`; add inspection/installation timeouts.
- Add `next_step` guidance to blocked plan entries and print the exact Flathub remote command or package-manager setup hint on failure.
- First allowlisted, opt-in LocalSend installer and read-only verification for WinGet/Flatpak with mocked acceptance, failure and idempotency tests. Other apps remain blocked.
