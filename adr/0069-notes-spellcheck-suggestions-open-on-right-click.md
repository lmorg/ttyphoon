# 69. Notes spellcheck suggestions open on right-click

Date: 2026-10-14

## Status

Accepted.

## Context

Notes rendered spelling suggestions when the user left-clicked a misspelled word.
This interrupted ordinary caret placement and text selection. Notes already
uses right-click for editor context menus, so suggestions should use that
gesture only when the pointer is over a spelling underline.

## Decision

- Open the spelling suggestion menu on `contextmenu` over a detected misspelled
  word, positioned below the word as before.
- Prevent the browser menu and stop propagation only when a misspelling is hit,
  so the suggestion menu replaces the general Notes editor menu at that point.
- Do not show suggestions on left-click. Right-clicks elsewhere continue to
  the existing editor context-menu handler.
- Apply a selected suggestion through the existing text replacement and input
  event path so spellcheck and editor state refresh normally.
- Monaco follows the same right-click rule. Its capturing `contextmenu`
  listener resolves the pointer to a model position and intercepts the event
  only for a misspelling, before the general editor menu. Left mouse-down and
  click remain untouched; dispose the listener with the editor adapter.

### Native Text Assistance (2026-10-24)

- Use `frontend/src/text-input.js` to disable browser/OS autocomplete,
  autocorrection, capitalization, native spellchecking, and writing suggestions.
  Apply the existing Grammarly opt-out attributes through the same helper.
- Notes initializes its static text controls through the shared helper. Dynamic
  graph fields, worktree paths, editable table cells, inputboxes and string
  variables, JSON/XML inline editors, Swagger request fields, and popup filters
  apply it when created or rebuilt.
- `attachSpellCheck` also applies the policy and supports single-line inputs
  without wrapping their measurement mirror. Commit messages and graph titles
  use the application's checker, as do the existing supported editor and
  multiline prompt surfaces.
- Filters, paths, structured keys/values, single-line generic prompts, and
  variable controls do not gain dictionary checking merely because native
  assistance is disabled. These fields can contain syntax or identifiers rather
  than prose.
- Detach the graph-title checker before removing its temporary form. The commit
  field persists with the Notes panel; when a commit clears its value, emit an
  input event so its checker and other input listeners refresh.

## Consequences

Ordinary left-clicks remain available for editing; misspellings expose their
suggestions through the pointer's context-menu gesture. Tests cover both the
spelling hit and pass-through behavior elsewhere.

## Reference

- `frontend/src/text-input.js` - native text-assistance policy
- `frontend/src/spellcheck.js`
- `frontend/src/spellcheck.test.js`
- `frontend/src/notes.js` - general editor context menu
- `frontend/src/monaco_adapter.js` - Monaco spelling context-menu listener
- `frontend/src/monaco_adapter.test.js` - mouse gestures, pass-through, and cleanup