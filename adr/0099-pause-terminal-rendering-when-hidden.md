# 0099 - Pause terminal rendering while the window is hidden

Date: 2026-11-11

## Status

Accepted. Items 1 and 2 are implemented. Item 3 is deferred.

## Context

The app used about 30% CPU while the window was hidden.

The old SDL renderer stopped rendering while the window was hidden (commit
`ac66b75b`). The WebKit renderer never had this, so it is not a Wails v3
regression. It was simply never ported.

While idle, the WebKit renderer still renders every tile about 7 times a
second:

- the `RefreshInterval` timer (200 ms) calls `TriggerRedraw`;
- the blink loop (`blink.go`, 500 ms) calls `TriggerRedraw`.

Each redraw runs `PopDrawCommands`, which renders all tiles, then encodes the
commands and emits them to JavaScript. JavaScript then draws them to the
offscreen canvas. This happens whether or not anyone can see the window.

## Decision

Skip render requests while the window cannot be seen.

- `webkitRender.renderPaused` (`atomic.Bool`) is the pause flag.
  - `redraw()` returns early while paused.
  - The timer case does not call `TriggerRedraw` while paused.
- `SetRenderingPaused(paused)` only acts when the state changes.
  - It logs `[trace] renderer: rendering paused|resumed` (ADR 0098).
  - On resume it calls `TriggerRedraw()`.

Pausing drops render *requests*, not batches that were already rendered.

- Draw commands and `fnSchedule` deallocations are only created inside
  `PopDrawCommands`, so nothing builds up while paused.
- PTY reading and `Term` state carry on as normal.
- The first redraw after resume therefore renders the current state in full.
- A batch that was already rendered is never thrown away. Throwing one away
  would be unsafe: the frontend's offscreen canvas builds up draws over time,
  and a batch only holds the tiles that rendered.

### Window events

`registerRenderPauseEvents` in `window/frontend/main/window.go` wires:

- **pause** to `Common.WindowHide` and `Common.WindowMinimise`;
- **resume** to `Common.WindowShow`, `WindowUnMinimise`, `WindowRestore` and
  `WindowFocus`.

On macOS, Wails emits `WindowShow`/`WindowHide` from
`windowDidChangeOcclusionState`. Minimising, Cmd+H, and a window fully
covered by other windows all count as hidden.

### Frontend safety net

`resumeTerminalAfterSuspend()` in `terminal.js` already runs on
`visibilitychange`, `focus` and `pageshow`. When `document.hidden` is false, it
now also emits `ttyphoon:terminal-visible`, and Go resumes rendering when it
receives it. This covers the case where a native show event is missed.

`TerminalRequestRedraw` deliberately does **not** resume rendering. The rAF
watchdog calls it, and if it unpaused, rendering could restart while the
window is still hidden.

### Rejected: polling `IsVisible()`

`WebviewWindow.IsVisible()` uses `InvokeSync` on the main thread. If the main
thread is stuck, polling it would block the render loop, which is the failure
mode described in ADR 0020.

### Backpressure

Only one terminal redraw batch may be in flight:

- The backend assigns each batch a monotonically increasing ID and emits the
  ID with the compact draw operations.
- The frontend applies all operations to the offscreen canvas, schedules the
  visible paint, then emits `ttyphoon:terminal-redraw-ack` with that ID.
- The backend does not call `PopDrawCommands` or emit another batch until it
  receives the matching acknowledgement. Redraw requests continue to coalesce
  in the existing one-slot channel.
- The acknowledgement only triggers a redraw if one was held back while
  waiting (`redrawDeferred`). Every full render produces commands, so
  triggering unconditionally made an unthrottled emit→ack→emit loop that
  pegged the CPU. The fix was applied on 2026-11-11; the user confirmed it
  significantly improved CPU use.
- Acknowledgement is sent in a `finally` block, including malformed or failed
  batches, so an exception cannot permanently stall rendering. Errors
  emitting the acknowledgement are logged.

Acknowledgement happens after synchronous offscreen-canvas updates, not after
the requestAnimationFrame paint. This bounds Wails event delivery and draw
command work without relying on animation frames that WebKit may suspend while
the window is occluded. The visible paint remains coalesced as before.

## Deferred

3. **Render only when something changed.** Skip timer and blink redraws when
   no tile is dirty and the cursor is not blinking. This would cut idle CPU
   while the window is visible too.

## Consequences

- Idle CPU while hidden drops significantly (reported 2026-11-11).
  What's left is mainly PTY reading and the blink ticker, which does almost
  nothing while paused.
- If neither the native resume event nor the frontend event ever arrived, the
  terminal would stay frozen. Focus, show, unminimise, restore and
  `visibilitychange` all resume rendering, so this needs every one of those
  paths to fail at once.
- Tests:
  - `window/backend/renderer_webkit/start_test.go` checks pause/resume behavior,
    that only one batch is emitted until its matching acknowledgement, and
    that stale acknowledgements do not release backpressure;
  - `frontend/src/terminal.test.js` checks that a batch is acknowledged after
    its draw operations are applied and checks the visibility event.
