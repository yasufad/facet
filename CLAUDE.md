@AGENTS.md

## Claude Code

- `docs/architecture.md` is deliberately not imported. Read it when the work is
  structural, so it does not sit in context for small changes.
- The reference checkouts in `../gpui` and `../wails` are outside the working
  directory. Start with `--add-dir ../gpui --add-dir ../wails`, or ask for access,
  before trying to read them.
- Use plan mode for any change that crosses a layer boundary or alters an interface
  named in `AGENTS.md`.
- Per-package instructions go in `.claude/rules/`, scoped with `paths:` frontmatter
  so they load only when the matching files are touched. Add one there rather than
  growing this file.
