@AGENTS.md

# Claude Code — Sovereign

- `AGENTS.md` is the shared source of instructions. Do not duplicate it here or add contradicting rules. If imports are not interpreted, read `AGENTS.md` explicitly before editing.
- At session start and after compacting context, read `docs/STATUS.md` first; it states the active milestone and the next action.
- Use the session task list for short steps; persist decisions, evidence and the next action in `docs/STATUS.md`, not only in memory.
- Never run real installations, `remote-add`, elevation or package-manager mutations from tests or exploratory commands. `apply` against a real provider happens only on the user's test device.
- Do not delegate to parallel agents unless the user asks; if authorized, give each one disjoint files.
- Reply to the user in their language (usually Spanish) with changes, tests actually run, limitations and the next step.
