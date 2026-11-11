# Multisuite — optional connector (H8)

Status: not started. Planned last among functional milestones ([ADR-0005](../../docs/architecture/adr/0005-roadmap-release-first.md)) so that Sovereign is complete and useful on its own first.

## What Multisuite is today (checked 2026-11-11)

- A private Laravel/Livewire portal that links existing self-hosted services; each keeps its own UI and accounts. It currently integrates **Navidrome** (music) and provisions Navidrome accounts through invitations. No SSO.
- It exposes **no API for Sovereign**. Remote private access (VPN/TLS) is pending in its milestone P03. Planned additions (P04) are being evaluated: Komga, Jellyfin and a notes service.

## Contract-first rule

No adapter is written against an assumed API. H8 starts with a short interface document agreed in **both** repositories, covering at least:

| Capability | Purpose |
| --- | --- |
| `discover` | List the services the authenticated user may use (id, kind, base URL, protocol such as Subsonic) |
| `health` | available / degraded / offline, so Sovereign can fall back to local use |
| `version` | Contract and service versions for compatibility checks |

## First use case: music

With Multisuite reachable: Sovereign proposes a client compatible with Navidrome's Subsonic API on each device (verified like any catalog app) and shows the service URL. Credentials are entered by the user in that client or kept in the OS secure store; never in YAML, logs, Git or exported profiles.

With Multisuite off or without VPN: `doctor`, `plan`, `apply` and local files work unchanged.

## Prerequisites

- Multisuite: private access (P03) and a published discovery interface.
- Sovereign: H3–H5 done, so the connector only adds clients and links on top of a working local suite.
