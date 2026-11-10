# 0097 - Cloned panes that hang blank: diagnosis and proposed fixes

Date: 2026-11-10

## Status

Proposed. Fixes 1, 2 and 5 are implemented; fix 3 is a low-priority
nice-to-have; fix 4 remains proposed.

## Context

Panes created through `ClonePane` (ADR 0072) are sometimes created but then
remain blank: no output is rendered and the pane appears frozen. It happens most
often when the AI `bash` tool (ADR 0056) runs a command, and rarely for the
commandless Clone control.

The difference matters. A commandless clone starts an interactive shell, which
takes time before producing output. A command clone launches
`bash <wrapper script>`, and the wrapper's first line emits an OSC 7 sequence
immediately (`tmux/bash.sh.tmpl`). The first `%output` for the new pane
therefore arrives at almost the same moment as the `split-window` response and
its `%layout-change`.

Code review of `tmux/` identified the following candidate causes, in order of
likelihood.

### 1. Racing pane creation produces two `PaneT` objects for one tmux pane

Three goroutines can independently discover the new pane ID:

- `clonePane` calls `updatePaneInfo(newPaneID)` after `split-window`.
- `_respOutput` sees an unknown pane, queues its output, and starts a goroutine
  that calls `updatePaneInfo(paneId)`.
- `%layout-change` → `RefreshWindowList` → `GetTermTiles` → `updatePaneInfo("")`.

All three reach `paneInfo.updatePane`, which does a check-then-act:
`tmux.panes.Get(info.Id)` and, if nil, `tmux.newPane(info)`. `paneMap` locks
each `Get`/`Set` individually, so the sequence is not atomic. The window between
check and `panes.Set` is wide: `newPane` constructs a `virtualterm.Term`, runs
`agent.New` (which initialises every AI tool), and starts the terminal's read
goroutine before publishing the pane.

When two callers lose the race together, each builds its own `PaneT`, with its
own `runebuf` and `Term`:

- `tmux.panes` and `win.panes` can end up holding different objects. Output is
  routed via `tmux.panes`; the renderer draws `win.panes`. The visible pane
  stays blank. Input still reaches tmux (`send-keys` is addressed by pane ID)
  but its echo lands in the unseen object, so the pane looks frozen.
- The losing object's read loop and agent leak, and `allTheAgents` is
  overwritten for that tile ID.
- `pendingPaneState` is consumed by the first `newPane`, so the other object
  misses its seeded environment and working directory.

This fits the AI-heavy symptom profile best.

### 2. Early output is discarded when pane initialisation fails

In `finishPendingPaneOutput`, if `updatePaneInfo` returns an error (for example
a 5s `tmuxCommandTimeout` on `list-panes`) or the pane is not found, the queued
backlog is deleted rather than replayed. A short command can emit all of its
output, including the completion APC, in that first burst. The pane stays
blank, the interactive-action callback never fires, and `Bash.run` waits
indefinitely because it only stops on context cancellation.

### 3. Control-mode responses are paired by order, not by command number

`SendCommand` assumes the next `%end`/`%error` belongs to the command it just
wrote, using a one-slot channel. It ignores the command number tmux prints on
`%begin`/`%end`. If a command times out and its reply arrives after the next
command has been written, that next command consumes the stale reply and every
later command is shifted by one. Unsolicited blocks have the same effect. Under
AI load (frequent `RefreshWindowList`, `list-panes`, and `resize-pane` calls)
`split-window` or `list-panes` could receive the wrong reply and leave a
half-initialised pane. Whether tmux hooks generate unsolicited blocks has not
been verified.

### 4. Commands blocked on stdin (not a ttyphoon defect)

Since ADR 0056, AI commands run with a real TTY. Commands that read stdin
without prompting (`grep pattern` with no file, `cat`, `read`, confirmation
prompts) sit silently in the cloned pane. This should be ruled out when
reproducing.

### Distinguishing the causes

- Tool returns a result but the pane is blank: cause 1. The completion APC was
  parsed by the hidden duplicate `Term`.
- Tool also hangs: look for causes 2 or 3, e.g. a timeout or
  "pane not found" notification.
- Debug logs showing two `newPane` calls for the same `%id` confirm cause 1.

## Decision

Proposed fixes, in recommended order:

1. **Implemented: make pane get-or-create atomic.** Serialise concurrent
   discovery while a pane is being constructed, re-check `tmux.panes.Get`
   under the creation lock, and publish the fully initialised pane to both pane
   maps only once. Protect subsequent updates to the winning pane from racing
   metadata updates. Startup directory/environment seeds arriving during
   construction are applied before the pane is published. A concurrent
   discovery regression test verifies one `PaneT`, one `Term`, and preserved
   startup state.
2. **Implemented: retain pane output when initialization fails.** Keep buffered
   chunks and surface the failure. A later output notification may retry pane
   initialization; any successful `updatePaneInfo` discovery flushes the
   backlog into the pane buffer before returning. Remove the pending entry only
   after delivery.
3. **Low priority (nice-to-have): bound the AI `bash` wait.** Add a configurable
   upper bound or a liveness check (pane exists, not dead) to `Bash.run`, so a
   lost completion becomes a model-visible error rather than an indefinite
   hang. Deprioritised because the user can already close a stuck pane
   manually, which cancels the pending interactive action and returns control
   to the tool.
4. **Proposed: correlate control-mode replies by command number.** Track the command
   number from `%begin` and discard `%end`/`%error` blocks that do not belong
   to the outstanding command, replacing the drain-one-stale-response
   heuristic.
5. **Implemented: improve diagnostics.** Using the standard `log` package with
   the existing `[debug] ` category prefix (which the Notes log panel filters
   as debug level, ADR 0060), log:
   - `tmux: cloned pane <new> from source pane <src> (action=...)`
   - `tmux: creating pane <id> in window <win> (caller=...)`, where the caller
     is the first stack frame outside `updatePaneInfo`/`updatePane`
   - `tmux: duplicate creation of pane <id> avoided; reusing existing pane`
   - `tmux: pane <id> initialization failed; retaining <n> queued output chunks`
   - `tmux: flushed <n> queued output chunks to pane <id>`

Cause 4 needs no code change; at most, the `bash` tool description could warn
the model to avoid commands that wait on stdin.

## Consequences

- Fix 1 removes the dominant identified failure mode and prevents duplicate
  terminals and agents.
- Fix 2 preserves early output across transient initialization failures and
  flushes it when pane discovery later succeeds. Fix 3, if implemented, would
  turn a remaining lost completion into a visible error.
- Fix 5 lets future occurrences be attributed from the debug log: a blank pane
  with a "duplicate creation avoided" line points at discovery races, a
  "retaining" line without a matching "flushed" line points at failed setup.
- Fix 4 touches the core control-channel path and needs careful testing,
  including the sleep/wake behaviour described in ADR 0020.
- Until these land, a blank cloned pane can be recovered by closing it; a hung
  AI tool call can be stopped by cancelling the prompt.

## Reference

- `tmux/pane.go` - `clonePane`, `updatePaneInfo`, `updatePane`, `newPane`
- `tmux/pane_map.go` - per-operation locking only
- `tmux/tmux.go` - `_respOutput`, `queuePaneOutput`, `finishPendingPaneOutput`,
  `_respLayoutChange`, `_respEnd`
- `tmux/command.go` - `SendCommand` response pairing
- `tmux/window_tab.go` - `GetTermTiles` calls `updatePaneInfo("")`
- `tmux/interactive_action.go` - completion callback registry
- `tmux/bash.sh.tmpl` - immediate OSC 7 output from command clones
- `virtualterm/term.go` - `NewTerminal` runs `agent.New` before publication
- `ai/tools/shell/bash.go` - unbounded wait for completion
- ADR 0020 - sleep/wake hang hardening
- ADR 0056 - Bash tool uses cloned panes
- ADR 0072 - interactive pane actions and cloned pane lifecycle
