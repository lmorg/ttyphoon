# 0100 - Notes: multiple open documents

Date: 2026-11-14

## Status

Accepted and implemented on 2026-11-15 (stages 1 to 8).

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
must also not attempt full MDI in one change.

*Decided (2026-11-15, question 1):* version 1 has tabs plus recursive splits
of document views. A view can be split side by side or stacked, and any view
in a split can be split again, in either direction. Terminal, AI and logs stay
where they are; making them leaves of the same layout tree is the later move
to full MDI.

### 2. Three layers: document, view, layout

Separate state into three layers from the start, even while the UI only uses
one view:

- **`NotesDocument`**, one per open file, keyed by absolute path. All views of
  that file share it: the Monaco model, dirty state, autosave, LSP
  registration and diagnostics.
- **`NotesView`**, one per visible editor pane. It owns the editor instance,
  its preview and structured-view host, its tab list, the active document, and
  per-document view state (cursor, scroll, view mode).
- **Layout**, a serialisable tree. Each leaf is a `NotesView`. Each branch is
  a split with a direction (side by side, or stacked), two or more children,
  and their relative sizes. Version 1 only has view leaves; full MDI later adds
  terminal, AI and logs as leaf kinds.

  The tree is kept normalised: a split with one child is replaced by that
  child, and a split inside a split of the same direction is merged into its
  parent. Closing a view's last tab closes the view, unless it is the only
  view.

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

*Done (Stage 6, task 3):*

- `focusedView()` in `notes.js` returns the view that global chrome follows;
  for now it is the one view. The LSP code reaches the editor and its host
  through it (`getMonacoMainEditor`, `getFocusedEditorHost`) to place hover,
  completion and LSP-menu overlays. The status bar and Tools panel show the
  active `NotesDocument`, which is the focused view's document.
- The Find query, its matches and current match are `NotesDocument` fields
  (question 10). Loading a document shows its own query: switching files
  starts with an empty query, and reloading the shown file keeps its query
  and finds it again. Search options (case, regex, whole word) stay global.
- Loading a file no longer clears project-search results, which belong to no
  document, so they survive switching files. The `keepFindTabOpen` load
  option, which existed only to protect them, is gone.
- Until Stage 7 keeps documents open, a document's Find query is lost when
  another file replaces it.

*Done (Stage 7, task 1):*

- Each `NotesDocument` owns its own buffer and Monaco model
  (`doc.editorModel`). The model is created the first time a view shows the
  document (`showDocumentInView`), and the buffer attaches to the model handle
  rather than to the editor. A document that is not shown can still be edited
  through its buffer, and it keeps its own undo history.
- `createDocumentModel()` and `showModel()` in `monaco_adapter.js` create
  models and swap them in with `setModel()`. When a model is hidden, its
  selection is saved, and it is restored when the model is shown again. Typos
  decorations are cleared before each swap. A placeholder model is shown when
  no document is.
- The `mainDocument` that `notes.js` keeps is now a facade over the active
  document's buffer. Its change listeners fire only for the active document.
- Disposing a model after its editor has already been disposed is safe.
- Checked in real Chrome against real Monaco: model swapping, edits to hidden
  models, separate undo histories, separate model URIs, selection restore and
  dispose order.
- Stage 8 limitation: a model handle routes selection through the adapter that
  created it. Before two views can share one model, selection must be tracked
  per view.

*Done (Stage 7, task 2):*

- `RunNote(docPath, id, runID, code, language)` takes a run ID that the
  frontend makes unique across open documents. The backend keys its kill
  functions by run ID, `StopNote(runID)` stops a run, and `noteRun` and
  `noteComplete` carry `runId` as well as `blockId`. Two documents can now run
  blocks with the same block ID at the same time.
- In `notes.js`, `jupyterRuns` maps each run ID to its document and block, and
  `doc.runningJupyterBlocks` maps each block to its current run ID. Late events
  from a run whose document has closed are ignored. A completed run does not
  reset the buttons of a newer run of the same block.
- Output is still shown only for runs in the active document. Once the tab
  strip keeps documents open in the background, it must keep their output too,
  so that switching back to a document shows output from runs that continued
  while it was hidden.

*Done (Stage 7, task 3):*

- Each `NotesView` keeps an ordered list of tabs, one per document key. Opening
  a file from the file list, history or go-to-definition opens a tab after the
  current one, or focuses the file's tab if it is already open. A document
  stays open while a view has a tab for it, and is closed (releasing its model,
  LSP and typos documents) once no view does.
- Switching tabs saves the outgoing document if it is dirty, and flushes its
  pending LSP and typos changes. Background tabs are therefore normally clean.
  A background document is dirty only if its save failed, and switching back
  to it keeps its buffer instead of the disk text.
- Switching back to a document keeps its model, so undo history survives.
  `loadFile` only replaces the text when it differs from the disk text.
- Document tabs match the terminal tab strip and show a dot for unsaved edits.
  They have no close button; use the tab's context menu, middle-click or Cmd+W.
- Mode tabs (View, Edit, Run, etc.) and Tools tabs (Find, Git, etc.) instead
  use the flat Swagger request/response tab style: muted inactive text and an
  accent underline on the selected tab, with accent-coloured text on hover.
  Document and terminal tabs retain
  their rounded, bordered style.
- `.notes-mode-tab` and `.tools-tab` share the underline styling.
  `.notes-toolbar-btn` owns toolbar-button styling independently; do not use
  descendant button selectors on the mode bar, which also hosts the toolbar.
- Document toolbar buttons share the Git toolbar icon buttons' border,
  padding and hover treatment, but use the normal Notes font size for larger
  icons rather than the Git buttons' two-pixel-smaller font. New keeps its green highlight;
  maximize uses the Git toggle buttons' selected treatment while enabled.
  The toolbar is vertically centred in the mode bar, whose background uses
  the normal theme background rather than the document strip's dark overlay.
- The focused Notes view has an accent outline in both single-view and split
  layouts, hidden while the terminal has focus. Monaco's inner editor-shell
  border does not change colour on focus. Unfocused Monaco editors are
  darkened with a non-interactive overlay above their opaque backgrounds,
  also shown when the terminal owns focus.
- Monaco spelling decorations retain the original SVG undercurl to match the
  terminal: an 8-by-6-pixel repeating tile with a 1.5-pixel stroke. Mask a
  non-interactive `::after` layer, not the inline text span; masking the span
  clips the misspelled word itself. Do not substitute native wavy text
  decoration, whose geometry differs.
- Closing a tab saves the document first if it is dirty, and keeps the tab if
  the save fails. Closing the active tab opens its neighbour; closing the last
  tab leaves no document open and hides the strip.
- Changing workspace closes the background tabs. Persisting tabs per workspace
  (Q8) will replace this.
- `doc.jupyterOutputs` records each block's output, so output from a run that
  continues while its document is in the background shows when the document
  is reopened.
- `loadFile` no longer clears `state.currentFileUri` before switching, which
  had wiped the outgoing document's URI.

*Done (Stage 7, task 4):*

- The backend watches the files of every open document, not just the last
  file read. `GetFile` no longer changes what is watched. Instead the frontend
  sends the open documents with `WatchNotesFiles(generation, files)` whenever
  the set changes; a call older than the last one applied is ignored. The
  `notesFileChanged` event carries `{path, keys}`, the keys of the documents
  open on that file.
- The watcher keeps a self-write expectation per file, kept until it expires,
  so saving several documents together, or one write raising several events,
  is not reported as an external change.
- A clean active document reloads when its file changes, as before. A clean
  background document needs nothing: showing a tab reads the file, so it shows
  the disk version when next activated.
- A document with unsaved edits is conflicted when its file changes, unless
  the disk text matches its edits. A conflicted document is never saved
  automatically: autosave, saving on switching tabs and saving on closing a tab
  all leave the disk version alone. Its tab is marked, and while it is active a
  banner in the view offers "Keep my edits" (save over the disk version) or
  "Load disk version" (discard the edits). Saving with Cmd+S keeps the edits.
- Showing a background document whose edits now match the disk marks it clean.

*Done (Stage 7, task 5):*

- Renaming a file or folder moves every open document inside it, active or in
  the background, to its new path. Each tab keeps its place and its remembered
  view state, and the watch set follows. Unsaved edits are saved before the
  rename so they move with the file. The active document reloads from its new
  path, keeping any edits that could not be saved.
- The backend now reports deletes. A missing file is checked again after a
  short delay, and reported as `{path, keys, deleted: true}` only if it is
  still missing, so an editor that replaces a file is not reported as deleting
  it. An external rename looks like a delete, because fsnotify does not link
  the old and new names.
- When an open document's file is deleted, in the app or outside it, a clean
  document's tab closes. A document with unsaved edits stays open with its tab
  marked deleted, and is never saved automatically. The view's banner offers
  "Recreate file" (save it, which recreates the file) or "Discard edits" (close
  the tab). Saving with Cmd+S also recreates it, and closing the tab drops the
  edits.
- Deleting the active file activates its neighbouring tab. History (ADR 0064)
  decides what opens only when no tabs remain.

*Done (Stage 7, task 6):*

- With Notes focused, Cmd+Shift+] and Cmd+Shift+[ step to the next and
  previous document tab, wrapping round. Cmd+W closes the active tab, saving
  it as closing it from the context menu does. Cmd+Shift+T reopens the most
  recently closed tab that is not open again; up to 20 closed tabs are remembered.
  Tabs closed because their file was deleted, or whose edits were discarded,
  are not remembered. Ctrl+Tab still cycles the Notes mode tabs.
- The keys are handled by Notes in the capture phase, so they do not reach the
  terminal or Monaco, and do nothing while a Notes dialog is open.
- The command palette lists the same four commands as "Notes: ..." entries.
  They send a `notesDocumentTabCommand` event naming the command.

*Done (Stage 7, task 7):*

- Each workspace (tmux window name, ADR 0074) remembers its open tabs and
  focused tab, saved whenever the tab strip changes. They are kept with the
  workspace's history in `notesw_history`, under `tabs:workspace:<name>`, as
  expanded filenames, and returned with Notes aliases, as history is. Up to 50
  tabs are kept, for 30 days.
- Opening Notes, or switching to a workspace, restores its tabs in place of its
  history position. Only the focused tab's file is read; the others are read
  when first shown. A workspace with no remembered tabs restores its history
  position as before.
- Saving waits until the workspace's tabs have been restored, so the empty
  strip at startup or during a switch never overwrites them. The backend also
  ignores a save that names a workspace other than its current one.

*Done (Stage 7, task 8):*

- typos-lsp only has the shown document open (Q13). Moving a document to a
  background tab closes its typos document, and showing it again reopens it.
  Language LSP documents stay open for every open document, as before.
- AI Ask sends only the focused document's contents (Q11). It already did, so
  this needed no change.

*Done (Stage 8, task 1):*

- `notes_layout.js` holds the layout tree as plain data: view leaves
  `{ type: 'view', id }` and split branches `{ type: 'split', direction,
  children, sizes }`, where `direction` is `row` (side by side) or `column`
  (stacked) and `sizes` are shares summing to 1. Every operation returns a new
  tree.
- Splitting a view inside a split of the same direction adds a sibling that
  takes half of the view's share; otherwise the view becomes a new two-way
  split. Closing a view gives its share to its neighbours in proportion, then
  collapses one-child splits and merges same-direction nesting. The only view
  is never closed.
- Dividers are addressed by the split's path of child indices and the index of
  the child before them. Resizing clamps both sides to a minimum share;
  equalising gives the two sides equal shares.
- `parseLayout` rebuilds saved data, dropping unknown nodes and duplicate view
  ids and repairing bad sizes. A `parseView` callback keeps per-view data,
  which task 9 uses for tabs.
- Not yet used by the UI.

*Done (Stage 8, task 2):*

- `notes_layout_dom.js` renders a layout tree into `#notes-views`. Views are
  placed by their root elements; each split is a `.notes-layout-split` flex
  container with a `data-direction`, and its children's `flex-grow` is their
  share. A single view is placed directly in the host, so the DOM is unchanged.
- Rendering only moves an element that is not already in place, because
  moving a view's root reloads its iframes and resets Monaco's layout. Changes
  to the tree's structure still move the views whose parent changes.
- `notes.js` keeps `notesViews` (views by ID) and `notesLayout`, creates the
  main view detached, and renders the one-view layout.

*Done (Stage 8, task 3):*

- Split right (`Cmd+\`, a tab's context menu, the command palette) and Split
  down (context menu, palette) add a view beside or below the focused one. The
  new view gets a tab for the focused document, with the source view's
  remembered mode and editor state, and takes focus.
- One view is focused at a time. The toolbar moves into it, and the active
  document, `state`, the `elements` view parts and global chrome follow it.
  Focusing a view reloads its document into it with `loadFile`, keeping unsaved
  edits.
- Views of one document share its model, so edits show in every view. Each
  Monaco editor keeps its own selection in each model it has shown. A model's
  get/set selection and `replaceRange` go through the editor that last showed
  it (`activeEditor`). Disposing a model hands every editor showing it the
  placeholder.
- Each view binds its own mode tabs, tab strip, context menus, scroll and
  resize observers (`bindNotesView`). Theme and typography changes reach every
  view's editor.
- A document stays open while any view has its tab. Switching workspace
  removes the other views.

Known limitations, for the following tasks:

- An unfocused view keeps its last rendered DOM. Its Monaco editor stays live,
  but previews, Jupyter output and LSP decorations update only when it is
  focused again.
- The first click on an unfocused view only focuses it. Clicks on its tab
  strip act straight away.
- Only the first view's tabs are remembered per workspace (task 9).
- A view stays when its last tab closes (task 5), and its size is fixed
  (task 6).

*Done (Stage 8, task 4):*

- Cmd+Option+arrows focus the neighbouring view. `neighbourLayoutView` works
  out each view's rectangle from the split sizes and picks, of the views
  touching that edge, the one sharing most of it. The editor takes keyboard
  focus when the view shows it.
- The keys are taken only when there is a view on that side. Otherwise Monaco
  keeps Cmd+Option+Up/Down for adding a cursor above or below.
- With more than one view, the focused one is outlined in the accent colour
  and the others are dimmed. Clicking a view focuses it (task 3).
- The tab hotkey handler ignores keys once its Notes app is no longer in the
  page.

*Done (Stage 8, task 5):*

- Closing a view's last tab removes the view and the tree collapses around it,
  unless it is the only view, which stays empty. A focused view first hands
  focus to its neighbour on the left, above, right or below, in that order.
- A document closes when no view has its tab. Moving focus off a removed view
  releases its document through `setActiveNotesDocument`, as switching tabs
  does.

*Done (Stage 8, task 6):*

- Each split has a `.notes-layout-divider` between its children, carrying the
  split's path and the divider's index. Dividers are reused on re-render, so
  views are not moved.
- Dragging a divider resizes the views either side of it, measured from the
  layout as the drag began, with pointer capture so iframes and Monaco don't
  take the drag. No view shrinks below 120px. Double-clicking a divider gives
  the two views equal shares.
- Views have no depth limit. Monaco relayouts through each view's
  ResizeObserver.

*Done (Stage 8, task 7):*

- Document tabs are draggable. Dropping one on a tab strip places it before
  the tab under the pointer (after it past its middle, or last on empty
  strip space). Within one view this reorders the tabs.
- Dropped on another view, the tab joins that view, which takes focus showing
  the document, and leaves its source without a save or a reopen entry, as
  the document stays open. A view emptied this way closes.
- The drag uses a module-level record of the dragged tab rather than
  `dataTransfer`, which can't be read during `dragover`.
- A temporary cloned tab supplies `dataTransfer.setDragImage`, with all four
  corners rounded to 5px, a 1px border on every edge and opacity 0.75. Its
  separate `.notes-doc-tab-drag-image` class replaces the shared `.tab` border
  shape without changing the source tab. It keeps the source dimensions and
  pointer offset, and is removed on the next animation frame after the browser
  snapshots it. The tab-bar and terminal-tab styles are unchanged.
- Dragging onto any of a view's four outer quarters previews the new half
  with an accent overlay; dropping splits that target on the indicated side
  and moves the tab into the new view, removing it from its source. Left and
  top drops use `splitLayoutView`'s `before` option; right and bottom drops use
  the usual after ordering. The new view takes focus and retains the dragged
  document's remembered mode, selection, edits and model undo history.
- At every corner the nearest proportional edge wins, with left/right winning
  exact ties. `data-drop-split` identifies the edge
  (`left`, `right`, `top`, `bottom`), while the layout still uses row/column.
  Tab strips retain reorder/move priority, even along the top or left edge.
  Dragging a view's only tab onto its own edge does nothing, rather than
  leaving an empty view. Preview markers clear on leaving, cancellation or drop.

*Done (Stage 8, task 8):*

- Edits, renames, deletes and LSP diagnostics already reached every view:
  views share the document's model, and markers are set per model.
- An unfocused view keeps the DOM it last rendered. A View or HTML view
  showing a document that changes, by an edit in another view or on disk, is
  re-rendered after a short debounce. A clean document changed on disk that
  only unfocused views show reloads in place.
- Jupyter output is recorded on the document. An unfocused Run view shows it
  when focused, as its code blocks are bound to the focused view.
- Typos underlines are model decorations, so they show in every view of a
  document. A document's typos registration and underlines stay while any
  view shows it, and are released once none does.
- A view focused and then left before its document finished loading keeps
  tracking that document.

*Done (Stage 8, task 9):*

- Each workspace's cache entry now stores a list of views (an ID, its tabs and
  its active tab), the serialised layout tree and the focused view ID. There
  are at most 16 views and 50 tabs per view.
- Stage 7 entries (a flat `Files`/`Active` list) are read as a single view, so
  existing tab lists survive the upgrade.
- On restore, the layout is parsed keeping only views that were saved, and it
  falls back to the first view alone if the tree is unusable. Each view shows
  its active tab, and the saved focused view takes focus.
- Saves happen when tabs, views or focus change. Divider drags are saved after
  300ms without movement.

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
8. **Split views:** add the version 1 layout (question 1): recursive splits
   in both directions, including the same file in several views.

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

Answered on 2026-11-15 (Stage 7):

- **8. Persistence:** open tabs and the focused tab are restored per
  workspace, using ADR 0074's workspace naming. Navigation history and recent
  files work as before.
- **11. AI context:** AI Ask uses only the focused document, as now.
- **12. Renames and deletes:** a renamed file's tabs follow the new path in
  every view. A deleted file's clean tabs close; dirty tabs stay open, marked
  deleted, and saving recreates the file. ADR 0064's history rules still apply.
- **13. Spellcheck:** typos-lsp keeps only documents shown in a view open, and
  reopens a document when it is shown again.
- **14. Navigation:** with Notes focused, Cmd+Shift+] and Cmd+Shift+[ select
  the next and previous tab, Cmd+W closes the tab and Cmd+Shift+T reopens the
  last closed one; each is also in the command palette. Ctrl+Tab still cycles
  the mode tabs (View, Edit, …). Go-to-definition into another file opens or
  focuses a tab in the focused view.
- **16. Tabs and views:** each view has its own tab strip. A file can be a tab
  in both views; the tabs share one `NotesDocument`.

Answered on 2026-11-15 (Stage 8), updated 2026-11-17 for four-edge tab drops:

- **1. Version 1 layout:** tabs plus recursive splits of document views, side
  by side or stacked, to any depth (decision 1). Terminal, AI and logs are
  unchanged.

- **17. Splitting:** "Split right" and "Split down" in the
  tab context menu and the command palette, with `Cmd+\` for Split right as in
  VS Code. The new view opens the same document. Dragging a tab to any edge
  instead splits and moves that tab into a new view on that side. Top and left
  drops insert before the existing tab group; bottom and right insert after it.
- **18. Moving focus between views:** clicking a view
  focuses it, and Cmd+Option+arrow keys move focus to the neighbouring view
  in that direction.
- **19. Moving tabs:** a tab can be dragged onto another
  view's tab strip, or onto any pane edge to split and move it. "Move to other
  view" is not needed beyond that.
- **20. Sizes:** dividers can be dragged, double-clicking a
  divider makes its neighbours equal, and a view cannot shrink below a
  minimum width or height. There is no depth limit.
- **21. Persistence:** the whole layout tree, with each
  view's tabs and the focused view, replaces Stage 7's per-workspace tab list.
  Saved tab lists from Stage 7 load as a single view.

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
