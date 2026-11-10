# 0098 - Backend logging conventions

Date: 2026-11-10

## Status

Accepted. Documents existing practice; new code must follow it.

## Context

Backend diagnostics need to reach the developer without extra plumbing. The
desktop app replaces the standard logger's output with `logWriter`
(`window/frontend/main/lifecycle.go`, `window/frontend/main/logging.go`), which:

- writes every line to stderr, and
- emits it as a `notesLog` event to the Notes Tools log panel (ADR 0060).

The panel takes the level from a leading token after the timestamp. It
colours and filters `error`, `warn` and `info` (visible by default) and
`debug` and `trace` (hidden by default). Before emitting, `logWriter` prefixes
`[debug] ` to any line that does not start with a lowercase `[level] ` token, and
it collapses newlines to ` | `.

A survey of the Go code at the time of writing found:

- Logging uses only the standard library `log` package, with no `slog` or
  third-party logger. No production code calls `log.Fatal*` or `log.Panic*`.
- About 200 `log.Print*` calls carry a level prefix (`[debug]` ~80,
  `[error]` ~60, `[warn]` ~50, `[trace]` ~10, `[info]` ~10).
- About 110 legacy calls have no prefix (for example `log.Println(err)` or
  `"ERROR: ..."`). These all appear as debug, so real errors from them are
  hidden by default.
- A separate `debug` package (`debug.Log`, `debug.Enabled`, `debug.Trace`) is
  compiled in only with the `debug` / `trace` build tags. `debug.Log` emits
  `[trace]` lines that include the caller.
- User-facing notifications (`renderer.DisplayNotification` /
  `DisplaySticky` with `types.NOTIFY_*`) are not written to the log.

## Decision

1. **Use the standard `log` package.** Call `log.Printf` / `log.Println` /
   `log.Print`. Do not introduce another logging library, do not change
   `log` flags or output outside `lifecycle.go`, and do not call `log.Fatal*` or
   `log.Panic*`. A desktop app must report errors, not exit.
2. **Always start the message with a level prefix**, which must be exactly one
   of `[error] `, `[warn] `, `[info] `, `[debug] ` or `[trace] `. Use lowercase
   and keep the single trailing space.
   - `error`: an operation failed and data or functionality was lost.
   - `warn`: unexpected but recovered from, such as an unhandled escape
     sequence or a restarted subprocess.
   - `info`: an infrequent lifecycle event worth seeing by default, such as
     starting the LSP or shutting down a server.
   - `debug`: diagnostics for attributing a bug, such as pane creation, races
     avoided, or queued output retained or flushed.
   - `trace`: very high-volume or per-byte detail, such as subprocess
     stderr, CSI lookups or readline updates.
3. **Follow the level with an area tag**, in the form
   `[level] <area>: message`. Use the subsystem name, for example `tmux:`,
   `term:`, `lsp[<cmd>]`, `MCP OAuth:` or `grep:`. This keeps the panel
   searchable.
4. **Write one line per event and quote values.** Use `%q` or `"%s"` for
   values that may be empty or contain spaces, and keep IDs in their native
   form (`%42`, `@1`). Multi-line text is flattened to ` | ` in the panel, so
   avoid relying on layout.
5. **Never log secrets.** Redact tokens, credentials, OAuth codes and
   sensitive paths, as `utils/jupyter/write.go` does with `******`.
6. **Logging does not replace user notifications.** If the user needs to act,
   call `renderer.DisplayNotification` (or return the error to the tool or
   caller) **and** log it. A notification alone leaves no record in the log.
7. **Hot paths:** use `[trace]` for anything that may fire per output chunk or
   per keystroke. If even formatting the message is too costly, guard it with
   `if debug.Trace { ... }`. Use `debug.Log` only for temporary developer
   instrumentation that should vanish from release builds.
8. **Legacy unprefixed calls** may stay until the surrounding code is touched.
   When editing such code, add the correct prefix; most `log.Println(err)`
   calls should become `[error]` or `[warn]`.

## Consequences

- The Notes log panel's level filters behave predictably, and errors are no
  longer hidden as debug.
- New log levels need changes to both `logWriter` (which only recognises
  lowercase letters) and the panel filters (ADR 0060).
- `logWriter` assumes the default `log` timestamp prefix (20 characters).
  Changing `log` flags would break level detection.

## Reference

- `window/frontend/main/logging.go` - `logWriter`, default `[debug]` prefixing
- `window/frontend/main/lifecycle.go` - `log.SetOutput`
- `frontend/src/notes-log-panel.js` - level parsing, colours and filters
- `debug/debug.go`, `debug/debug_enabled.go`, `debug/trace_enabled.go` -
  build-tag-gated developer logging
- `types/notification.go` - notification levels (separate from logging)
- Examples: `utils/lsp/process.go`, `virtualterm/ansi_c1.go`,
  `utils/jupyter/write.go`, `tmux/pane.go` (ADR 0097 diagnostics)
- ADR 0060 - Notes log filters
