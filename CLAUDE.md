@AGENTS.md

## Claude Code

`AGENTS.md` above is the shared instruction file for every agent working here. What
follows applies only to Claude Code.

- `docs/architecture.md` is deliberately not imported. Read it with the Read tool
  when the work is structural, so it does not sit in context for small changes.
- The reference checkouts in `../gpui` and `../wails` are outside the working
  directory. Start the session with `--add-dir ../gpui --add-dir ../wails`, or ask
  for access, before trying to read them.
- Use plan mode for any change that crosses a layer boundary or alters an interface
  named in `AGENTS.md`.
- Per-package instructions live in `.claude/rules/`, scoped with `paths:`
  frontmatter so they load only when the matching files are touched. Add one there
  rather than growing this file.
- Do not spawn subagents unless asked to.
