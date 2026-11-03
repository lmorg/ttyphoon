# 0021 — Editor navigation must target the live editor surface

## Status

Accepted.

## Context

`Go to symbol` and `Go to workspace symbol` appeared completely inert: the menu
opened, a symbol could be picked, and then nothing visible happened.

Notes has two editor surfaces. Monaco is the default, and the original
`<textarea>` (`elements.editor`) is still present but **hidden** when Monaco is
active — it is retained as the fallback surface and as the value store that
several helpers read from.

Both symbol handlers drove the textarea directly:

```js
elements.editor.focus();
elements.editor.setSelectionRange(offset, offset);
elements.editor.scrollTop = Math.max(0, (line - 2) * lineHeight);
```

Every one of those calls succeeds. No exception is thrown, no console warning is
produced, and the caret genuinely does move — inside an element the user cannot
see. The feature was fully "working" against the wrong surface.

`goToWorkspaceLspSymbol` was doubly misleading: its `await loadFile(target)` step
*did* work, so picking a symbol in another file switched files correctly and only
the caret placement was lost, making the bug look intermittent.

Definition navigation (`requestLspDefinition`) had always branched on
`isMonacoActive()` and so was unaffected — which is why "go to definition works
but go to symbol doesn't" was the observed symptom.

## Decision

Route all programmatic caret/scroll navigation through a single surface-aware
helper, `jumpEditorToOffset(start, end = start)`:

1. `setMainEditorSelectionRange(start, end)` — updates whichever surface is live.
2. If `isMonacoActive()` — `revealOffset` + `focus` on Monaco, then return.
3. Else if `state.useMonacoEditor` — retry on `requestAnimationFrame` for ~20
   frames, because Monaco is created and laid out asynchronously when the editor
   view opens. This matters after `loadFile()`, where the jump can be requested
   before the new model is mounted.
4. Otherwise fall through to the textarea path (`focus` +
   `scrollEditorToSelection`).

The Markdown ToC uses the same navigation path in Edit mode. Its entries map
rendered headings back to source lines, then call `jumpEditorToLine`; in View
and Run modes they continue scrolling the rendered heading instead.

`jumpEditorToLine` is now a thin wrapper that computes the end-of-line offset and
delegates. The default collapsed range (`end = start`) is deliberate:

- **Symbol/definition navigation** places a caret at the symbol.
- **Grep/find navigation** passes an explicit `end` to highlight the whole
  matched line.

Conflating those two is a real regression, not a cosmetic one — see below.

Also removed the silent early return in `goToCurrentLspSymbol`. It previously
bailed without feedback when `state.lspOpenFile !== state.currentFile`, which is
indistinguishable from "the command did nothing". It now attaches the document
first and reports `Language server is not active for the current file` if that
fails.

### Line-aware history and rendered Find

Notes navigation history persists `(filename, line)` entries. Updated 2026-10-21:
history is scoped to tmux window names and supports files outside the active
root; recent files are derived from history (ADR 0074). Same-document
positions at least 20 lines apart create separate entries; nearby positions do
not append another item. Legacy filename-only cache entries migrate to line 1.
Monaco reports its active cursor offset, including reverse selections, and the
textarea fallback reports selection/input events. Position capture is debounced.
The toolbar's Previous/Next actions use the position-aware Go bindings, load
the target file, select its Edit mode, await Monaco initialization, and restore
the saved line through `jumpEditorToLine`. Filename-only compatibility bindings
remain available.

Updated 2026-10-20: automatic workspace history restoration preserves the
document tab restored from the document cache. A saved line greater than 1
does not itself request Edit mode. Restore that line only when the restored
tab is the file type's Edit mode; explicit Previous/Next navigation continues
to select Edit. This prevents terminal pane context changes, including clone
closure, from overriding the document's cached View or Run tab.

Updated 2026-10-21: successful Markdown saves, including autosave, refresh the
Tools ToC and Frontmatter while Edit mode remains selected. Both helpers read
the live editor value rather than the hidden textarea. This refresh does not
regenerate the Markdown preview or Jupyter view; those retain their deferred
rendering behavior. Adding or removing Frontmatter also updates tab visibility.

Rendered Find must likewise target the visible surface: CSV View and CSV Run
search `elements.csvView`, not the hidden Markdown preview. Highlight cleanup
includes the CSV pane, and active queries are reapplied when that pane rerenders.

### Structured Edit Find and Monaco shortcuts

Updated 2026-10-14: JSON, YAML and XML source editing uses `swagger-edit`,
not the ordinary `editor` mode. The Monaco Find/Replace proxy must therefore
check the active Find editor rather than require `viewMode === 'editor'`.
Otherwise structured Edit searches the hidden textarea instead of the live
Monaco model, and match selection/reveal never reaches the visible editor.
Rendered structured View continues using the tree's rendered search path.

Notes handles Cmd+F / Ctrl+F in a dedicated document-level capture listener,
before Monaco's own Find shortcut can consume it. Prevent the default and stop
propagation for that shortcut only, preserving the existing Notes Find-panel
toggle behavior. Terminal focus, an open inputbox and fullscreen image overlays
still leave the shortcut unhandled by Notes. Other shortcuts retain their
existing listeners.

Regression tests enable a mock Monaco adapter and deliberately make the hidden
textarea stale, verifying model searches and visible selection/reveal for JSON,
YAML, YML and XML. Shortcut tests simulate a Monaco-level handler that would
consume the event and verify Notes captures both modifier variants first.

## Consequences

- Symbol navigation works on the visible editor, and survives the
  `loadFile` → Monaco-remount race.
- New navigation entry points should call `jumpEditorToOffset` rather than
  touching `elements.editor`.
- Programmatic insertions, including Markdown emitted for clipboard images,
  must use `insertTextInMainEditor` so Monaco updates its model and emits the
  input event that refreshes the rendered preview.
- Clipboard images cannot rely on a DOM `paste` listener alone: Monaco may
  consume Cmd/Ctrl+V before that event is delivered. Its adapter intercepts the
  primary-paste keydown in the capture phase for Markdown documents, prevents
  Monaco's native paste, and routes to the same Go-backed clipboard handler as
  the custom Paste menu.
- Direct `elements.editor` selection/scroll writes are now limited to the
  low-level helpers (`setMainEditorSelectionRange`), the fallback branch, and
  full-document resets.

## Lessons

- **A hidden fallback surface makes "wrong target" bugs silent.** There is no
  error to catch when you successfully manipulate an invisible element. The
  absence of a stack trace is not evidence that the code path is fine.
- **Compare against the sibling that works.** `requestLspDefinition` already had
  the correct `isMonacoActive()` branch; the divergence between it and the symbol
  handlers localised the bug faster than reading either in isolation.
- **Extracting a shared helper changes defaults for every caller.** Folding the
  symbol jump into `jumpEditorToLine`'s body silently gave symbol navigation
  grep's select-to-end-of-line behaviour. Two pre-existing tests caught it
  (`expected 44 to be 30`). Preserve each caller's semantics explicitly when
  unifying code paths — the tests were right and the refactor was wrong.

## Reference

- `frontend/src/notes.js` — `jumpEditorToOffset`, `jumpEditorToLine`,
  `goToCurrentLspSymbol`, `goToWorkspaceLspSymbol`, `requestLspDefinition`
- `utils/notes/history.go` — line-aware entries and legacy cache migration
- `frontend.go` — position-aware Notes history bindings
- `frontend/src/notes.test.js` — `navigates to selected document symbol from
  editor context menu`, `navigates to selected workspace symbol from editor
  context menu`, `jumps from the ToC to a heading source line in Markdown Edit
  mode`
- ADR 0018 — Monaco provider rendering
