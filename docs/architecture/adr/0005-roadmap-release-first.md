# ADR-0005 — Roadmap: installable profile and distributable release before integrations

Status: accepted (2026-11-11). Supersedes the milestone order in PLAN_TECNICO_SUITE_LIBRE.md v1.0 §10.

## Context

- H0–H2 are complete: H2 installs one allowlisted application (LocalSend) on Windows/WinGet and Linux/Flatpak, and the user confirmed real-device testing on 2026-11-11.
- The product goal is a **local-first suite that a person can prepare on their devices without friction**. In the v1.0 order, a distributable binary arrived only in H7: until then every user needed Git and a Go toolchain, and the only installable profile had one application.
- Multisuite exists today as a private Laravel portal in front of Navidrome. It exposes no API for Sovereign yet, and its private network access (VPN) is still pending (Multisuite roadmap P03). A connector built earlier would target an interface that does not exist.

## Decision

Reorder the remaining milestones so that each one makes the suite more usable **offline and on its own** before any integration:

| New | Milestone | Was |
| --- | --- | --- |
| H3 | Installable essential profile: verified apps one PR at a time, `apply --profile`, local state journal | part of H2 |
| H4 | First distributable release: prebuilt binaries, checksums, SBOM, provenance; bootstrap without Go | H7 (release part) |
| H5 | Data and portability: `BibliotecaPersonal`, profile export/import, tested restore | H3 |
| H6 | Guided Android | H4 |
| H7 | Offline kits | H6 |
| H8 | Optional Multisuite connector, contract-first | H5 |
| H9 | TUI/GUI and end-user guides, if still justified | H7 (UX part) |

Rules that come with the order:

- A milestone starts only when the previous one has acceptance evidence recorded in `docs/STATUS.md`.
- Every newly installable application needs its own verification, allowlist entry and real-device acceptance (ADR-0004), even inside H3.
- H8 starts with a written interface agreed with the Multisuite repository; Sovereign never writes an adapter against an assumed API.

## Consequences

- Historical references to H3–H7 in the CHANGELOG and earlier PRs keep their original meaning; the table above maps them.
- The v1.0 acceptance criteria (PLAN §11) are unchanged; they are simply reached in a different order.
- Delaying Multisuite does not block local use, which already must work without it.
