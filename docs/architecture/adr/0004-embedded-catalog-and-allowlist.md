# ADR-0004 — Embedded catalog and data-driven install allowlist

Status: proposed (2026-11-11). Refines ADR-0003; does not widen what can be installed.

## Context

- A release binary must work without a Git checkout (plan §11). `suitectl` read
  `catalog/` and `profiles/` relative to `--root .`, so a binary run from any
  other directory failed with `catalog/apps: no YAML manifests`.
- ADR-0003 requires an allowlist *additional* to the catalog. It was implemented
  as LocalSend-specific Go constants repeated in `Resolve`, `Status` and tests, so
  enabling a second reviewed package required new adapter code.
- Package-manager output was captured in full, so a slow install or a WinGet
  prompt could not be seen, and no timeout bounded a stuck process.

## Decision

1. `catalog/apps/*.yaml`, `profiles/*.yaml` and `catalog/install-allowlist.yaml`
   are embedded with `go:embed`. Without `--root`, every command uses the
   embedded copy. `--root <checkout>` remains for development and reads the
   catalog and profiles from disk with the same strict validation (regular files
   only, size limit, unknown keys and duplicate keys rejected).
2. The installation allowlist is a reviewed YAML file that is **only ever read
   from the embedded copy**. `--root` can change what is *planned*, never what
   may be *installed*. Approving a package therefore still requires review and a
   rebuilt binary, exactly like the previous constants.
3. A target is installable only if the catalog entry is `verified` **and**
   application, upstream, platform, provider, package ID and source all match one
   allowlist entry. `Engine.Resolve`, `Status` and `Apply` read the same allowlist
   (defense in depth), so planning inspection and installation cannot disagree.
   Providers stay pinned: WinGet on Windows, Flatpak on Linux. CI (Go and Python)
   fails if an allowlist entry does not match a verified catalog entry.
4. Provider behaviour is generic per provider, so enabling another reviewed
   package is a data change plus acceptance evidence, not new adapter code.
5. Installation runs attached to the terminal when stdin and stdout are real
   terminals: progress and package-manager prompts are visible and answered by
   the user. Detection asks the OS (`ioctl(TCGETS)` on Linux, `GetConsoleMode`
   on Windows) instead of testing for a character device, because `/dev/null`
   and `NUL` are character devices too. Otherwise, or with `--non-interactive`,
   output is captured and WinGet runs with `--disable-interactivity`, so an
   unattended run fails instead of blocking invisibly. Every provider call has a
   deadline (inspection 2 min, installation 30 min).
6. Plans add an optional `next_step` field (additive; plan schema stays 1) for
   blocked or actionable states. Sovereign still never installs package managers
   or adds repositories; it only tells the user the exact command.

## Consequences

- The allowlist still contains only LocalSend for Windows and Linux. Each
  further application needs its own verification, allowlist entry and
  real-device acceptance (AGENTS.md checklist).
- `apply --profile` is deliberately not added here; it is planned for H3
  (ADR-0005). The generic adapter is its prerequisite.
- Editing YAML on disk no longer affects a built binary; developers pass
  `--root .` (or rebuild) to test catalog changes.
