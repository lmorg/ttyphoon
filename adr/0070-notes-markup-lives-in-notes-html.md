# 70. Keep static Notes markup in `notes.html`

Date: 2026-10-15

## Status

Accepted.

## Context

`frontend/src/notes.js` owns Notes behavior, state, event handling, and rendering,
but it also contained the full static Notes interface as a large template literal.
That made the document structure harder to scan independently from its behavior.
The markup is for the Notes interface, not the terminal, so a name containing
"shell" would be ambiguous.

## Decision

- Keep the static Notes interface markup in `frontend/src/notes.html`.
- Import it into `notes.js` as a string with Vite's `?raw` import and inject it
  into the Notes app root during initialization.
- Keep behavior, element lookup, and dynamic or data-dependent HTML rendering in
  JavaScript. Do not split small dynamic fragments into additional template files.
- Keep styles in the existing Notes CSS; `notes.html` contains structure only.

## Consequences

The static interface can be read and edited separately while remaining part of
the normal Vite bundle; no runtime HTML fetch or terminal-shell abstraction is
introduced. Element IDs and classes in `notes.html` are an interface used by
`notes.js`, styles, and tests, so changes to that structure must preserve or
update those consumers.

## Reference

- `frontend/src/notes.html` - static Notes interface markup
- `frontend/src/notes.js` - imports and injects the markup; owns behavior
- `frontend/src/notes.test.js` - Notes rendering and structure tests