# 0038 - Typed stream blocks must honour workspace and prompt scope

## Status

Accepted. Fixes A, B, C and D are implemented.

Amends 0005 and 0037. The compatibility layer D preserves is scheduled for
removal by 0039.

## Context

0037 replaced the linear markdown stream with addressable blocks and a new
`aiStreamBlock` transport. Four defects were reported against the live panel
after that migration. All four trace back to the same two oversights: the new
transport bypassed the emit gating 0005 established, and the live-restore path
still flattens blocks back into a single markdown string.

### 1. Old-style output reappears after a workspace switch

Switching workspace calls `markAISessionCachePending()` →
`loadAISessionCache()` → `GetAISessionCache()` → `GetSessionLog()`.

`GetSessionLog` picks the newest prompt from `listPromptLogMetasMerged()`, which
deliberately merges legacy markdown prompts with sqlite prompts (0037, step 10).
When that newest prompt is a **pre-migration** prompt it has no `stream_blocks`
rows, so `GetStreamPromptContent` fails and the code falls back to
`GetPromptLog()` — the legacy markdown file, still containing the retired
framing (`**Tool call:**`, `~~~~` fences, `> **Thinking:**`).

It is then rendered with `setAIFinalOutput()` → `setText()`, i.e. one flat
markdown document. That is the "old style output": genuinely old content, read
from the legacy fallback and rendered through the legacy one-shot path.

### 2. The live stream is not workspace-scoped

`einoRuntime.newStreamBlockWriter()` emits every block directly:

```go
return newAIStreamBlockWriter(r.agent.Workspace(), func(block sessiondb.AIStreamBlock) {
    runtime.EventsEmit(r.agent.Renderer().GetWindowContext(), "aiStreamBlock", block)
})
```

There is **no gating at all** on this path:

- no `agt.IsWorkspaceActive()` check
- no `panelShowsLive(workspace)` check
- the payload carries no workspace identifier, so the frontend cannot filter either

Meanwhile the legacy `aiResponseStream` path *is* gated, via
`SessionLogContext.emitContent()`. The two transports therefore disagree: an
agent running in workspace A keeps painting typed blocks into workspace B's
panel, while its legacy chunks are correctly suppressed.

The frontend compounds this. `markAISessionCachePending()` clears the panel but
leaves `aiActiveRunId` untouched, so blocks from the other workspace's run still
match the active run id and render into the freshly-cleared panel.

### 3. Viewing a historical prompt still receives live output

Same root cause. Selecting a prompt calls `setAIPanelLive(false)`, which sets
`panelView` in the backend — but `panelView` is only consulted by
`emitContent()`. `aiStreamBlock` never asks, so live blocks continue to append
underneath the historical prompt the user is reading.

### 4. Returning to Live output shows an unformatted lump

`resumeLiveAIOutput()` → `loadAISessionCache()` → `GetSessionLog()` →
`GetStreamPromptContent()`, which concatenates raw block content:

```go
content.WriteString(blockContent)
```

0037 step 5 deliberately moved **all** framing out of stored content and into
presentation: fences, blockquote prefixes and headings are applied by the
renderer from `kind`/`label`. Concatenating the raw content therefore discards
every block boundary, every fence and every quote marker, producing exactly the
undifferentiated text the user sees. `setText()` then renders that as one
markdown document.

The history path does **not** have this defect, because it uses `setBlocks()`.
Only the live/restore path flattens.

## Decision

Treat the typed transport as a first-class citizen of the 0005 gating model, and
stop flattening blocks anywhere in the UI.

### A. Gate typed block emits exactly like content emits

Implemented. Blocks are persisted to sqlite *before* they are emitted, so
suppression costs nothing — it remains the 0005 invariant that persistence and
presentation are separate. `newStreamBlockWriter` now emits only when both axes
agree:

```go
if !r.agent.IsWorkspaceActive() || !sessiondb.PanelShowsLive(workspace) {
    return
}
```

`panelShowsLive` was exported as `PanelShowsLive`, because the typed transport
gates on it directly rather than through a `SessionLogContext`. This fixes
issues 2 and 3 at the source.

### B. Carry the workspace in the payload, and filter on it

Implemented. `AIStreamBlock` gained a `Workspace` field, populated from the
writer. The frontend drops any block whose workspace disagrees with the panel's,
and forgets `aiActiveRunId` / `aiBlockStreamRunId` both when leaving a workspace
and when opening a historical prompt.

The filter is deliberately permissive when either side is empty, so a payload
from an older backend is not silently discarded. Gating A is authoritative; B
exists because a block can still be emitted in the window between the user
switching and the `SetAIPanelLive` binding round-trip completing.

### C. Restore live output as blocks, never as flattened markdown

Implemented. `ListLiveStreamBlockMeta(workspace)` resolves the blocks the panel
should treat as live: the in-flight run when `GetActiveStreamIdentity` reports
one open, otherwise the newest finalized prompt. It is exposed as the
`ListAILiveStreamBlocks` binding, and `loadAISessionCache()` now calls
`setBlocks()` — the same path history already used — falling back to the flat
markdown render only when the workspace has no blocks at all.

Three supporting changes were required:

- `GetStreamBlockContent` now accepts `promptId = 0`, because an in-flight run's
  rows are not assigned their prompt id until finalize.
- `setBlocks()` hydrates any block whose status is `open` eagerly rather than on
  scroll. Only an open block can receive live deltas, so it must not sit as an
  empty shell waiting to be scrolled into view.
- `appendBlock()` accepts a content snapshot from *any* event, not just a
  `closed` one. Every emitted block event already carries the full accumulated
  content alongside its delta, so the longer-or-equal snapshot heals a delta
  dropped while the panel was suppressed. This is what makes "switch away, switch
  back, catch up" correct without buffering.

The frontend also adopts the restored run id, so a still-running agent streams
into the rebuilt view instead of being discarded as a stale run.

`GetStreamPromptContent`'s flat concatenation is retained **only** for callers
that genuinely want a plain-text transcript (e.g. export); it must not back the
panel. If no caller needs it, delete it.

### D. Legacy markdown never backs the live view

Implemented as presentation only. After C, the live view reaches the flat
markdown path solely when a workspace has no blocks at all, so whatever the
session log returns there genuinely predates block storage. Rather than showing
nothing, it is rendered and flagged: `setText(text, { legacy: true })` toggles a
`notes-ai-legacy` class on the job root, which CSS renders with a rule and a
"Legacy transcript" label.

The flag is a class rather than a child node because `renderTextAsMarkdown`
keys its reset off `childElementCount`; an extra element would force a full
rebuild on every pass.

**Legacy files are not imported into `stream_blocks`.** Importing would
duplicate every legacy prompt (the file is retained *and* rows are written), and
would require parsing the retired framing back out of the markdown: the exact
inverse of the transform 0037 removed, and lossy.

Done this way D *reduces* footprint. The legacy path previously held two full
copies of the prompt — `state.aiSessionCache` and the formatter's `streamText`.
`state.aiSessionCache` was write-only (nothing read its contents) and has been
deleted, leaving one copy.

Note the block path is not unconditionally cheaper: `entry.text` is retained per
hydrated block with no eviction, so a fully-scrolled prompt converges on roughly
one full copy. Unhydrated blocks cost only metadata.

## Consequences

- Concurrent agents in different workspaces can no longer paint into each
  other's panel, on either transport.
- Reading history is stable again while an agent is running.
- Returning to live re-queries and rebuilds, so suppressed blocks are recovered
  rather than lost — the "catch up and resume" behaviour the user expects.
- Suppressed emits are still **not** buffered; recovery is by re-query, which is
  the same trade-off 0005 made and which 0037's addressable blocks make cheap.
- **Lesson: a new transport must inherit the old transport's gating.** 0037
  added `aiStreamBlock` alongside `aiResponseStream` but only the latter passed
  through `emitContent()`. Two transports with different gating is the same
  class of defect as two writers sharing one sink (0006).
- **Lesson: if content is stored without framing, it can never be concatenated.**
  Once presentation moved into the renderer, every read path had to become
  block-aware. One that was missed produced issue 4.
- **Lesson: a write-only cache is invisible waste.** `state.aiSessionCache` held
  a full copy of the largest prompt opened, indefinitely, and nothing ever read
  it. It survived because every reference was an assignment.

## Reference

- `ai/agent/runtime_eino.go` — `newStreamBlockWriter`
- `ai/agent/sessiondb/session_log.go` — `panelShowsLive`, `emitContent`,
  `GetSessionLog`
- `ai/agent/sessiondb/stream_blocks.go` — `GetStreamPromptContent`,
  `ListStreamBlockMeta`
- `frontend/src/notes.js` — `aiStreamBlock` handler, `markAISessionCachePending`,
  `loadAISessionCache`, `resumeLiveAIOutput`, `jumpToAIPromptTarget`
- `frontend/src/ai_pipeline_formatter.js` — `setBlocks`, `setText`
