# 10. Wails-facing shared state must be explicitly synchronised

Date: 2026-09-06

## Status

Accepted

## Context

Wails dispatches each frontend call on its own goroutine
(`Frontend.processMessage` → `go func()`). Every exported `WApp` method is
therefore a concurrent entry point into shared state.

This surfaced as a hard crash:

```
fatal error: concurrent map writes
main.(*WApp).clearLspStartError(...)  frontend.go:138
main.(*WApp).notesLspServerFor(...)   frontend.go:1290
main.(*WApp).NotesLspChangeDocument(...)
```

`lspStartErrs` was read and written by LSP start/error handling while being
deleted from by document-change calls. Rapid typing produced overlapping
`NotesLspChangeDocument` calls and the runtime aborted the process — an
unrecoverable failure, not a recoverable error.

## Decision

Every mutable map or slice reachable from an exported `WApp` method is guarded by
a dedicated mutex, scoped as narrowly as possible.

- `lspStartErrMu` guards `lspStartErrs`.
- Distinct state gets a distinct mutex (`notesMu`, `notesListMu`, `typosMu`) so
  unrelated operations do not serialise against each other.
- Locks are **not** held across renderer notifications, IPC emits, or other slow
  calls. In `notifyLspStartError` the map is updated under the lock, the lock is
  released, and only then is the notification displayed.

The same principle applies to the agent (`toolMu`, `toolPermissionMu`) and to
`sessiondb` (`aiSessionLogStore`, `panelView`).

## Consequences

- Frontend-driven concurrency can no longer abort the process.
- Fine-grained locks avoid a single global bottleneck on a UI-latency path.
- **Deliberately not** guarded by one coarse `WApp` mutex: that would serialise
  unrelated frontend calls and hurt responsiveness.
- New shared state on `WApp` needs a matching mutex — the compiler will not catch
  its absence, and `go test -race` only catches it if a test exercises the
  concurrency.

## Reference

- `frontend.go` — `lspStartErrMu`, `notifyLspStartError`, `clearLspStartError`
- `ai/agent/agent.go` — `toolMu`, `toolPermissionMu`
- `ai/agent/sessiondb/session_log.go` — `aiSessionLogStore`, `panelView`
