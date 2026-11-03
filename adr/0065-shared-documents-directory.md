# 65. Shared application documents directory

Date: 2026-10-14

## Status

Accepted.

## Context

Notes, asset serving, image uploads, generated images/graphs, Git worktrees,
session databases/logs and command history independently constructed
`~/Documents/ttyphoon`. Some used `app.DirName`; worktrees hard-coded `ttyphoon`.
Duplicated root policy could drift across storage producers and consumers.

## Decision

Use `file.DocumentsDir(home)` in `utils/file` as the single constructor of the
application documents root. It joins the supplied home directory, `Documents`
and `app.DirName` using `filepath.Join`.

The helper is pure: it does not discover the home directory, create directories,
change permissions, cache paths or apply sandbox policy. Callers retain their
existing home source (`os.UserHomeDir`, `xdg.Home` or the Wails app's home field)
and error handling, then append their own storage suffixes.

Global notes retain their existing trailing separator. The project-root fallback
to `~/Documents` is not application storage and remains separate. Documentation
examples and test expectations may still contain literal paths.

## Consequences

All production constructors of the application documents root share one policy,
without a migration of existing files or changes to asset allowlists, storage
permissions or directory creation timing. Path tests cover supplied home paths,
including spaces and trailing separators; existing caller tests verify storage
paths remain unchanged.

## Reference

- `utils/file/documents.go` and `utils/file/documents_test.go`
- `frontend.go`
- `utils/notes/pwd.go`, `utils/history_md/history_md.go`, `utils/git/git.go`
- `ai/agent/sessiondb/session_db.go` and `session_log.go`
- `ai/tools/image/generate.go` and `ai/tools/graphs/graph.go`