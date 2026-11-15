# 0100 - Notes: multiple open documents

Date: 2026-11-14

## Status

Proposed. Scope, same-file and fallback-editor decisions were made on
2026-11-14. The remaining open questions must be answered before the UI
stages (stage 7 onwards) start.

## Context

Notes can only have one document open at a time. Supporting several documents
is an open item in `improvements.md` and has been attempted twice without
success. This record explains why those attempts failed, what is already
multi-document capable, and what has to change.

### Previous attempts

**1. In-window MDI (`experiment/notes-multi-doc`, ADR 0036, Sep 2026).**
ADR 0036 only exists on that branch. It aimed for a recursive split layout
where Notes documents, the terminal, AI and logs were all movable surfaces, and
the same file could be open in several surfaces at once. It was built
bottom-up over 33 slices: layout model, surface controller, surface-scoped
DOM, runtime and service bundles. It failed because:

- **The scope was far too large.** It covered layout, tabs, splits, the same
  file in several views, and moving terminal, AI and logs, all at once.
- **There were two document pipelines.** The first surface kept the full
  `notes.js` pipeline. Secondary surfaces got a cloned DOM and a thinner
  loader and binding path that never reached parity with Monaco, LSP, Jupyter,
  tables, Swagger, hex or autosave. Every feature would have had to be built
  twice.
- **Legacy names aliased the active surface's services.** This moved storage
  but not ownership: handlers still read "the current one", so a second
  surface could not use them.

**2. A Wails window per document (`experiment/notes-new-window`, Nov 2026).**
Each document got its own webview, and therefore its own JavaScript realm,
which avoided the frontend singletons. On the backend, each window got its own
scoped `WApp` from `NewWailsApp()`. It failed because:

- **The backend was duplicated.** Each `WApp` builds its own `lsp.Manager`
  and `DocumentStore`s, so every window started its own gopls (and other
  servers). The same file open in two windows was tracked by two servers.
- **Every UI emission had to be scoped.** Notifications, menus, input boxes,
  AI panel output and hyperlinks all needed a target window (`scoped_ui.go`),
  which spread the change across 44 files.

Taken together: isolating the frontend per document pushes the problem into
shared backend services, and sharing one frontend with several half-supported
surfaces creates two pipelines.

### What already supports multiple documents

- **Backend LSP.** `utils/lsp.DocumentStore` keys documents by URI, each with
  its own version, and `lsp.Manager` runs one server per
  `{workspace, language}` (ADR 0016). Jupyter blocks already open virtual LSP
  documents alongside the main file.
- **Diagnostics.** `notesLspDiagnostics` events carry a URI, and the frontend
  stores them per URI in `lspDiagnosticsStore`. Only the active URI is
  rendered.
- **Monaco.** One editor instance can switch between many `ITextModel`s with
  `editor.setModel()`. Each model owns its own content, undo stack and
  decorations, and `saveViewState()`/`restoreViewState()` keep the cursor and
  scroll position.

### What assumes a single document

At 2026-11-14, `frontend/src/notes.js` is about 18,000 lines with:

- One `state` object with about 100 fields, mixing document state, view state
  and global state. `state.currentFile` is referenced 188 times.
- 123 `elements.*` handles. The hidden-textarea source of truth,
  `elements.editor`, was referenced 181 times; ADR 0021 describes its live
  surface split. (Removed in Stage 1 task 3, see decision 7.)
- `monacoMainEditor` (81 references) and `getMainEditorValue()` (51) assume
  one model.
- `state.lspOpenFile` (35 references) holds one path:
  - `closeOpenLspDocument` closes it before `openCurrentLspDocument` opens the
    next file, so switching files always sends `didClose`.
  - Every Monaco LSP provider checks
    `state.lspOpenFile === state.currentFile` and reads
    `getMainEditorValue()`, rather than resolving the document from the
    requesting model.
- Single global LSP UI state: hover, completion, signature-help timers and
  the tooltip elements.

Backend gaps:

- `DocumentStore` has no reference count. Two frontend owners of one URI
  (for example two tabs, or a tab and a block) would get `didClose` when the
  first one closes.
- `NotesTyposOpenDocument` closes and reopens a URI that is already open. That
  is fine for one owner and wrong for several.

## Proposed decisions

Decisions marked *Decided* came from the user on 2026-11-14. The rest are
recommendations to accept or amend while answering the open questions.

### 1. Scope: more than tabs, less than full MDI

*Decided:* version 1 must be more than tabs only, because one visible
document with a tab strip is barely different from today's file pickers. It
must also not attempt full MDI in one change. The exact version 1 UI is still
open (question 1), but the code must not prevent a later move to full MDI with
recursive splits and movable terminal, AI and logs.

### 2. Three layers: document, view, layout

Separate state into three layers from the start, even while the UI only uses
one view:

- **`NotesDocument`**, one per open file, keyed by absolute path. All views of
  that file share it: the Monaco model, dirty state, autosave, LSP
  registration and diagnostics.
- **`NotesView`**, one per visible editor pane. It owns the editor instance,
  its preview and structured-view host, its tab list, the active document, and
  per-document view state (cursor, scroll, view mode).
- **Layout**, a serialisable tree of views. Version 1 uses a fixed, small
  shape; full MDI later grows the same tree and adds terminal, AI and logs as
  leaf kinds.

*Decided:* the same file can be open in two views. Both share one
`NotesDocument` and one Monaco model, which Monaco supports natively, so edits
appear in both with no reconciliation.

### 3. One pipeline: every view comes from the same factory

ADR 0036 failed because the first surface kept the full legacy pipeline and
secondary surfaces got a thinner one. To avoid that:

- the existing Notes editor area is refactored into a `NotesView` factory, and
  the first view is created through that factory, not kept as a legacy view;
- a view's DOM is created from one template and looked up within that view's
  root, never with global IDs;
- any feature that works in one view works in every view, because there is
  only one implementation.

To keep version 1 small, chrome that only one view needs at a time stays
global and follows the **focused view**: the tools panel (Find, ToC,
Frontmatter), LSP hover and completion overlays, and the status bar. Only the
tab strip, editor host and preview/structured-view host are per view.

### 4. State classification

| Per document (shared by all its views) | Per view | Global (follows focused view) |
|---|---|---|
| path, URI, file type, project | Monaco editor instance | file list and index |
| Monaco model (content, undo, decorations) | tab list and active document | tools panel: Find, ToC, Frontmatter |
| dirty flag, saved version, autosave timer | per-document cursor, scroll, view mode | LSP hover, completion, signature UI |
| LSP registration, diagnostics, semantic tokens | preview and structured-view DOM | status bar, Git panel, AI, logs |
| Jupyter block runtimes and kernels | | LSP servers (backend), hotkeys |

`state.currentFile` and similar fields become read-only accessors for
`focusedView().activeDocument`. UI handlers may read the focused view.
Asynchronous work must not.

### 5. Asynchronous work captures its document and view

Anything that awaits (load, save, LSP requests, rendering, table queries,
Jupyter output) captures the `NotesDocument`, and the `NotesView` if it paints,
when it starts. When it finishes, it updates that document if it is still open
and paints only views still showing it. This replaces ad-hoc comparisons such
as `lspOpenFile === currentFile`.

### 6. LSP follows open documents, not the focused one

- `didOpen` when a document is first opened in any view; `didClose` when its
  last view closes it. Switching tabs or focus sends nothing.
- Monaco providers resolve their document from `model.uri`, never from
  `state.currentFile`. A request from either view of a shared model resolves
  to the same document.
- The backend `DocumentStore` (LSP and typos) gets a per-URI reference count:
  the first open sends `didOpen` and the last close sends `didClose`.
- Move LSP code from `notes.js` into its own module first (already on the
  refactor list), keyed by document from the start.

*Done (Stage 2, task 1):*

- `frontend/src/notes_lsp.js` holds the main document's LSP and typos code:
  open, change and close, diagnostics, hover, completion, signature help,
  code actions, rename, symbols, references, and the Monaco provider
  callbacks. Nothing changed behaviour; the code moved as is.
- It does not import `notes.js`, which is the side-effecting entry point.
  `notes.js` calls `initNotesLsp(deps)` once, passing the shared state and
  helpers it needs. `monacoMainEditor` and `notesSpellCheckHandle` are
  reassigned in `notes.js`, so they are passed as getters. `initNotesLsp` also
  subscribes to the `notesLspDiagnostics` and `notesTyposDiagnostics` events.
- The semantic-tokens delta callback called `NotesLspSemanticTokensDelta`
  without importing it. The new module imports it.

*Done (Stage 2, task 2):*

- Monaco providers are registered per language, so they also ran for other
  models of that language, such as the Git diff editor's, and answered with
  the main file's results. Each provider now passes the model it was asked
  about to its callback (`{ model, line, character }` and so on), and results
  use that model's URI and range.
- `notes_lsp.js` keeps `lspModelDocuments`, which maps a model URI to
  `{ filePath, fileUri }`. `openCurrentLspDocument` registers the main
  editor's model (`getModelUri()` on the adapter) and `closeOpenLspDocument`
  removes it. Callbacks resolve their document with `lspDocumentForModel` and
  read text from the model, replacing the
  `state.lspOpenFile === state.currentFile` guards and `getMainEditorValue()`.
  A model that is not registered gets an empty result.
- Code actions carry their model and position in the command arguments, so
  applying one uses the position it was computed for, not wherever the cursor
  is by then.
- The semantic-tokens fallback cache is per model, and the edits provider now
  takes Monaco's `(model, previousResultId)` arguments; it was treating the
  model as the result ID.
- Not yet keyed by document: hover, completion, signature help from the cursor,
  and `didChange` scheduling still follow the focused main editor and
  `state.lspOpenFile`. They move with `NotesDocument` (Stage 4).

*Done (Stage 3):*

- `utils/lsp.DocumentStore` counts references per URI. The first `DidOpen`
  sends `didOpen`; later opens add a reference and send `didChange` only if
  the text differs (their language ID is ignored). Only the last `DidClose`
  sends `didClose`. Closing an unknown document does nothing. The counts do not
  depend on order, so concurrent Wails calls are safe.
- `DidClose` with a nil transport only updates the store.
  `NotesLspCloseDocument` calls it that way, and before this it could panic.
- `NotesTyposOpenDocument` no longer closes and reopens a document that is
  already open. Reopening it adds a reference and sends `didChange`.
- Each frontend holder now pairs every open with exactly one close. The
  holders are the main document (`state.lspOpenFile`, `state.typosOpenFile`)
  and each Jupyter block (`lspFilePath`, `typosFilePath`).
  - `openCurrentLspDocument` closes the previous document first.
    `lspOpenFile` is set even when no server started, so closing first means
    a failed start is retried.
  - `openCurrentTyposDocument` opens the new document and then releases the
    old reference, so reloading the same file becomes a `didChange`.
  - `renderJupyterView` releases the previous blocks' documents
    (`releaseJupyterBlockLspDocuments`).
- Remaining gap: after switching to another file, the previous file's Jupyter
  block documents stay open until the next Jupyter render. This predates the
  change. They are released once block runtimes move onto `NotesDocument`
  (Stage 4).

*Done (Stage 4):*

- `frontend/src/notes_document.js` defines `NotesDocument` and a registry
  (`open`, `get`, `close`). Opening a file that is already open returns its
  existing document.
- The registry key is the Notes path used by every backend call (for example
  `$NOTES/a.md`). For `$PROJECT/...` paths, the prefix is replaced with the
  project root, because the same path names a different file in each project.
  The absolute path is not used because it is only known after an
  asynchronous lookup.
- A document holds its path, project, URI, file type, text buffer, dirty
  flag, autosave timer, LSP and typos registration and change timers, Jupyter
  blocks, and the blocks that are running.
- Closing a document:
  - clears its timers;
  - stops its block runs (`StopNote`);
  - releases its blocks' LSP and typos documents. This closes the Stage 3 gap.
- `state.currentFile` and `state.currentFileProject` are read-only accessors
  for the active document. The other fields read and write the active
  document.
- `loadFile` and the Git diff view switch document with
  `activateNotesDocument`. Deleting the current file, and history leaving no
  file, call `closeActiveNotesDocument`. Notes still shows one document at a
  time, so switching closes the previous document.
- A document's LSP and typos registrations are released when it closes
  (`releaseLspDocument`), not by `loadFile` before every switch. Switching
  only resets the UI the view showed for that document (`resetLspViewState`:
  hover, completion, spellcheck mode). If the new file fails to load, the
  previous document stays active with its LSP and typos still open; before
  this, they were closed while it was still shown. The delete and Git
  deleted-file paths still close explicitly, because the file itself is gone.
- Switching files within the one-second autosave delay used to lose those
  edits. The timer then fired on the new file after the editor text had been
  replaced. `loadFile` now saves pending edits to the previous file before
  switching. The project captured when the file was opened is used for that
  save, even if the project has changed since.
- Output events (`noteRun`, `noteComplete`) are applied only for runs started
  from the active document. Block IDs (`jupyter-block-N`) are only unique
  within a document, so this keeps a closed document's output, including the
  backend's `[process killed]` message, out of the next document's blocks.
- Not yet per document:
  - The Monaco model. The adapter still has one model, and the active
    document's buffer is attached to it. A model per document needs model
    swapping in the adapter, which is needed once two documents are open at
    the same time (Stage 7).
  - Backend run IDs and output events. These carry only the block ID, so
    running blocks from two open documents needs the document in the ID
    (Stage 7).

*Done (Stage 5):*

- `loadFile` takes a sequence token and checks it after every `await`. A load
  that finishes after a newer one started stops without touching the editor,
  views, status bar, LSP or typos. Previously a slow load could overwrite the
  file opened after it.
- `saveFile` and the autosave timer capture the document, its path, project
  and text when they start. After `SaveFile` returns:
  - if the document has closed, nothing else is updated;
  - if it is open but not shown, only its dirty flag is cleared;
  - if it is shown, it is marked saved only when its text has not changed
    since the save started. Edits made while a save was in flight used to be
    marked as saved.
- A document holds at most one LSP and one typos registration. An open that
  finishes after its document closed, or after another open already
  registered it, closes its own reference. This keeps the Stage 3 reference
  counts balanced when files are switched quickly.
- LSP requests (completion, hover, signature help, definition, symbols, code
  lens, code actions, format, rename) capture the shown document with
  `captureShownDocument()` and drop the response if another document is shown
  by then. Format, code action and rename used to apply their edits to
  whichever file was open when the response arrived. Requests also send the
  path captured at the start.
- `renderMarkdown` and `renderJupyterView` stop after an `await` if a newer
  render started or their document is no longer shown. Jupyter code blocks
  are only registered on the document that started the render.
- Table formulas send `RunFunction` the file captured when evaluation started.
  Image previews and Swagger responses are dropped if their document is no
  longer shown.
- Jupyter output was already scoped to the active document's runs in
  Stage 4.
- Still compared against the single active document, not a `NotesView`.
  Stage 6 replaces "shown" with "shown in this view".

*Done (Stage 6, task 1):*

- `frontend/src/notes_view.html` is the view template: the mode tab row
  (View, Edit, Run, …) and the panel holding the editor, Git diff, hex,
  preview, HTML, Run, CSV, image, Meta and Swagger hosts. Elements carry a
  `data-part` name and a class; none has an ID.
- `createNotesView({ host })` in `frontend/src/notes_view.js` creates a view
  from the template and returns its `root` and `parts`, looked up by
  `data-part` within that root. `notes.js` creates the one view (`mainView`)
  inside `#notes-views` at startup, and its `elements` map takes the view
  elements from `mainView.parts`.
- `notes.html` no longer contains the view markup. `#notes-panel` now holds
  `#notes-views` (the layout host) above the Tools panel, and each view holds
  its own tab row above its panes, which is the shape a second view needs.
- The toolbar (New, history, maximise) is global and keeps its IDs. It is
  moved into the view's tab row at startup, where it sat before. Where it
  goes with two views is part of question 1.
- The Git diff flag (`data-git-diff`) moved from `#notes-panel` to the view
  root, so it hides only that view's panes.
- CSS selectors for view elements changed from IDs to classes scoped by
  `.notes-view` (for example `#notes-preview` became
  `.notes-view .notes-preview`). The extra class keeps them above
  single-class rules such as `.markdown-body`, which the IDs used to beat.
  The runtime font-size styles and the hex viewer styles changed the same way.
- The tab row now sits inside `#notes-panel`, so it gets the
  `#notes-main` background overlay explicitly, and the Tools resizer
  subtracts the tab row's height so the Tools size is the same share of the
  same space as before.
- Checked by rendering the old and new markup with the real CSS in Chrome
  across 15 states (each view mode, Git diff, Tools expanded, AI and log
  maximised, terminal focused, full size), for screen and print. Screenshots
  are pixel-identical, and computed styles match except the tab row's
  background, which composites to the same colour.

*Done (Stage 6, task 2):*

- The view owns its Monaco editor (`view.editor`, created lazily into the
  view's editor host). The module-level `monacoMainEditor` is gone.
- Each view keeps, per document key, the view mode and Monaco's view state
  (cursor, selection and scroll) it last showed (question 15).
  `loadFile` records them for the outgoing document and restores them for
  the incoming one. The view's remembered mode overrides the persisted
  document cache, which still supplies the mode for a document the view has
  not shown, and is still written so it survives restarts.
- State is recorded only once a load has put the document's content in the
  editor (`view.shownDocumentKey`), so an interrupted load cannot store one
  file's cursor under another. Loads that ask to skip the cache (opening a
  project-search result) skip it too, and callers that jump to a line after
  loading still win.
- Preview scroll positions are not restored; the preview re-renders on
  activation.

### 7. Remove the textarea fallback editor

*Decided:* remove `notes-editor-monaco = 0` and the textarea fallback. Monaco
becomes the only editor and its models become the source of truth, which
retires the hidden textarea described in ADR 0021.

There is a test cost. `notes.js` disabled Monaco under jsdom, so the whole
`notes.test.js` suite ran against the textarea fallback. Removing it needs a
fake Monaco adapter for tests. It replaces the textarea as what tests drive.

*Done (Stage 1, tasks 1–2):*

- `frontend/src/test-support/fake_monaco_adapter.js` is a string-backed fake of
  the `monaco_adapter.js` API. `notes.test.js` uses it as the default
  `createMonacoAdapter` mock and reaches it through `mainEditorAdapter()`. It
  has helpers that simulate Monaco input: `typeText`, `moveCursor`,
  `pressEscape`, `requestCompletion` and `contextMenu`.
- The `notes-editor-monaco` setting, the textarea's input handlers (keyboard,
  Vim, mouse, paste, context menu, managed undo shortcuts) and the fallback CSS
  are gone. `isMonacoActive()` now only means "Monaco has been created yet".
- `setMainEditorValue` normalises CRLF to LF. The textarea did this implicitly,
  and Run-view block offsets (from `marked.lexer`) assume LF.

*Found while doing this — undo gap:* programmatic edits made through the
mutation adapter and `notesUndoManager` (for example context-menu JSON format
and CSV Run-mode row insertion) write the textarea and reach Monaco through a
whole-model `setValue`. The managed Cmd+Z shortcut that undid them only ever
worked in the fallback editor, so in Monaco they can't be undone. LSP format,
rename and quick-fix edits use `executeEdits` and are undoable. Task 3 should
route programmatic edits through Monaco edits (or a document buffer that does)
so that every edit is on Monaco's undo stack.

*Decided for task 3:* Monaco is created lazily, so the source of truth is a
per-document text buffer (text, selection, change event) that Monaco binds to
when it mounts, rather than eagerly creating a Monaco model for every open file.
This buffer becomes part of `NotesDocument` in Stage 4. Programmatic edits
(format actions, CSV Run-mode edits, preview checkbox toggles) go through the
buffer as range edits that Monaco applies with `executeEdits`, so they land on
Monaco's undo stack and Cmd+Z undoes them.

*Done (Stage 1, task 3):*

- `frontend/src/notes_document_buffer.js` exports `createDocumentBuffer`. It
  holds text and selection until a view attaches, then delegates reads and
  edits to the view. `setValue` is a reset (file load: no change event, not
  undoable). `replaceRange` and `replaceAll` are edits: with Monaco attached
  they go through the adapter's `replaceRange` (`executeEdits`), and
  `replaceAll` diffs the common prefix and suffix so the edit is minimal.
  Listeners registered with `onChange` replace the old textarea `input`
  handler.
- `notes.js` has one buffer, `mainDocument`. `replaceMainDocumentText` replaces
  every whole-document `notesMutationAdapter.replaceDocumentText(elements.editor, …)`
  call (format actions, CSV and Markdown table edits, checkbox toggles, ToC,
  LSP format, code actions and rename, structured edits, frontmatter, Find
  replace-all and git-diff edits). The textarea, `elements.editor` and its CSS
  are gone.
- `spellcheck.js` gained `createSpellCheckEngine`, which checks text from a
  `getText` callback rather than a DOM element. `attachSpellCheck` is built on
  it, and the main document uses the engine directly.
- The adapter's `replaceRange` pushes undo stops around each edit, so every
  programmatic edit is its own undo step. Edits made from another view (such
  as saving a Run-mode Markdown block) pass `preserveSelection` so the Edit
  caret does not jump.
- `notesUndoManager` and `notesMutationAdapter` remain for Jupyter cell
  textareas only.
- LSP completion trigger characters now come only from Monaco's `onDidType`
  (`handleMonacoCompletionRequest`). The change listener used to check the
  character before the textarea caret, which worked only because setting a
  textarea's value moves its caret to the end; at Monaco's content-change event
  the caret has not moved yet. The listener still re-filters an open menu.
- The Monaco container no longer has `aria-hidden="true"`, which made sense
  only while the textarea was the accessible surface.

### 8. Migrate in stages that each ship on `main`

Each stage leaves behaviour unchanged until the UI stages at the end, so the
full test suite stays green at every step:

1. **Remove the fallback editor.** Add the test fake, make Monaco the only
   editor, and stop treating the hidden textarea as the source of truth.
2. **Extract Notes LSP** into its own module, keyed by model URI.
3. **Reference-count** the backend `DocumentStore`.
4. **Introduce `NotesDocument`** and route `state.currentFile` and similar
   fields through it. There is still one document.
5. **Make asynchronous paths capture their document** (decision 5).
6. **Introduce `NotesView`.** Refactor the editor area into the view factory
   and create the one existing view through it. There is still one view.
7. **Tabs:** allow several documents in a view.
8. **Second view:** add the version 1 layout (question 1), including the same
   file in both views.

Any stage that needs a "secondary" or "legacy" code path is a sign that an
earlier stage is incomplete.

Tasks for each stage are tracked in `improvements.md`, under the multiple
documents item in "future ambitions".

## Open questions

Answered on 2026-11-14:

- **2. The same file twice:** allowed. Views share one model (decision 2).
- **3. File types:** all types can be tabs. Only views showing a document
  keep its rendered DOM; others re-render when activated.
- **9. Fallback editor:** removed (decision 7).
- **4. Jupyter in background tabs:** kernels keep running while a notebook is
  not visible and stop when its last tab closes. Block runtimes belong to the
  `NotesDocument`.
- **5. Limits:** no cap on open tabs and no eviction.
- **6. Autosave:** each `NotesDocument` keeps its own autosave timer, including
  in background tabs, so closing a tab never loses work.
- **7. External changes:** a clean open document reloads silently. A dirty one
  is marked as conflicted and the user chooses which version to keep. Both
  apply to every view of the document.

Answered on 2026-11-15:

- **10. Find:** the Find query is kept per document. Project-search results
  (ADR 0019) are not tied to a document and survive tab and focus changes.
- **15. View mode:** chosen per view, so one view can edit a file while another
  previews it. Each view remembers the mode it last used for each document.

Still open:

- **1. Version 1 layout.** More than tabs only, but not full MDI (decision 1).
  Options:
  - tabs plus one fixed side-by-side split (two views);
  - tabs plus recursive splits of document views only, with terminal, AI and
    logs unchanged.

  *Recommendation:* one fixed split. It exercises every boundary MDI needs
  (two views, a shared document, focus following) with minimal UI.
- **8. Persistence.** Are views, their tabs and the focused view restored per
  workspace, using the workspace naming in ADR 0074? How does this interact
  with history and recent files?
- **11. AI context.** Does AI Ask use only the focused document, or can it
  reference other open documents?
- **12. Renames and deletes.** How do ADR 0064's navigation rules apply when the
  renamed or deleted file is open in background tabs or other views?
- **13. Spellcheck.** Does typos-lsp keep every open document open, or only
  visible ones? Aspell runs on demand and only needs the focused document.
- **14. Navigation.** What are the hotkeys for next/previous tab, close tab,
  reopen closed tab and switch view? Does go-to-definition into another file
  open a tab in the focused view or replace the current document?
- **16. Tabs and views.** Does each view have its own tab strip, or is there one
  shared tab list where each view shows one of the tabs?

## Consequences

- One pipeline, with one editor implementation and shared models, avoids both
  previous failure modes. No legacy surface is kept beside a thinner secondary
  one, and no backend services are duplicated.
- Removing the fallback editor removed about 50 `isMonacoActive()` branches
  and the hidden-textarea sync. The remaining `isMonacoActive()` checks only
  mean "Monaco has been created yet".
- The bulk of the work is stages 4–6: moving references from singleton state
  to documents and views. It is mechanical but large, and each stage is
  testable alone before any UI changes.
- Full MDI later means growing the layout tree and adding terminal, AI and logs
  as leaf kinds. Document and view code should not need to change.

## References

- ADR 0036 (branch `experiment/notes-multi-doc`, commit `dae08bd7`) — previous
  in-window MDI attempt
- Branch `experiment/notes-new-window` (commit `5b7ac4a3`) — previous
  window-per-document attempt
- ADR 0016, 0018 — LSP integration and Monaco providers
- ADR 0021 — hidden textarea and the live editor surface
- ADR 0064 — file mutation navigation
- ADR 0074 — workspace-scoped history
- `utils/lsp/document.go` — URI-keyed `DocumentStore`
- `window/frontend/main/lsp.go` — `NotesLsp*` and `NotesTypos*` bindings
- `frontend/src/notes.js` — `closeOpenLspDocument`,
  `openCurrentLspDocument`, `notesLspDiagnostics` handler
