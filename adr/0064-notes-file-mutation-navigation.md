# 64. Notes file mutations preserve document navigation

Date: 2026-10-14

## Status

Accepted.

Updated 2026-10-21: rename/delete update history for the active tmux window name,
not the project root. Recent files are a distinct projection of that history;
there is no separate recent-file list to mutate (ADR 0074).

## Context

The Go rename/delete bindings already update recent files, navigation history,
the file index and LSP workspace notifications. Frontend navigation was incomplete:
deleting the open note cleared the editor without selecting another document,
and rename left the old document active until after the list refresh. The open
document watcher can emit `notesFileChanged` before the rename/delete binding
returns, causing the frontend to reload a path that no longer exists.

## Decision

- After renaming the open note, load the new path before refreshing the file list.
  Refresh without workspace history restoration. Renaming another note does not
  switch the open document.
- Folder rows offer rename through the same dialog and backend operation. Rename
  remaps descendant history paths and expansion state; an open descendant follows
  its new path with watcher reloads suppressed during the operation. Matching uses
  path-separator boundaries so similarly named sibling folders are unaffected.
- Track the path being renamed/deleted and suppress its watcher-driven reloads
  during the operation. Clear the gate in `finally`, including on failure.
  Subsequent notifications reload the new current document normally.
- History deletion removes every position for the deleted path. When the current
  position is removed, prefer the preceding surviving entry; if there is none,
  use the next surviving entry. Keep the selected entry when another file is
  deleted. Empty history remains empty.
- After deleting the open note and refreshing the list, ask Go for the current
  surviving history position and load it. Do not step Previous again. Select its
  Edit mode and restore its saved line through the live-surface navigation helpers
  from ADR 0021. If no entry remains, keep the editor empty. Deleting another note
  leaves the open document unchanged.
- Reuse asynchronous `loadFile` and indexed list refreshes rather than adding
  synchronous filesystem reads or full rescans (ADR 0058).

## Consequences

Go owns history selection; the frontend owns opening the surviving document and
restoring its cursor. Regression tests cover renamed-path reads, open/non-open
deletion, empty history, repeated same-document positions, previous/next fallback
and saved-line restoration. Frontend tests inject watcher notifications before
the mutation binding resolves and verify normal reloads resume afterwards.

The separate grep-option improvement was already implemented:
case-sensitive, regex and whole-word toggles call
`triggerImmediateProjectFindSearch`, cancel pending debounce and rerun using an
option-sensitive search signature. Tests now verify all three toggles; no search
implementation change was needed.

## Reference

- `frontend/src/notes.js` and `frontend/src/notes.test.js`
- `utils/notes/history.go` and `utils/notes/history_test.go`
- ADR 0021: live-surface editor navigation
- ADR 0058: indexed file lists and write-through mutation hints