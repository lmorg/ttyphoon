# 0059 - Notes layout sizing and persistence

## Status

Accepted and implemented on 2026-10-07.

## Context

Notes has a resizable boundary between its file list and editor, and the main
window has a resizable boundary between Notes and the terminal. User-selected
dimensions should survive reopening the application rather than reverting to
defaults. The Tools panel also needs a user-operated resize handle and caching
of its chosen dimensions.

These are layout preferences, separate from file contents, document navigation
state, and transient panel visibility. Fullscreen/maximize and collapse modes
temporarily change geometry (ADR 0015), so blindly persisting measured sizes
would overwrite the user's normal split with temporary dimensions.

## Decision

- Cache and restore the `file list | notes editor` splitter size.
- Cache and restore the `notes | terminal` splitter size.
- Add Tools-panel resizing and cache/restore the resulting dimensions.
- Use application-wide preferences independent of workspace and document.
  `notes.LayoutSizing` stores `fileListRatio`, `notesRatio`, and `toolsRatio` in
  `NS_NOTESW_LAYOUT`, under the `application` key with the existing one-year TTL
  policy. These are persistent UI preferences, not the directory snapshots
  declined in ADR 0058. Per-field updates are validated and mutex-protected so
  independent frontend callers cannot overwrite each other's values.
- Prefer container-relative ratios for splitters so restoration adapts to a
  different window size. Clamp restored and dragged dimensions to usable pane
  minimums and the available container size; missing or invalid preferences
  fall back to current defaults.
- Restore normal-layout preferences when the relevant container is measurable,
  then trigger existing editor layout and terminal resize handling as needed.
  Handle application restart and workspace switching according to the selected
  preference scope.
- Save completed user resize operations, or coalesce writes during a drag;
  avoid a cache write for every pointer movement. Window resizing may re-clamp
  the rendered dimensions without replacing the user's preferred ratio.
- Keep normal-layout sizing independent of fullscreen, maximized, hidden, and
  collapsed geometry. Such transitions must not overwrite cached preferences
  or persist panel open/closed state as a side effect of dimension caching.
- Updated 2026-10-22: preserve the selected Tools tab across document, project
  root and workspace changes while that tab remains available. Do not force
  Find on document load or AI on background job events; opening the document
  Diff surface also preserves Tools selection and collapsed state. When ToC or
  Frontmatter genuinely becomes unavailable, select Find without closing the
  Tools panel. Git is always a visible/selectable tab: unavailable operations
  hide its toolbar and show their reason instead of changing Tools selection
  (ADR 0062).
  Explicit tab clicks and dedicated Find/Ask commands remain user navigation.
  Startup still defaults to Find; selection is not restored per document.
- Ensure the Tools-panel resize handle does not interfere with its tabs,
  scrolling, collapse controls, or AI/Log maximization. Use the existing
  splitter interaction conventions where applicable.

### Implemented Controls

- The Tools panel is bottom-docked and resizes vertically through a separator
  above its tabs. Save on mouse-up; Up/Down keys resize in small increments.
  It reserves normal minimums for Tools and editor content, reducing those
  minimums when the container cannot accommodate both.
- The file-list splitter supports Left/Right keys and double-click reset.
  Its existing normal minimums are retained; very narrow Notes panes fit both
  columns rather than leaving stale pixel widths overflowing the container.
- Tools double-click resets its default ratio. Reset is stored as zero, meaning
  use the current default on restoration. The outer Notes/terminal splitter
  retains double-click collapse/restore and supports Left/Right resizing.
- All three restore container-relative ratios. Resize observers and window
  resize handlers re-clamp rendered dimensions without rewriting the preferred
  ratio. Collapsed Notes restores its preferred outer ratio rather than forcing
  the midpoint. Maximize/restore does not save temporary pane widths.
- `layout-sizing.js` shares one initial cache read and tracks newer local
  choices. A late cache response cannot replace a completed user drag. Tools
  sizing is not applied or saved while collapsed or AI/Log-maximized; its
  resize handle is hidden in those modes.
- `GetNotesLayoutSizing` and `SetNotesLayoutRatio` provide the Wails bridge.
  Invalid field names and non-finite/out-of-range ratios are rejected;
  missing/invalid cached ratios use existing defaults. Binding failures are
  caught without preventing the user from resizing the current view.
- AI and Log maximization hide the entire Tools tab-bar header, not a list of
  individual tab buttons. This also hides Git, Frontmatter, future tabs and the
  header's background/border without adding new fullscreen selectors for each
  tab. The maximized pane's own toolbar remains visible, including its restore
  button. Restoring normal layout makes the tab bar visible again. Regression
  tests cover both modes, explicitly visible Git/Frontmatter and an added tab.

## Consequences

- Users retain their selected layout instead of repeatedly resizing panes.
- Layout restoration must coordinate with the temporary style snapshots used
  by maximize/restore and collapse behaviour.
- Verification should cover drag/save/restore, application restart, workspace
  switching, smaller windows, invalid cached values, and maximize/collapse
  transitions without preference corruption.
- Regression tests cover independent concurrent cache writes, validation,
  cached restoration, drag-end-only saves, keyboard/reset, narrow-window
  clamping, late responses, and collapse/maximize preservation. The improvement
  checklist is marked complete.

## Future Options

- Per-window/workspace preferences and configurable TTL/minimums are not part
  of this implementation; the chosen application-wide scope stays consistent
  while switching workspaces.

## Reference

- [improvements.md](improvements.md) - requested sizing and caching features
- `frontend/src/notes.js` - internal splitter and Tools-panel state
- `frontend/src/ttyphoon.js` - Notes/terminal split and maximize/restore
- `utils/notes/cache.go` - existing Notes cache infrastructure
- `utils/notes/cache_test.go` - validation and independent updates
- `frontend/src/layout-sizing.js` - shared restore/save handling
- `frontend/src/notes.test.js`, `frontend/src/ttyphoon.test.js` - layout regressions
- ADR 0015 - pane maximize and temporary layout geometry
- ADR 0013 - Wails binding checklist