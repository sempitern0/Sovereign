# Security policy

Report vulnerabilities privately using GitHub Security Advisories if enabled; otherwise contact the maintainer privately. Do not post credentials or exploitable details in public issues.

Current scope (H2–H3): Sovereign can install only applications in the reviewed allowlist embedded in the binary, after showing the plan and receiving explicit `--yes` consent. It uses structured package-manager arguments (no shell), no silent elevation, no repository additions and no auto-accepted agreements. Installs are user scope unless an allowlist entry declares machine scope (Windows only), which runs only from an attended terminal so Windows shows its own administrator prompt; community-built packages are flagged before consent ([ADR-0006](docs/architecture/adr/0006-install-scope-and-provenance.md)). It does not authenticate to remote services or send telemetry.

Every new installable application requires a verified origin and publisher identity, an allowlist entry and real-device acceptance ([ADR-0004](docs/architecture/adr/0004-embedded-catalog-and-allowlist.md)). There are no signed releases yet; signed artifacts, provenance and SBOM arrive with the first release (H4). SHA-256 from an untrusted source alone does not establish authenticity.

See [initial threat model](docs/security/threat-model.md).
