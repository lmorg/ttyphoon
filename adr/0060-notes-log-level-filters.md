# 0060 - Notes log filters are independent by level

## Status

Accepted and implemented on 2026-10-09.

## Context

The Notes Tools log panel previously had one toggle for trace output. Users need
separate control over error, warning, info, debug, and trace lines, as well as a
way to select the currently visible log lines without selecting filtered-out
entries.

## Decision

- Provide independent visibility toggles for `error`, `warn`, `info`, `debug`,
  and `trace`. Debug and trace are separate controls.
- Keep error, warning, and info visible by default; keep debug and trace hidden
  by default.
- The Select all control selects only lines allowed by the current level
  filters. Copy continues to copy selected lines when any are selected.
- Use the lowercase level initials `e`, `w`, `i`, `d`, and `t` in the toolbar,
  rendered through the Font Awesome font stack.
- Group toolbar controls by selection, level filters, display, and actions, with
  visible separators between groups.

## Consequences

- A new log level must be represented in the filter state, rendered-line
  visibility rules, Select all visibility check, and tests.
- Filter preferences are transient and reset to the defaults when the log panel
  is initialized.

## Reference

- `frontend/src/notes.js` - log toolbar markup
- `frontend/src/notes-log-panel.js` - level visibility and visible-line selection
- `frontend/src/notes.css` - level filtering and toolbar presentation
- `frontend/src/notes-log-panel.test.js`, `frontend/src/notes.test.js` - behavior
  and toolbar regressions
