# Android — guided setup (H6)

Status: not started; planned for H6 ([ADR-0005](../docs/architecture/adr/0005-roadmap-release-first.md)).

- Guided, not automatic: Sovereign proposes apps and verified sources (F-Droid, Obtainium) and the user installs them with the OS's own prompts.
- No ADB elevation, silent APK installs, signature bypass or sandbox exceptions.
- Each Android source and package ID needs the same verification and acceptance as desktop apps, on an emulator and at least one real device.
- `suitectl` is not assumed to run on Android; the first deliverable is an exportable profile plus a guide.
