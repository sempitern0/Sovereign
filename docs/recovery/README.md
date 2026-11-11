# Offline and recovery

Status: design notes; implementation in **H5 — Data and portability** ([ADR-0005](../architecture/adr/0005-roadmap-release-first.md)). Today Sovereign does not create, synchronize or back up personal data.

Principles for H5 (detail in [PLAN §5](../../PLAN_TECNICO_SUITE_LIBRE.md#5-datos-locales-sincronización-y-recuperación)):

- `BibliotecaPersonal` is opt-in, configurable and never overwrites existing files.
- A synchronized folder is not a backup. Backups go to a different destination and count only after a tested restore.
- Three separate scenarios: offline use of installed apps and downloaded data; LAN without Internet; fully offline reinstall (kits in H7).
- The local `suitectl` journal (H3) is never the only copy of user data and must be rebuildable with `doctor`.
