# AGENTS.md — Sovereign

## Starting a development iteration
1. Read `README.md`, `PLAN_TECNICO_SUITE_LIBRE.md`, `docs/architecture/adr/0001-core-stack.md`, `docs/architecture/adr/0002-planning-contract.md`, and current code/CI.
2. Check branch, working tree and related PRs. Preserve concurrent changes.
3. Implement the smallest safe, tested change in a branch/PR unless explicitly asked to extend the active PR. Do not merge automatically.
4. Record actual tests and limitations. Do not invent IDs, versions, API availability or success claims.

## Security invariants
- `doctor`, `catalog list`, `profiles list`, and `plan` are read-only. Only `verify --app localsend` and explicit `apply --app localsend --yes` are allowed in H2; all other installs fail closed.
- Never install `pending` or `unsupported` packages. Never treat `inspection_required` as proof of an installed package.
- Multisuite is an optional, future connector. Local use cannot depend on a network, account or NAS.
- Never store secrets or user files in Git/logs. Never use remote script execution pipelines, silent elevation, unsolicited telemetry or destructive operations.
- Keep YAML and JSON Schema versioned; reject invalid references, missing alternatives, unknown keys and dependency cycles.
- No installation side effects in test runs; use mocked providers for all installation tests. Treat JSON status 0 as successful planning only.

- Bootstrap: use pinned official Go tar/zip archives, verify SHA-256 before extraction and install only under the user's data directory, with no elevation or permanent PATH change. Do not make Go a runtime dependency for future prebuilt release binaries.

## Quality checks
```bash
go test ./...
go vet ./...
python3 -m pip install -r tools/requirements.txt
python3 tools/validate_catalog.py
python3 -m unittest discover -s tests -v
gofmt -l cmd internal
sh -n bootstrap/setup.sh
# Windows CI also parses and runs bootstrap/setup.ps1 with existing Go
```

## Next step: H2 acceptance
Review exact LocalSend source and package identities in `docs/H2_LOCALSEND.md`. Test WinGet and Flatpak on disposable real environments, including first install, `verify`, second `apply`, missing source, cancelled prompts, network loss and recovery. Keep Android guided and Multisuite opt-in. Do not enable additional apps before this acceptance.
