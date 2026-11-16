# 0101 - Notes documents in separate windows

Date: 2026-11-16

## Status

Proposed for future implementation. The user wants Notes documents to be
openable in a separate window without closing or reopening them when the main
workspace changes. The design below is a recommendation, not an accepted
implementation plan; no work is implied now.

## Context

Notes supports multiple documents, tabs and recursive split views in one
window ([ADR 0100](0100-notes-multiple-open-documents.md)). A separate OS
window would allow a document to remain visible while the main window switches
workspaces. Its project and file must continue to refer to the original
workspace rather than the newly selected one.

The previous `experiment/notes-new-window` attempt is described in ADR 0100:
each Wails window had a separate JavaScript realm and a separate `WApp`. This
avoided frontend singleton state but duplicated backend services, including
language servers, and required UI emissions to be scoped across many files.
The attempt predated ADR 0100's document/view separation and reference-counted
LSP and typos documents; those are useful foundations, not a complete
multi-window solution.

Today the main renderer derives Notes context from the active terminal or tmux
pane. A context change causes `refreshFiles` to close background tabs and
restore the selected workspace's layout. `WApp` also holds mutable current
project and UI state. For example, `GetFile` and LSP bindings resolve paths
against the current context, while `SaveFile` can take an explicit project.
Simply ignoring workspace-change events in the new webview would therefore
still leave backend operations at risk of targeting the wrong project.

## Proposed direction

### Window and context

- Add a Notes-only Wails window which reuses the existing Notes document and
  view pipeline rather than a reduced second editor. It must not start another
  terminal or inherit main-window startup and rendering lifecycle actions.
- Capture an immutable originating file context when opening or moving a
  document: project root, workspace identity and Notes roots. Preserve it
  across main-workspace changes. Context-sensitive operations must receive or
  resolve the correct context explicitly, including file reads and writes,
  binary files, asset and Markdown resolution, file watching, LSP and typos,
  execution, Git and any file navigation the window exposes.
- Use a stable document identity that distinguishes identical `$PROJECT`
  paths in different projects. Preserve the actual file and project
  association across workspace switches, renames and deletes; do not silently
  fall back to the currently active project.
- Isolate window-local state and requests from global backend state. In
  particular, audit shared Markdown base directories, file-index requests and
  cancellation, Git watches, dialogs and history. The main workspace must
  continue to behave as before.

### Document ownership

For an initial version, prefer **Move to new window** with one editable
document owner across windows rather than independent editable copies. A
Monaco model cannot be shared directly across webview JavaScript realms, and
LSP reference counts alone do not synchronize their contents or undo histories.
Reopening the same file elsewhere should focus the owning window or otherwise
prevent a second editable owner; the exact UX remains open.

The move must be transactional: retain the source until the destination is
ready and ownership is transferred, and surface errors without losing unsaved
text. Define what happens when the same document is shown in several split
views, and how dirty, conflicted or deleted documents, editor state, undo
history, active Jupyter runs and pending saves transfer. A save-and-reopen
approach is simpler but cannot by itself preserve all of that state; it must
not silently discard it. If full transfer is not feasible, make the limitation
explicit and require an appropriate user action rather than pretending the
move succeeded.

Later, simultaneous editable copies could use a shared document authority or
an explicit synchronization and conflict protocol. Disk autosave or
reference-counted LSP registrations alone are not sufficient.

### Shared services and lifecycle

- Share backend services such as the LSP manager and document stores across
  windows, with ownership and cleanup tracked per window. UI operations and
  window-specific events must be routed to their intended window; broadcasts
  such as diagnostics and file changes must be filtered by document and
  context. Audit application-wide events before adding another frontend.
- A detached window closing must not stop servers, kernels, watchers or other
  resources still used by another window. In particular, the existing
  frontend unload handler calls `NotesLspStopAll`; replace or scope that
  behaviour as part of multi-window support. Clean up registrations on normal
  close and unexpected window loss.
- Closing, hiding, application shutdown and unsaved or conflicting changes need
  explicit handling. Do not lose data if a window fails to open, a save fails,
  or a close is cancelled. Decide separately whether windows and their
  positions should be restored after app restart; persistence is not required
  merely to survive a workspace switch.

## Suggested implementation stages

1. Define window identity, context and document-ownership APIs; inventory
   context-sensitive bindings and events. Test two distinct project contexts
   without changing the existing main-window behaviour.
2. Introduce a Notes-only window and window-targeted UI routing with shared
   backend services. Verify its startup and close do not affect the terminal
   or other windows.
3. Implement safe document transfer and prevent duplicate editable owners,
   including explicit failure and close handling. Ensure workspace changes do
   not close, reload or redirect detached documents.
4. Bring all exposed Notes features to parity using the existing pipeline,
   including file changes, LSP, spelling, find and Jupyter. Only enable the
   feature when its supported behaviour and limitations are clear.
5. Consider window restoration and synchronized simultaneous editing
   separately, if wanted.

## Verification

- Open or move a document from project A, switch the main window to project B,
  and continue editing, saving, finding and using LSP in A. Test identical
  relative paths in both projects and verify neither reads nor writes cross
  projects.
- Exercise repeated workspace switches and multiple detached windows, including
  `$NOTES`, `$GLOBAL` and `$PROJECT` files; check that each retains its
  original context and document.
- Test transfer and closure with dirty files, save failures, conflicts, deleted
  and renamed files, multiple split views, active runs, failed window creation
  and cancelled closes. Verify no edits disappear or are overwritten.
- Verify duplicate-open behaviour, window-scoped menus and notifications, and
  correct LSP/typos registrations and cleanup when one window closes while
  another remains open. Test external changes and relevant Notes features in
  both windows.
- Test the supported desktop platforms and main-window close/hide and
  application shutdown behaviour. Confirm existing single-window tests and
  behaviour remain intact.

## Open questions

1. Should the initial action move a tab, open an independent window, or offer
   both? What happens to other split views showing the same document?
2. When another window opens an already-owned file, should it focus the owner,
   show a read-only view, or offer a transfer?
3. Which Notes chrome and features belong in the detached window? Can it open
   additional documents or split views, and must all of them share its
   originating context?
4. What editor state and running work must survive a move, and what should
   happen if it cannot be transferred safely?
5. Should detached windows, their geometry and documents be restored on app
   restart, and what should happen when their original workspace no longer
   exists?

## Consequences

- Pinned context makes detached windows independent of workspace changes, but
  requires auditing APIs that currently assume one active project.
- Shared services avoid duplicate language servers, but require careful event
  routing and per-window resource ownership.
- Single editable ownership avoids cross-window buffer conflicts initially,
  at the cost of limiting simultaneous views and requiring a safe transfer.

## References

- [ADR 0100: Notes multiple open documents](0100-notes-multiple-open-documents.md)
- [ADR 0091: Notes web link windows](0091-notes-markdown-middle-click-opens-wails-window.md)
- [ADR 0092: Wails window packages and feature bindings](0092-wails-window-packages-and-feature-bindings.md)
- `frontend/src/notes.js`, `frontend/src/notes_document.js`
- `window/frontend/main/window.go`, `window/frontend/main/service.go`,
  `window/frontend/main/files.go`, `window/frontend/main/lsp.go`
- `window/backend/renderer_webkit/notes.go`
