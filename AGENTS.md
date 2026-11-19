# AGENTS.md — Sovereign

Sovereign prepares Windows, Linux and (guided) Android devices with a local-first, open-source suite from a declarative catalog. It must stay useful with no network, account or NAS. Multisuite is an optional, later connector.

## Start of every iteration
1. Read `docs/STATUS.md`: active milestone, what works, known limitations and the **next action**.
2. Check branch, working tree and open PRs. Preserve concurrent changes.
3. Read only what the task needs (map below). Read code and tests before docs when they disagree; record any discrepancy in STATUS.
4. Implement the smallest safe, tested change in its own branch/PR unless explicitly asked to extend the active PR. Never merge automatically.
5. Finish by updating `docs/STATUS.md` (iteration log + next action). Record tests actually run, environment (real or simulated) and limitations. Never invent IDs, versions, API availability or success claims.

## What to read for each task
| Task | Read | Skip unless needed |
| --- | --- | --- |
| Any | `docs/STATUS.md`, this file | — |
| Add/verify an application | Checklist below, `catalog/apps/*.yaml`, `catalog/install-allowlist.yaml`, ADR-0003/0004/0006, `docs/H2_LOCALSEND.md` (reference acceptance) | PLAN beyond §3 |
| Planner / profiles | ADR-0002, PLAN §3–4, `internal/planner`, `profiles/` | install, bootstrap |
| Installer / providers | ADR-0003/0004, PLAN §4 and §7, `internal/install`, `cmd/suitectl` | planner internals |
| Release / bootstrap (H4) | ADR-0001, ADR-0005, PLAN §7–8, `docs/BOOTSTRAP_GO.md`, `bootstrap/`, CI | catalog details |
| Data / backup (H5) | PLAN §5, `docs/recovery/README.md` | providers |
| Android (H6) | PLAN §3 and §7, `android/README.md` | desktop providers |
| Multisuite (H8) | PLAN §6, `integrations/multisuite/README.md`, the Multisuite repo's current docs | everything else |
| Roadmap / scope | PLAN §1, §10–12, ADR-0005 | code |

`PLAN_TECNICO_SUITE_LIBRE.md` is the product source of truth; accepted ADRs and code override it where they differ.

## Security invariants
- `doctor`, `catalog list`, `profiles list`, `plan`, `status` and `verify` are read-only (`status` reads the journal with `state.Store.Peek`, never `Load`). Only `apply --app <id> --yes` and `apply --profile <id> --yes` change the host, and only for allowlisted applications; everything else fails closed.
- Installation requires a `verified` catalog entry that matches `catalog/install-allowlist.yaml` field by field. The allowlist is embedded at build time and must never be read from `--root` or any runtime path. Adding an entry enables real installs: one application per reviewed PR, with real-device acceptance.
- Never install `pending` or `unsupported` packages. Never treat `inspection_required` as proof of an installed package. Ambiguous provider output fails closed.
- Package managers run with structured arguments, never through a shell. No remote-script pipelines, silent elevation, repository/remote additions, auto-accepted agreements, telemetry or destructive operations (no uninstall, cleanup or overwrite by default).
- Install scope is `user` unless the allowlist entry declares `machine` (Windows only, ADR-0006): the plan warns, and `apply` runs it only from an attended terminal so the OS shows its own administrator prompt. Community-built packages (`provenance: community`) are flagged to the user and accepted per app by the maintainer.
- No installation side effects in tests: use simulated providers (`install.Runner`, `newRunner`/`isTerminal`/`detectProviders` in the CLI); CLI tests never touch the real state directory (`newStore`, set in `TestMain`). Treat JSON status 0 as successful planning only.
- Never store secrets, user files, host identifiers or personal paths in Git, logs, plans, the journal or exported profiles. The journal (`internal/state`) records error classes only (`install.ErrorClass`), never raw package-manager output.
- Multisuite is optional. Local use cannot depend on a network, account, VPN or NAS; a connector must degrade to the local flow.
- Keep YAML and JSON Schema versioned; reject invalid references, missing alternatives, unknown keys and dependency cycles.
- Bootstrap: pinned official archives, SHA-256 verified before extraction, installed only under the user's data directory, no elevation. Windows may add only the reviewed toolchain `bin` to the HKCU user PATH idempotently (opt-out `-NoPath`); never the machine PATH. Linux does not persist PATH. Release binaries must not require Go.

## Adding or verifying an application
1. Confirm the official upstream, license and, per platform, the exact package ID and source in the official channel (WinGet community manifest, Flathub page). Record the source URL and `verified_at` date. Never guess IDs. If a page is not reachable, read the channel's manifest repository with git (`microsoft/winget-pkgs` sparse checkout, `flathub/<app-id>`).
2. From the manifest, record `scope` (WinGet installers' `Scope`; Flatpak is always `user`) and `provenance` (`upstream` if the channel installs the project's own binaries, `community` if a third party builds it; Flathub's "unverified/community" notice counts). Also record WinGet `PackageDependencies` in `provider_dependencies`. A machine-only package, a community package or one with provider dependencies needs the maintainer's explicit decision.
3. Set the platform entry to `verified` in `catalog/apps/<id>.yaml` with those fields and evidence in `notes`; leave other platforms `pending`/`unsupported`.
4. Add matching entries to `catalog/install-allowlist.yaml` (application, upstream, platform, provider, package_id, source, scope, provenance and, if any, provider_dependencies).
5. Run the quality checks; Go and Python CI fail if allowlist and catalog differ. Make sure no test uses an approved app with a real runner (use `withProcess` in CLI tests); tests that need an unapproved app create a fictional pending one (`catalogWithPendingApp`).
6. Add simulated-provider tests only if behaviour differs from existing providers.
7. Acceptance on real devices (follow the scenarios in `docs/H2_LOCALSEND.md`): clean install, `verify`, second `apply` unchanged, missing source, cancelled prompt, network loss. Record the result in STATUS before merge.

## Quality checks
```bash
gofmt -l cmd internal embed.go        # must print nothing
go vet ./...
go test ./...
python3 -m pip install -r tools/requirements.txt
python3 tools/validate_catalog.py
python3 -m unittest discover -s tests -v
sh -n bootstrap/setup.sh
git status --short --ignored -- cmd internal   # new source files must not show as ignored ("!!")
python3 tools/release.py --version 0.0.0-dev --out dist   # release dry run (reproducible; never publish from here)
# CI also runs on Windows: go vet/test, bootstrap/setup.ps1 parse and smoke test
```
Catalog edits are embedded at build time: test them with `--root .` or rebuild.

## Definition of done
Code, tests and docs agree; the milestone's acceptance evidence (PLAN §10) is recorded in STATUS; contract changes are versioned or additive; security invariants reviewed; error paths implemented and tested; no secrets in logs; reproducible commands listed; remaining risks explicit. Distinguish *implemented*, *validated with simulated providers* and *validated on real devices*.
