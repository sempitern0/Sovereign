# ADR-0003 — H2: adaptador limitado a LocalSend

Status: experimental / pending real-device QA (2026-11-10). Allowlist location and provider handling refined by [ADR-0004](0004-embedded-catalog-and-allowlist.md).

## Context and decision

To validate installation end to end, enable only reviewed LocalSend identities on WinGet (Windows) and Flatpak (Linux). Keep the catalog declarative, but enforce an additional application/ID/source allowlist in the H2 execution adapter. All other applications remain read-only or pending.

`suitectl apply --app localsend --yes` is single-app, not profile-wide. This prevents accidental rollout of unverified dependencies. The command checks local state, refuses an unreviewed source, executes the package manager without an intermediary shell, then verifies state. There is no implicit privilege escalation, remote registration, auto-accept of package terms, or update operation. Other apps and Android cannot be installed.

`verify` is a read-only native package-manager query. Ambiguous results fail closed. Success does not prove an authentic executable; package managers remain responsible for artifact verification. Multi-application transactions, state files, audit journal, broad distro support, and offline packages are deliberately deferred.

The design includes a mockable process runner and negative tests. Validate behavior on test machines before widening the allowlist.
