# 67. Terminal hover is scoped to pointer presence

Date: 2026-10-14

## Status

Accepted.

## Context

Moving the mouse from Terminal into Notes left terminal hover chrome active.
Canvas mouseleave closed the shared tooltip but did not tell Go that the pointer
had left. The renderer retained a valid last mouse cell and replayed its hover
on every redraw. Terminals also retained cached hover drawing callbacks and
the last hovered element.

## Decision

- Canvas `mouseleave` closes the shared tooltip and calls `TerminalMouseLeave`.
  Add its Go method, JS/TypeScript bindings, import and test mock together,
  following ADR 0013.
- The renderer invalidates the last mouse position before clearing hover on
  every visible terminal pane. It restores the base cursor, clears hover status
  text and requests a redraw to erase transient chrome.
- `Term.MouseLeave` clears the cached hover drawing callback and sends
  `MouseOut` to the previous element once, clearing the element reference.
  Repeated leave calls are safe and never write input to the PTY.
- Redraws outside Terminal cannot replay hover from an old cell. A subsequent
  terminal mouse move records a valid position and resumes normal hover.
- Do not change terminal keyboard focus, the active pane or text selection
  merely because the pointer crosses the canvas boundary.

## Consequences

Terminal hover decorations remain active only while the pointer is over its
canvas. This also applies when leaving for terminal tabs, tools or overlays,
not just Notes. OSC 22's base cursor policy remains intact (ADR 0044).

Tests cover frontend leave-to-Notes dispatch, clearing all visible pane hover,
no replay after exit, normal re-entry, idempotent element MouseOut and no PTY
input during cleanup. The binding requires a rebuilt/restarted backend in an
already-running Wails app.

## Reference

- `frontend/src/terminal.js` and `frontend/src/terminal.test.js`
- `frontend.go` and `frontend/wailsjs/go/main/WApp.{js,d.ts}`
- `window/backend/renderer_webkit/mouse.go` and `mouse_test.go`
- `virtualterm/mouse.go` and `mouse_capture_test.go`
- `types/term.go`