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
- `frontend/src/notes.test.js` — `navigates to selected document symbol from
  editor context menu`, `navigates to selected workspace symbol from editor
  context menu`
- ADR 0018 — Monaco provider rendering
