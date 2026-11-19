# Threat model

Revised 2026-11-11 for H2 (first real installer). Review again at H4 (release), H6 (Android) and H8 (Multisuite).

Assets: personal documents, credentials, OS package state, installer provenance and optional Multisuite accounts.

| Threat | Control |
|---|---|
| Malicious/typosquatted package | Verify source, publisher, ID and evidence before enabling; install only allowlisted targets matching the verified catalog field by field |
| Tampered catalog on disk (`--root`) | Allowlist is embedded at build time and never read from disk; disk catalogs can change plans, not install targets |
| Unexpected elevation | Machine scope only when the allowlist declares it; announced in the plan; attended terminal required; Windows shows its own UAC prompt |
| Hidden extra packages | WinGet-declared dependencies listed in `provider_dependencies`, matched in CI and shown before consent |
| Community-built package | `provenance: community` recorded from the channel manifest, shown before consent and accepted per app by the maintainer |
| Personal data in local state | Journal stores error classes, never raw provider output or paths; per-user directory, 0600 |
| Concurrent or interrupted runs | Exclusive apply lock with dead-process detection; atomic journal writes; corrupt journal moved aside |
| Tampered release download | Reproducible builds; SHA256SUMS plus Sigstore-signed SLSA provenance (`gh attestation verify`); SBOM per archive |
| Compromised CI action or dependency | Actions pinned by commit SHA; least-privilege jobs; vendored dependency checked against `go.sum` |
| Invisible prompt or hung installer | Interactive only on a real terminal; otherwise prompts disabled; timeouts on every provider call |
| Command injection | Pass structured arguments to trusted package manager executables |
| Unintended elevation | Show plan and explicitly request per-operation consent |
| Compromised release | Future signed artifact, authenticated provenance and SBOM |
| Credential leaks | No secrets in manifests, logs or exported profiles |
| NAS/network outage | No dependency for local diagnostics or offline documents |
| Data deletion through sync | Never equate synchronization with backup |

Residual risks: package managers remain responsible for artifact authenticity; there is no signed Sovereign release yet (H4); Android and Multisuite surfaces do not exist yet.
