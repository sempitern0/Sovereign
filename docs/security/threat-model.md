# Threat model — initial

Assets: personal documents, credentials, OS package state, installer provenance and optional Multisuite accounts.

| Threat | Control |
|---|---|
| Malicious/typosquatted package | Verify source, publisher, ID and evidence before enabling |
| Command injection | Pass structured arguments to trusted package manager executables |
| Unintended elevation | Show plan and explicitly request per-operation consent |
| Compromised release | Future signed artifact, authenticated provenance and SBOM |
| Credential leaks | No secrets in manifests, logs or exported profiles |
| NAS/network outage | No dependency for local diagnostics or offline documents |
| Data deletion through sync | Never equate synchronization with backup |

H0 uses schema contracts to block unverified packages; it does not establish operational installer security.
