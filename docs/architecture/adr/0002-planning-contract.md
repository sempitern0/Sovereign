# ADR-0002 — H1: deterministic read-only planning

Status: accepted for H1 (2026-11-10).

## Decisions
- Decode YAML using the pinned `gopkg.in/yaml.v3 v3.0.1` dependency; reject unknown fields, duplicate mapping keys, multiple documents and oversized/non-regular manifest files.
- Validate cross-references and dependency cycles before planning. The Python JSON Schema validator remains a complementary CI contract. Schema version stays at 1, with optional `depends_on` and `alternatives`.
- A plan is an **assessment**, not an installation command: `inspection_required` is the most positive state. H1 cannot claim installed state, disk sizes, package trust beyond supplied catalog evidence, or imply that a real package manager will succeed.
- Profile-required applications and platform-specific defaults from mutually exclusive alternative groups are included. Optional applications are visible but not selected for installation.
- Dependencies are listed before dependents. Missing dependencies, invalid manifests, missing alternatives and unsupported targets fail closed.
- JSON output is schema-versioned, ordered and contains no timestamps or user/host identifiers. No networks, persistent state or Multisuite access is necessary.
- On the current host, package manager executables are detected through local PATH only. Cross-platform planning assumes no providers are available, avoiding false claims about remote devices.
- `apply` and `verify` remain disabled until H2. A verified package is still not auto-installed.
- Exit codes: 0 successful read-only operation (not a statement of installability); 1 invalid catalog/plan; 2 CLI usage; 3 disabled operation.

## Future work
H2 needs real provider queries, exact package and publisher verification, user-consented installation, audit logs, safe invocations, idempotency and tests of failed/partial operations. Do not infer installed state from a package manager's presence.
