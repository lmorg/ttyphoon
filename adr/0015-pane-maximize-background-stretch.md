# 15. Pane maximize stretches the background pane

Date: 2026-09-06

## Status

Accepted

## Context

Maximizing notes and maximizing the terminal behaved differently: with notes
maximized the terminal filled the window behind the dimmed overlay, but with the
terminal maximized the notes pane stayed at its split width, leaving dead space.

The maximize code was not the cause. `enterNotesFullsize` and
`enterTerminalFullsize` were near-identical copies, both lifting their pane into
a `position: fixed` overlay on `document.body`.

The asymmetry came from the split layout itself. `contentWrapper` is a flex row:

```js
notesPane:    'width:50%' … 'flex-shrink:0'   // fixed width
splitHandle:  'width:2px' … 'flex-shrink:0'
terminalPane: 'flex:1'                         // absorbs remaining space
```

Lifting a pane out removes it from that row. Removing notes lets the terminal's
`flex:1` expand to fill the window — correct by accident. Removing the terminal
leaves notes pinned at `width:50%` with `flex-shrink:0`, so nothing expands.

Two smaller inconsistencies compounded it: `splitHandle` stayed visible in both
cases, and the two buttons signalled their active state by different mechanisms
(notes mutated inline styles, the terminal set `data-enabled`). The notes exit
path also cleared `color` and `backgroundColor` but not `borderRadius`, leaking
state after exiting.

A third defect only became visible once the panes were otherwise consistent: the
maximized terminal had top and bottom margins but none on the left or right.
`terminalPane`'s base style is `flex:1`, which expands to
`flex-grow:1; flex-basis:0%`. Inside the overlay's flex container, `flex-basis`
**overrides `width`**, so `calc(100vw - 100px)` was ignored and the pane grew to
the full overlay width. Height was unaffected, because `flex-grow` only acts
along the main axis of a row. Setting `flex-shrink:0` does not help — it leaves
`flex-grow` and `flex-basis` untouched. `notesPane` has no `flex-grow`, so its
width was always honoured; the bug was invisible on that side.

## Decision

Maximizing is one parameterised implementation, `enterFullsize(kind)` /
`exitFullsize()` / `toggleFullsize(kind)`, keyed by `'notes'` or `'terminal'`.

It explicitly manages the pane left behind rather than relying on flex defaults:

- the background pane is saved and set to `width:100%; flex:1; flex-shrink:1`
- `splitHandle` is hidden
- both are restored from the snapshot on exit

The maximized pane is set to `flex: '0 0 auto'` so its explicit width applies
regardless of the pane's base flex configuration. `flex` is included in the
saved/restored style snapshot alongside `width` and `flex-shrink`.

Restore state lives on the overlay element (`_savedParent`, `_savedStyle`,
`_backgroundStyle`, `_splitDisplay`), so it cannot desynchronise from the overlay
it belongs to.

Only one pane may be maximized at a time; `toggleFullsize` exits any active
pane first. Both buttons now signal active state through `data-enabled`, with
matching CSS rules, so no inline style is left behind on exit.

## Consequences

- Both directions look the same: the other pane fills the window behind the
  dimmed border, and the maximized pane has an equal margin on all four sides.
- One implementation to change instead of two copies that had already drifted.
- **When sizing a flex child, setting `width` is not enough.** Any `flex-grow` or
  `flex-basis` on the element wins. Neutralise the `flex` shorthand explicitly.
- The background pane's inline styles are temporarily overwritten. This is safe
  against splitter drags (which set `notesPane.style.width` and are restored on
  exit), but any future code that mutates pane width *while* a pane is maximized
  would have its change reverted on exit.
- `setTerminalJupyterMode` and `applyNotesCollapsed` perform their own
  save/restore of the same properties. They are not currently reachable while
  maximized, but a third overlapping mechanism would need reconciling.
- Geometry is covered by regression tests, since these defects are visual and
  otherwise only caught by eye.

## Reference

- `frontend/src/ttyphoon.js` — `enterFullsize`, `exitFullsize`, `toggleFullsize`,
  pane construction in the split layout
- `frontend/src/notes.css` — `#notes-fullsize-btn[data-enabled="true"]`,
  `#terminal-zoom-btn[data-enabled="true"]`
- `frontend/src/terminal.js` — zoom button, which also calls `TerminalPaneZoom()`
- Tests: `neutralises flex growth on a maximized pane so its width applies`,
  `maximizes notes with the same geometry as the terminal`
  (`frontend/src/ttyphoon.test.js`)
