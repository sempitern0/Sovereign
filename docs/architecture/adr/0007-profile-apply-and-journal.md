# ADR-0007 — Profile apply, local journal and apply lock

Status: accepted (2026-11-11), H3-3.

## Context

H3 requires installing the essential profile with one consent, visible partial failures and a local record that survives interruptions (PLAN §2 and §10). Until now `apply` handled one application and kept no state.

## Decision

1. **`apply --profile <id>`** prints the same plan as `plan` (with scope, provenance and dependency notices), then installs every *required* application whose inspected state is `not_installed`, in the plan's topological order.
   - One `--yes` authorizes exactly the applications listed in that plan. Without it the command prints what would be installed and exits 3 without changes.
   - A failure does not stop the run. An application whose dependency failed is reported as `blocked` and not attempted.
   - Optional, pending, unsupported or non-allowlisted applications are `skipped`. A required application whose provider is missing or whose inspection failed counts as a failure.
   - Exit code 0 only if nothing failed or was blocked; otherwise 1. Installed applications are kept: re-running the same command is the recovery path.
   - Machine-scope packages keep their ADR-0006 rule (attended terminal); in an unattended run they fail individually.
2. **Journal** (`internal/state`): `journal.json` in a per-user state directory (`%LOCALAPPDATA%\Sovereign\state` on Windows; `$XDG_STATE_HOME/sovereign` or `~/.local/state/sovereign` elsewhere), directory 0700, file 0600.
   - One entry per application and platform: provider, package ID, scope, result (`installed`, `already_installed`, `failed`), whether it changed the host, a short **error class** and an RFC 3339 UTC timestamp.
   - It never stores raw package-manager output, host names, user names or paths: provider errors can contain `C:\Users\<name>`, so only classes such as `install_failed`, `timeout` or `admin_terminal_required` are persisted.
   - Writes are atomic (temporary file in the same directory, fsync, rename). An unreadable journal is moved aside as `journal.json.corrupt-<unix>`, never deleted.
   - The journal is informational. Package managers stay the source of truth and every run re-inspects them; a journal write failure is a warning and never changes the exit code.
   - Both `apply --app` and `apply --profile` record results; a refused consent records nothing.
3. **Apply lock**: `apply.lock` in the same directory, created exclusively and holding the PID. Any consented `apply` takes it. A second run fails with the lock path. A lock whose process no longer exists (signal 0 on Unix, `OpenProcess` on Windows) is replaced automatically.

## Consequences

- H3-4 (`status`) can compare the journal with live inspection without new storage.
- The journal does not drive decisions; deleting it is always safe.
- No new dependencies: locking and process checks use the Go standard library and `syscall`.
