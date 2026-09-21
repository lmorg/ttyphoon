# 5. AI panel emits are gated on what is actually on screen

Date: 2026-09-06

## Status

Accepted

## Context

Agents can run concurrently for different workspaces (tmux tabs), and the panel's
prompt dropdown lets the user view a historical prompt while a run is still in
progress.

`WriteToSessionLog` already gated emits on `ctx.WorkspaceActive`, which handled
the workspace axis. Nothing gated the *prompt* axis: after selecting an older
prompt, `aiStreamOrder.runId` still matched the live run, so a running agent kept
appending live chunks on top of the historical view the user was reading.

## Decision

Separate **persistence** from **presentation**.

- Markdown is always written. The pending-file append happens under the
  `aiSessionLogStore` lock *before* any emit decision, so the on-disk log is
  complete regardless of what the panel is showing.
- Emits are dropped unless the panel is currently displaying that workspace's
  live output.

`sessiondb` holds the view state per workspace:

```go
var panelView = struct {
    sync.Mutex
  byWorkspace map[string]bool
}{byWorkspace: map[string]bool{}}
```

A single predicate, `ctx.emitContent()`, guards all four emit sites
(`aiJobStart`, `aiResponseStream` ×2, `aiJobFinish`). The frontend reports state
changes through the `SetAIPanelLive` binding:

- `false` when jumping to a historical prompt
- `true` when starting any new request, or selecting the **Live output** entry in
  the prompt dropdown

Because dropped chunks are not buffered, returning to live first asks
`GetActiveStreamSnapshot` for the in-progress run's accumulated text, run ID and
next sequence number. The frontend resets its ordered cursor from that snapshot
and renders the accumulated text. If no run is active, it falls back to the
finalized on-disk session log.

## Consequences

- Concurrent workspace runs cannot suppress or corrupt each other's panel output;
  panel live/history state is keyed by workspace.
- Reading history is stable while an agent is running.
- Dropped emits are **not** buffered for later delivery — active-run recovery is
  by snapshot, and completed-run recovery is from disk. This avoids unbounded
  memory growth for long runs.
- The prompt dropdown must remain enabled while viewing history, otherwise the
  user can become stranded with no way back to live.
- An unreported workspace defaults to live, so a frontend that never calls the
  binding behaves exactly as before.

## Reference

- `ai/agent/sessiondb/session_log.go` — `panelView`, `SetPanelView`,
  `GetActiveStreamSnapshot`, `emitContent`
- `frontend.go` — `SetAIPanelLive`, `GetAIActiveStreamSnapshot`
- `frontend/src/notes.js` — `setAIPanelLive`, `applyActiveStreamSnapshot`,
  `loadAISessionCache`, `jumpToAIPromptTarget`
