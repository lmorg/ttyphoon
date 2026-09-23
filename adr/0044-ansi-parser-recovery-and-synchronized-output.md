# 0044 - ANSI parser recovery and synchronized output

## Status

Accepted. Synchronized output, OSC 22 pointer cursors, ordered tmux pane output,
and parser recovery are implemented by this decision.

## Context

Terminal logs from modern interactive applications exposed four distinct ANSI
support gaps:

1. Repeated `CSI ? 2026 h` and `CSI ? 2026 l` sequences were reported as
   unknown DEC private modes. Mode 2026 is synchronized output: set begins an
   atomic update and reset ends it. Applications use it to prevent users seeing
   partially drawn frames.
2. `OSC 22` requests with `default`, `pointer`, and `text` payloads were reported
   as unknown. OSC 22 selects the pointer cursor shape. The warning omitted the
   final payload character because `parseOscCodes` removes the terminator and
   then the unknown-code logger slices one additional rune.
3. Warnings containing partial sequences such as
   `38;2;198;208;245 ESC [` are interrupted CSI sequences, not missing
   true-colour support. Valid semicolon-separated true-colour SGR is already
   supported and tested. `parseCsiCodes` currently treats an embedded `ESC` as
   an unknown parameter, and `parseCsiExtendedCodes` consumes through the `[` of
   the replacement sequence.
4. `Unexpected rune after escape: 27` is the same recovery gap at the C1 entry
   point: a second `ESC` should cancel/restart escape parsing rather than become
   a warning.
5. `CSI > 25 u` is a kitty keyboard-protocol push request, while
   `CSI ? 2026 $ p` and `CSI ? 2048 $ p` are DEC private-mode queries. These
   capability negotiations were syntactically valid but logged as unsupported.

The interrupted stream may be valid cancellation or upstream reordering.
`tmux._respOutput` writes known-pane output inline but initializes an unknown
pane in a goroutine, allowing subsequent chunks to overtake its first chunk and
manufacture split ANSI sequences. Parser recovery is required regardless, but
recovery cannot restore styling bytes that arrived out of order.

## Decision

### Synchronized output

Implement DEC private mode 2026 per terminal, not on the shared renderer:

- `CSI ? 2026 h` starts or refreshes a synchronized-update deadline.
- While active, `Term.Render` returns without emitting any draw commands.
- `CSI ? 2026 l` ends the update and explicitly requests one redraw.
- An unmatched reset is ignored.
- A repeated set refreshes the deadline.
- A one-second timeout ends synchronization so a crashed application cannot
  leave a pane permanently frozen.
- The gate is checked while holding the terminal mutex and before `DrawFrame`,
  preventing a redraw already queued by the sequence introducer from painting
  an intermediate frame.

Terminal state continues to ingest and mutate during synchronization; only
presentation is deferred. Other panes remain renderable because synchronization
is owned by each `Term`.

### Pointer cursor shape

Implement OSC 22 through the existing backend cursor and frontend `setCursor`
event path:

- Accept `default`, `pointer`, and `text`, matching the CSS cursor names emitted
   by observed applications. Empty and unsupported values fall back to
   `default`, matching xterm's fallback semantics without exposing arbitrary CSS.
- Store the validated base cursor on each `Term`.
- Apply an OSC update immediately only when its terminal is focused. Background
   terminals retain their requested cursor and apply it when focused.
- Treat hyperlink, image, table, and other element cursors as temporary global
   overrides. Returning to the arrow restores the active terminal's base cursor
   rather than always forcing `default`.
- Activating a pane clears any temporary override inherited from the previously
   active pane.
- Synchronize cursor backend state because parser output and UI hover events
   arrive on different goroutines.

The OSC parser's unknown-code logger now reports the complete parsed payload;
the terminator had already been removed, so its second slice incorrectly
removed the final content character.

### Unknown-pane tmux output ordering

Preserve `%output` byte order while a newly reported pane is initialized:

- Do not initialize synchronously on the tmux scanner goroutine. Pane discovery
   sends a tmux command and waits for its response, which only that scanner can
   process; waiting inline would deadlock or time out.
- Keep a per-pane output backlog on `Tmux`, protected by one mutex.
- The first chunk for an unknown pane creates the backlog and launches exactly
   one initializer goroutine. Later chunks append to that backlog in scanner
   order rather than launching more goroutines.
- The initializer obtains the pane, flushes every queued chunk in order, and
   removes the pending marker while holding the same mutex. A later scanner
   chunk therefore cannot take the direct-write path until the backlog is fully
   written.
- If initialization fails, discard that attempt's backlog, remove its pending
   marker so a future chunk can retry, and emit one error notification.

This state belongs to `Tmux`, not a placeholder `PaneT`: publishing an
incomplete pane would cause `updatePaneInfo` to treat it as initialized and skip
construction of its terminal and rune buffer.

### CSI cancellation and escape restart

Recover from interrupted standard and private CSI sequences without leaking
control bytes into terminal text:

- `ESC` inside CSI cancels the current sequence and returns a restart result to
   the C1 parser.
- The C1 parser handles restart iteratively. A replacement sequence begins at
   the new `ESC`; repeated `ESC` bytes continue restarting rather than recursing
   or producing `Unexpected rune after escape: 27`.
- CAN (`0x18`) and SUB (`0x1a`) cancel standard or extended CSI and return to
   ground state. The following byte is parsed normally.
- Extended/private CSI parsing reports restart and cancellation as distinct
   internal conditions, preventing a cancelled parameter buffer from reaching a
   handler that expects a final byte.
- Stream-level tests verify replacement SGR state, private-sequence
   cancellation, repeated escape handling, and that no interrupted parameter
   bytes render as text.

### Capability negotiation

Respond truthfully to private-mode queries and recognize unsupported keyboard
negotiation without claiming functionality:

- Handle DEC Request Mode (`DECRQM`, `CSI ? Ps $ p`) with the required
   `CSI ? Ps ; Pm $ y` response.
- Report mode 2026 as set (`Pm = 1`) while synchronized output is active and
   reset (`Pm = 2`) otherwise. Querying also expires a stale synchronization
   deadline before reporting its state.
- Report unsupported modes, including mode 2048 in-band resize notifications,
   as unrecognized (`Pm = 0`). This lets applications disable the extension
   instead of waiting for or assuming unavailable resize events.
- Recognize kitty keyboard query, set, push, and pop controls ending in `u`.
   Ttyphoon sends no query response and ignores state changes, keeping legacy
   keyboard encoding active. Advertising flag 25 would be incorrect because the
   current key path cannot report every key as an escape sequence with associated
   text, alternate keys, or release events.
- Retain diagnostics for unrelated secondary and tertiary CSI controls.

## Consequences

- Applications using mode 2026 no longer produce warning floods or visibly torn
  frames.
- A missing end marker delays a pane by at most one second rather than freezing
  it indefinitely.
- Shared renderer activity and unrelated panes are not blocked.
- OSC 22 requests no longer produce warning floods, background panes cannot
   change the visible pointer, and hover feedback restores the requested cursor.
- Initial output from newly discovered panes cannot overtake earlier chunks and
   manufacture malformed ANSI sequences.
- Interrupted CSI and repeated escape sequences resynchronize at the replacement
   escape rather than producing warnings or consuming the next sequence.
- CAN and SUB safely abandon incomplete CSI input while preserving subsequent
   printable text.
- Applications can query synchronized-output state and receive an exact DECRQM
   response; unsupported mode 2048 is explicitly reported as unrecognized.
- Kitty keyboard negotiation no longer produces log noise, but ttyphoon remains
   intentionally undiscoverable as a kitty keyboard implementation until its
   key encoder can honor the protocol.

## Reference

- `virtualterm/ansi_csi_private.go` - DECSET/DECRST dispatch
- `virtualterm/ansi_csi.go` - CSI parser and interrupted-sequence behavior
- `virtualterm/ansi_c1.go` - escape-sequence entry point
- `virtualterm/ansi_recovery_test.go` - restart and cancellation coverage
- `virtualterm/ansi_capability_test.go` - DECRQM and kitty negotiation coverage
- `virtualterm/ansi_csi_secondary.go` - kitty push/pop recognition
- `virtualterm/ansi_osc.go` - OSC dispatch and logging truncation
- `virtualterm/osc_pointer_cursor.go` - OSC 22 validation and pane ownership
- `virtualterm/render.go` - per-terminal presentation gate
- `window/backend/cursor/cursor.go` - base and temporary cursor layers
- `frontend/src/terminal.js` - `setCursor` CSS event sink
- `tmux/tmux.go` - unknown-pane output backlog and initialization ordering
- `tmux/output_order_test.go` - split ANSI sequence ordering coverage
- ADR 0020 - parser blocking and tmux output-reordering risks
