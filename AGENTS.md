# AGENTS.md — Sovereign

## Start any development iteration
1. Read `README.md`, `PLAN_TECNICO_SUITE_LIBRE.md`, `docs/architecture/adr/0001-core-stack.md` and current code/CI.
2. Check branch, working tree and previous PRs. Preserve concurrent work.
3. Implement the smallest verified vertical slice in a separate PR; do not merge automatically.
4. Document actual tests and limitations. No invented package IDs, compatibility guarantees or success claims.

## Invariants
- H0 `doctor` and validators are read-only. `apply` MUST stay disabled until H2 is verified with user consent.
- Never install `pending` or `unsupported` package definitions.
- Multisuite and other services are optional; local files and functions must not depend on NAS availability.
- Never store secrets, user content or credentials in Git or logs. No remote script execution pipelines, silent elevation, unrequested telemetry or destructive operations.
- Keep catalog schemas versioned; use explicit platform support and provider capabilities.
- Prefer small idiomatic Go packages, mocks in tests and platform-independent semantics.

## Required checks
```bash
go test ./...
python3 -m pip install -r tools/requirements.txt
python3 tools/validate_catalog.py
python3 -m unittest discover -s tests -v
gofmt -l cmd internal
```

## Next step (H1)
Implement catalog YAML decoding in Go with an audited, pinned dependency; platform/profile resolver, deterministic read-only plan and JSON output; add tests for invalid data, missing dependencies and unsupported OS. Only after that, implement WinGet/Flatpak with explicit consent and verification.
