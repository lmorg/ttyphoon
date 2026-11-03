# 66. End key names must reach terminal encoding

Date: 2026-10-14

## Status

Accepted.

## Context

Pressing End did not move the shell cursor, including with tmux enabled.
The ANSI lookup already encoded `AnsiEnd` correctly, but `KeyNameToCode`
did not recognize the browser's `End` key name. The WebKit keyboard handler
therefore received code zero and returned without writing anything.
Existing sequence tests started with `AnsiEnd`, bypassing the failed lookup.

## Decision

- Map `End` to `AnsiEnd` in the shared key-name lookup, preserving modifiers.
- Keep the existing terminal-mode encoding: normal mode uses `ESC [ F`,
  application mode uses `ESC O F`, and VT220 uses `ESC [ 4 ~`.
- In tmux control mode, send a named key, not a hard-coded ANSI sequence.
  The internal NUL-prefixed `End` token is sanitized by `PaneT.Write` into
  `send-keys -t <pane> End`, without `-l`. Tmux selects the sequence delivered
  to the pane based on its terminal state.
- Test the full key-name-to-sequence path, including tmux token sanitization,
  rather than testing only the already-resolved numeric key code.

## Consequences

End now reaches both direct PTYs and tmux panes. No shell-specific Ctrl+E
substitution or change to tmux's keyboard-mode handling is required. Regression
tests cover all four encoding modes and modified End key names.

## Reference

- `codes/keynames.go`, `codes/ansi_lookup.go`, `codes/modifiers_test.go`
- `window/backend/renderer_webkit/keyboard_handlers.go`
- `window/backend/renderer_webkit/keyboard_mode.go`
- `tmux/pane_pty.go`