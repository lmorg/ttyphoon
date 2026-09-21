# 0020. Sleep/wake hang hardening

**Status:** Accepted

## Context

The compiled app frequently — but not always — hangs when the laptop enters or
leaves sleep. Investigation found no single cause. Instead there are several
independent latent deadlocks and dropped-state bugs, and a suspend/resume cycle
is unusually good at triggering them because it produces exactly the conditions
each one needs: a truncated pty stream, a failed pty write, a large buffered
output burst, and a suspended WebKit content process.

Six theories were identified. Three are fixed by this ADR. The remaining three
are recorded here so they are not rediscovered.

### Fixed: tmux `SendCommand` leaked its mutex on write error

`SendCommand` acquired `tmux.limiter`, wrote to the tmux control pty, and
returned early on write error **without unlocking**. A pty write returning
`EIO`/`EAGAIN` across suspend/resume permanently locked the mutex.

Every keypress routes through this function (`PaneT.Write` -> `send-keys`), as
do resize, `RefreshWindowList`, `GetTermTiles`, pane zoom and window select. The
canvas kept painting but nothing responded to input, and goroutines piled up
behind the mutex.

The blocking receive on the response channel also had no timeout, so any lost
response wedged the caller forever while it still held the mutex.

### Fixed: the tmux reader goroutine could deadlock or die silently

Two failure modes in the `bufio.Scanner` goroutine started by `NewStartSession`:

1. **Unbuffered response channel.** `resp` was `make(chan *tmuxResponseT)` and
   `_respEnd` did a bare blocking send. If a `%end`/`%error` block arrived with
   no `SendCommand` waiting — which happens the moment the mutex leak above
   fires, or when tmux emits a block we did not initiate — the reader goroutine
   blocked forever. All `%output` processing stopped: the terminal was dead,
   permanently, with no error surfaced anywhere. Worse, the *next* `SendCommand`
   would receive that stale response, desyncing every subsequent
   request/response pair by one.

2. **`bufio.Scanner`'s 64 KB default token limit.** `%output` lines are
   octal-escaped, so one byte of pane output can become four characters. On wake
   tmux flushes everything it buffered while suspended, and a single line can
   easily exceed 64 KB. `Scan()` then returns `false` with `ErrTooLong`, the loop
   exits, and **the goroutine returns silently** — tmux I/O is gone for the life
   of the process. This best explains the "often, but not always" character of
   the hang: it depends on how much output accumulated during sleep.

### Fixed: `rafPending` latched true in the frontend

`terminal.js` guarded the `terminalRedraw` handler with a `rafPending` boolean
that was only cleared from inside a `requestAnimationFrame` callback. WebKit
stops servicing rAF when the window is occluded or the display sleeps, and a
callback scheduled immediately before content-process suspension can be dropped
rather than deferred. When that happened the latch stayed `true` forever and
*every* subsequent redraw event was discarded — Go kept emitting, the canvas
never updated again.

### Not fixed here: parser blocks on a partial escape sequence

`Term.readLoop` holds `term._mutex` across `readChar`, and `readChar` dispatches
into `parseCsiCodes`/`parseOscCodes`/`parseDcsCodes`/`parseApcCodes`, which call
`term.Pty.Read()` in a loop **while that lock is still held**. `runebuf.Buf.Read`
is an infinite 15 ms poll with no timeout and no error path other than `Close()`.

A stream cut mid-escape-sequence therefore parks the parser goroutine while it
holds `_mutex`. `Term.Render()` then blocks on the same mutex, and because
`Render()` is driven from the single redraw goroutine in
`renderer_webkit/start.go`, the whole UI freezes — every pane, not just the
affected one.

Fixing this requires a deadline-capable `Buf.Read` and resync-on-timeout in the
sub-parsers, or restructuring so the lock covers only mutations.

Related: `_respOutput` writes inline when the pane is known but from a spawned
goroutine when it is not, so output for a newly discovered pane can be reordered
and manufacture a mangled escape sequence on its own.

### Not fixed here: post-wake burst starves the renderer

`Term.Render()` skips rendering while `Pty.BufSize() > 0`, for up to 1000
consecutive calls (`_ssLargeBuf`). `runebuf` only truncates its rune slice once
fully drained, so `BufSize()` stays large for the whole duration of a sustained
burst. While tmux flushes its post-sleep backlog this yields roughly one frame
per 1000 redraw cycles — indistinguishable from a hang, though it self-resolves.

`runebuf.Buf.BufSize()` also reads `len(buf.bytes)` while holding `rm` rather
than `bm`, which is a genuine data race on that read.

### Not fixed here: main-thread dispatch flooding

`runtime.EventsEmit` on Wails v2 darwin becomes `evaluateJavaScript` via
`dispatch_async` on the main queue. The redraw goroutine emits unconditionally
every `RefreshInterval` with no awareness of whether the webview is draining, so
blocks accumulate during display sleep and land as a flood on wake.

Relatedly, Wails' cgo `processMessage` export writes into a 100-slot buffered
channel and is called *on the macOS main thread*. If that channel fills — which
it will if any bound method backs up behind a Go-side lock — the send blocks the
main thread and the entire application freezes at the AppKit level. This is the
mechanism by which any of the Go-side locks above escalates from a stale canvas
to a full app hang.

Note also that `config/defaults.yaml` documents `RefreshInterval: 0` as
"disables a timer-based refresh entirely", but `time.After(0)` fires immediately,
so that value would spin the redraw goroutine at 100% CPU. The documented
behaviour is not implemented.

## Decision

1. **`SendCommand` never leaks its mutex and never waits forever.** Unlock via
   `defer`. Drain any orphaned response before writing, so a previously timed
   out command cannot desync the next one. Wait on the response with a
   `tmuxCommandTimeout` bound. Fail fast when the reader is already dead.

2. **The tmux reader goroutine can never park and never dies silently.** The
   response channel is buffered (capacity 1) and `_respEnd` sends
   non-blockingly, dropping unsolicited blocks rather than parking. The scanner
   gets a `tmuxMaxLineLength` (16 MB) maximum token size. When the loop exits,
   `scanner.Err()` is logged and surfaced as a notification, a `readerDead` flag
   is set, and a synthetic error response is pushed to unblock any waiter. The
   startup handshake is bounded by the same timeout.

3. **The frontend redraw latch always clears.** A watchdog `setTimeout` clears
   `rafPending` and force-paints if the rAF callback has not fired within one
   second, and `visibilitychange`/`focus`/`pageshow` reset the latch and request
   a fresh redraw on resume.

## Consequences

- Any of these three faults now degrades to a visible error notification and a
  recoverable state instead of a silent permanent hang.
- `SendCommand` can now return a timeout error where it previously blocked.
  Callers already surface errors as notifications, so no call-site changes were
  needed, but a slow tmux server now produces user-visible errors rather than a
  stall.
- Dropping unsolicited response blocks means a genuinely mismatched
  request/response pair is discarded rather than mis-attributed. The drain in
  `SendCommand` keeps the channel consistent across timeouts.
- The frontend watchdog paints outside a frame on the recovery path. That is
  acceptable for an error path and only runs when rAF has already failed.
- Theories 1, 5 and 6 remain open. A hang whose goroutine dump shows threads
  parked on `virtualterm.Term._mutex` is theory 1, not one of these fixes.

## Reference

- `tmux/command.go` — `SendCommand`
- `tmux/tmux.go` — `NewStartSession` reader goroutine, `_respEnd`, `Tmux.resp`,
  `tmuxCommandTimeout`, `tmuxMaxLineLength`, `Tmux.readerDead`
- `frontend/src/terminal.js` — `clearRedrawLatch`, `terminalRedraw` handler,
  `resumeTerminalAfterSuspend`
- `virtualterm/ansi_c0.go`, `virtualterm/ansi_csi.go`,
  `utils/rune_buf/rune_buf.go` — open theory 1
- `virtualterm/render.go`, `window/backend/renderer_webkit/start.go` — open
  theories 5 and 6
- Wails v2 darwin `internal/frontend/desktop/darwin/frontend.go` —
  `messageBuffer` (100 slots, written from the main thread)
- Diagnosis aid: trigger a goroutine dump while hung (a `SIGUSR1` handler
  calling `pprof.Lookup("goroutine").WriteTo(f, 2)`, or the existing
  `debug/pprof` package) to identify which theory fired.
