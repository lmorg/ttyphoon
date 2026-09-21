# 19. LSP references use a dedicated Find result mode

Date: 2026-09-07

## Status

Accepted

## Context

Users need to click a symbol in a Go or other source file, open the editor
context menu, and find references across the project. LSP is the right default
because it understands the symbol; text search is still useful when no server
is configured or the server does not support `textDocument/references`.

The existing Notes Find panel already has a project-wide result list with file,
line, context and navigation behaviour. Creating a separate references panel
would duplicate that interaction and make result navigation inconsistent.

There are two scope questions:

1. What happens when the origin file has no LSP?
2. What happens when a user opens a result file that has no LSP?

## Decision

Use the existing project Find results list with an explicit mode/source:

- `findFilesMode: 'grep' | 'references'`
- `findFilesSource: '' | 'LSP' | 'Text search fallback'`
- `findFilesReferenceSymbol` stores the invocation symbol

The editor context menu always offers **Find references** when an editor is
available, as a direct item alongside ordinary Find text. It is also exposed as
a **References** button beside the project Find input. The existing LSP options
menu retains its references entry for users who open that menu first. Other LSP
actions remain gated on the current file's LSP eligibility.

Invocation flow:

1. Extract the identifier under the cursor.
2. If the current file has an attached/configured LSP, call
   `textDocument/references` with `includeDeclaration: true`.
3. Normalize returned locations into the existing project-result shape and label
   them `LSP`.
4. If LSP is unavailable or returns no locations, run the existing project grep
   with whole-word matching and label results `Text search fallback`.
5. Keep the result list and its source when a result file is opened.

The current file's LSP capability controls only how a **new** references search is
started. It does not mutate an existing references result set when the user
opens a file without LSP. Typing into the project Find input explicitly exits
references mode and starts an ordinary grep search.

The backend request lives in `utils/lsp/references.go` and follows the existing
definition normalization and position-encoding conversion patterns. Unsupported
LSP methods are treated as unavailable and naturally reach the fallback path.

## Consequences

- One result UI serves grep and references, reducing duplicated navigation code.
- Users can use references on files without LSP, with honest labelling of weaker
  textual results.
- Text fallback may include comments, strings, shadowed identifiers and other
  unrelated matches; it is not semantic and must not be presented as equivalent
  to LSP references.
- Reference results are currently rendered without source snippets because LSP
  locations do not carry context. Navigation remains file/line based.
- LSP references require a full vertical slice: backend request, Wails bridge,
  generated bindings, frontend import, menu entry, and Find-state handling.

## Reference

- `utils/lsp/references.go` — request and normalization
- `frontend.go` — `NotesLspReferences`
- `frontend/src/notes.js` — `findReferencesFromEditor`, references Find mode,
  source labels, stable result navigation, direct context-menu action and Find
  panel button
- `frontend/wailsjs/go/main/WApp.js`, `WApp.d.ts`, `models.ts`
