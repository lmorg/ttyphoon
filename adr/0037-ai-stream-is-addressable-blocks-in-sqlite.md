# 0037 - The AI stream is a set of addressable blocks persisted in sqlite

## Status

In progress. Amended by 0038, which fixes the emit gating this record's
`aiStreamBlock` transport omitted, and the flattened live-restore path.
The legacy fallbacks retained below are scheduled for removal by 0039.

The SQLite schema foundation and idempotent initialization are
implemented. The runtime now owns typed SQLite blocks for text, reasoning, and
tool progress, and prompt finalization assigns the database prompt ID
transactionally. The runtime emits `aiStreamBlock` lifecycle events, and the
frontend renders each block in an isolated DOM root. The legacy markdown
transport remains active as a fallback for runs that do not produce blocks.
Tool calls, outputs, errors, summaries, questions, permissions, and completed
sub-agent output now receive dedicated block kinds rather than sharing a text
sink. Typed block content is raw; markdown fences, quote prefixes, and
sub-agent labels are generated only for the legacy compatibility stream.
Historical prompts now use `ListAIStreamBlocks(sessionID, promptID)` for a
metadata-only index and `GetAIStreamBlockContent(sessionID, promptID, blockID)`
for viewport-driven lazy hydration through the Wails bindings. Legacy markdown
prompt loading remains the fallback for prompts without block rows.
The frontend formatter now maintains a separate render queue and DOM root per
`blockId`; raw tool blocks use `textContent` in `<pre><code>`, while markdown
blocks are incrementally rendered within their own root.
`ParentID` is now propagated through the runtime context: a tool-call block is
the parent of its output, error, summary, and nested sub-agent blocks. The
frontend attaches child wrappers under a parent's dedicated children container,
including when the child event arrives before the parent event.
Continuous reasoning chunks share one `thinking` block, but opening any
non-thinking logical block closes it first. Reasoning after a tool call therefore
opens a new block instead of being appended to the earlier thinking block.
Tool-call blocks persist their tool name as separate metadata and render it as a
`Tool call: \`name.of.tool\`` heading above the untouched input payload.
The frontend no longer maintains a global stream sequence cursor or gap
recovery timer. Typed block events update their own roots directly; the legacy
`aiResponseStream` event is a direct-delivery compatibility fallback only.
Prompt metadata and assembled prompt content are now read from
`stream_prompts`/`stream_blocks` first. Existing markdown prompt files remain a
read fallback for pre-migration prompts that have no SQLite stream rows.
Prompt deletion removes its stream rows in the same transaction as the history
row. Session deletion and clear-history remove both stream tables in their
existing transactions; direct log-clearing APIs also remove SQLite rows before
cleaning up legacy files.
New runs no longer create, append, finalize, or rename static markdown session
logs. The old file format remains read-only compatibility storage for prompts
created before this cutover and is still removed by cleanup operations.
Lazy historical loading is covered by formatter tests: a small initial set is
prewarmed, off-screen blocks remain metadata shells, intersection hydrates one
block asynchronously, and repeated intersection does not refetch it.
Final verification includes a race-tested concurrent parent/child block writer,
SQLite prompt deletion coverage, nested DOM hierarchy coverage, and full Go and
frontend suites. These cover the parallel-tool/sub-agent ownership path without
reintroducing a shared markdown sink.
The final audit also verifies mixed legacy-file and SQLite prompt histories, with
SQLite taking precedence for duplicate prompt IDs. Full Go tests, AI race tests,
and the frontend suite pass. `go vet ./...` still reports unrelated existing
diagnostics in other packages and the pre-existing `frontend.go` timeout warning.

Supersedes 0006. Amends 0005.

## Context

The AI panel's live output is a **single linear markdown string**. Every producer
appends to it:

- the main agent's text and reasoning (`aiStreamEmitter.emitText` / `emitReasoning`)
- each tool call, tool output and tool error (`einoAgentTool.InvokableRun`)
- the tool-output summariser (`summariseToolOutput`)
- sub-agents (`ai/tools/subagents`, `ai/subagent`)
- permission and `askUser` prompts

Each producer frames its own content *in markdown*: `~~~~` fences for tool
output, `> **Thinking:** ` blockquote prefixes for reasoning, `> **Sub-agent X:** `
for sub-agents. Framing is therefore **stateful across many chunks** — a
summariser opens a fence, streams for seconds, then closes it.

`aiStreamChunk.Sequence` guarantees *delivery order*, not *logical grouping*, and
`aiStreamEmitter.mu` guarantees each individual write is atomic but nothing more.
So a reasoning flush timer (`time.AfterFunc(aiStreamEmitInterval, …)`) firing
between two summariser chunks lands a blockquote **inside an open code fence**:

```
| Summaring tool output:

> **Summaring tool output:** [HAP-13379](…): "Team review of the

> Thinking: I

Web

> Thinking: need to distill this down to the key facts while keeping it concise.
```

This is the same defect 0006 identified for sub-agents. 0006 solved it by
abandoning live streaming for sub-agents and flushing each job as one contiguous
block on completion. That mitigates one producer; it does not fix parallel tool
calls, the summariser, or reasoning racing tool output, and it costs live
feedback and unbounded per-job memory. Nested sub-agents make it worse still:
their markdown framing nests blockquotes inside blockquotes inside fences.

Secondary problems with the current design, all downstream of "the stream is one
string":

- **Persistence is a markdown file per prompt.** `session-log.{ws}.{sid}.{pid}.md`,
  with the prompt heading scraped back out of an HTML comment by regex, and the
  prompt list built by `filepath.Glob` + `os.Stat`.
- **File and DB can diverge.** `DeleteAIHistoryEntry` deletes the sqlite row and
  then best-effort `os.Remove`s the file. Either can fail alone.
- **One missing chunk stalls the panel permanently.** The old frontend cursor
  required contiguous sequences; addressable blocks remove that dependency.
- **History loads as one blocking parse.** `setText` reads the whole file and
  calls `marked.parse` on all of it before lazy post-processing can begin.

## Decision

Stop treating the panel stream as text. Model it as an **ordered set of
addressable blocks**, persisted in the existing per-workspace sqlite database,
and delete the static markdown log.

### 1. A block is the unit of streaming, and has exactly one writer

| Field | Meaning |
|---|---|
| `BlockID` | `"{runID}-{n}"` — unique, sortable, cheap, greppable in logs. A UUID is not required. |
| `ParentID` | Empty, or the owning block (a sub-agent's children, a summary under its tool call). |
| `Kind` | `request`, `text`, `thinking`, `tool-call`, `tool-output`, `tool-error`, `tool-summary`, `subagent`, `notice`, `question`. |
| `Ordinal` | Monotonic position within the prompt, allocated under the run lock. |
| `Seq` | Per-block revision, so a late delta for one block cannot stall another. |
| `Status` | `open` or `closed`. |
| `Content` | The block's own markdown body, **without framing**. |

The invariant that makes concurrency safe is **one writer per block**. A block
is opened at the point a logical unit begins and closed when it ends:

- `einoAgentTool.InvokableRun` opens a `tool-call` block, and a `tool-output`
  child. Parallel tool calls own disjoint blocks, so they cannot interleave.
- `summariseToolOutput` opens a `tool-summary` block parented to its tool call.
- `aiStreamEmitter` splits into two writers: a `text` block per contiguous text
  run and a `thinking` block per contiguous reasoning run. The `inThinking`
  flag disappears — "am I inside a blockquote" becomes "which block is open".
- Sub-agents open a `subagent` block and stream into it **live**. Nested
  sub-agents parent their blocks to it.

The writer handle is carried in `context.Context`, replacing the existing
`*aiStreamEmitter` stored under `aiStreamCallbackCtxKey{}`. That plumbing already
exists and already reaches every call site.

### 2. Framing is presentation, not content

`formatToolCallMarkdown`, `formatToolOutputMarkdown`, `summariserStreamOpenMarkdown`,
`StreamPrefix`/`StreamSuffix` and the `> ` prefixing in `emitReasoning` are
deleted. The frontend renders framing from `Kind`:

- `tool-output` / `tool-error` / `tool-summary` render into a `<pre><code>` via
  `textContent`. They are **never** passed to `marked`.
- `thinking` renders into a styled container, not a markdown blockquote.
- `subagent` renders a container whose children are DOM children.

This structurally removes the entire class of "fence leaked", "blockquote
interleaved" and "nested fence" defects: two producers' text is never
concatenated, and untrusted tool output is never parsed as markdown.

### 3. Transport

Replace `AIStreamChunk` with one event carrying an open/delta/close:

```go
type AIStreamBlock struct {
	RunID    uint64 `json:"runId"`
	BlockID  string `json:"blockId"`
	ParentID string `json:"parentId"`
	Kind     string `json:"kind"`
	Ordinal  int64  `json:"ordinal"`
	Seq      uint64 `json:"seq"`
	Delta    string `json:"delta"`
	Status   string `json:"status"` // open | closed
}
```

`aiJobStart` / `aiJobFinish` survive unchanged in spirit; the request
header currently smuggled through `AIJobStart.Title` becomes a `request` block.

Gap-free *global* sequencing is no longer required — only per-block contiguity,
which single-writer ownership makes automatic. The old snapshot and
gap-recovery path is deleted; the panel updates each block directly.

### 4. Persistence: sqlite, not files

Two new tables in the existing `session.{workspace}.db`:

```sql
CREATE TABLE IF NOT EXISTS stream_blocks (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	sessionId INTEGER NOT NULL,
	promptId  INTEGER NOT NULL DEFAULT 0,  -- 0 until the run is finalized
	runId     INTEGER NOT NULL,
	blockId   TEXT    NOT NULL,
	parentId  TEXT    NOT NULL DEFAULT '',
	kind      TEXT    NOT NULL,
	ordinal   INTEGER NOT NULL,
	status    TEXT    NOT NULL DEFAULT 'open',
	content   TEXT    NOT NULL DEFAULT '',
	created   TEXT    NOT NULL,
	updated   TEXT    NOT NULL,
	UNIQUE(sessionId, runId, blockId)
);
CREATE INDEX IF NOT EXISTS stream_blocks_prompt_idx
	ON stream_blocks(sessionId, promptId, ordinal);

CREATE TABLE IF NOT EXISTS stream_prompts (
	sessionId  INTEGER NOT NULL,
	promptId   INTEGER NOT NULL,
	runId      INTEGER NOT NULL,
	heading    TEXT    NOT NULL,
	started    TEXT    NOT NULL,
	finished   TEXT    NOT NULL DEFAULT '',
	blockCount INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (sessionId, promptId)
);
```

`stream_prompts` replaces `listPromptLogMetas`' glob + `os.Stat` + the
`<!-- request heading: … -->` regex scrape. The heading is a column.

**Do not write a row per token.** An open block keeps its content in a
`strings.Builder` (as `sessionLogState.streamed` does today) and flushes on a
~250ms debounce, and unconditionally on block close and on run finish. Crash
exposure is bounded to one debounce window, which is acceptable: this is a UI
transcript, and `sessions_{n}.llm_response` remains the authoritative record of
the final answer. Enable `PRAGMA journal_mode=WAL` and `busy_timeout`, since the
write rate is now materially higher than the current schema's.

Deletion becomes atomic and uses the transaction that already exists:

- `DeleteAIHistoryEntry` → `DELETE FROM stream_blocks WHERE sessionId=? AND promptId=?`
  plus the `stream_prompts` row, **in the same tx** as the history row delete.
- `DeleteAISession` → delete by `sessionId` in the tx that drops `session_{n}`.
- `ClearAISessionHistory` / `ClearAILog` → delete all blocks for the active session.

`ClearSessionLog`, `DeleteSessionLog`, `DeletePromptLog`, `ClearActiveSessionLog`
keep their names and signatures so `frontend.go` call sites are unchanged; only
their bodies move from `os.Remove` to SQL.

### 5. Lazy history loading

`ListPromptLogs` returns `stream_prompts` rows. Selecting an older prompt:

1. `GetAIPromptBlockIndex(sessionId, promptId)` returns `{blockId, kind, ordinal,
   parentId, size}` for every block — metadata only, no content.
2. The frontend immediately renders an empty shell per block. Scroll height is
   right and the UI never blocks.
3. An `IntersectionObserver` — the existing `lazyChunkObserver` pattern in
   `ai_pipeline_formatter.js` — calls `GetAIBlockContent(blockId)` as blocks
   approach the viewport, rendering each in its own async task.

This is strictly better than `setText`, which reads and `marked.parse`s an entire
session log synchronously before any lazy work begins, and it caps resident
memory at the blocks actually on screen.

### 6. Legacy markdown logs

The file **writer** is removed immediately. Existing `session-log.*.md` files are
left on disk and read as a fallback when a prompt has no rows in `stream_blocks`,
rendered as a single legacy `text` block. No migration step, no data loss, and
the files age out as sessions are deleted.

## Consequences

- Concurrent producers cannot corrupt each other's output, by construction rather
  than by buffering. Parallel tool calls, the summariser, reasoning and
  sub-agents are all covered by the same mechanism.
- **0006 is superseded**: sub-agents stream live again, and their memory no longer
  grows with total job output.
- Nested sub-agents nest in the DOM, not in markdown. Blockquote-in-fence-in-
  blockquote is not representable.
- Tool output is never parsed as markdown, which is also the correct security
  posture for content returned by an MCP server.
- **0005 is amended**: the `panelView` live/history gate stays, but dropped emits
  no longer desync anything, so the snapshot/gap-recovery machinery it justified
  is removed. Persistence remains separate from presentation — the target is now
  sqlite rather than a file.
- Deleting a prompt or session is atomic. The transcript can no longer diverge
  from the history row.
- sqlite write volume rises. Mitigated by the debounce, WAL, and batching a run's
  flushes; but note `sessiondb`'s single global `mu` now guards a hotter path and
  should be measured, and the debounce flush must not hold it across a render.
- Blocks left `open` by a crashed or cancelled run must be swept closed on
  `SESSION_LOG_FINISH_JOB`, else the panel shows a permanent spinner.
- The transport change is not backwards compatible: backend and frontend must
  land together (see 0013 for the three-edit binding checklist).

## Suggested sequencing

1. `sessiondb/stream_blocks.go` — schema, migration, block writer, queries + tests.
2. Block writer in `context`, replacing `aiStreamEmitter`; update `InvokableRun`,
   `summariseToolOutput`, `ai/subagent`, `user_question.go`, permission prompts.
   Delete the `format*Markdown` framing helpers.
3. Wails bindings and events.
4. `ai_pipeline_formatter.js` → block-addressed rendering, `renderIncremental`
   scoped per block (its `WeakMap` keying already supports this).
5. Delete the file writer and the snapshot/gap-recovery path. (Complete.)

## Reference

- `ai/agent/sessiondb/session_log.go` — `WriteToSessionLog`, `sessionLogState`,
  `AIStreamChunk`, `listPromptLogMetas`
- `ai/agent/sessiondb/session_db.go` — `openDB`, `initDB`, `mu`, per-session tables
- `ai/agent/runtime_eino.go` — `aiStreamEmitter`, `format*Markdown`,
  `aiStreamCallbackCtxKey`
- `ai/agent/summariser.go` — `StreamPrefix` / `StreamSuffix` framing
- `ai/subagent/subagent.go`, `ai/tools/subagents/subagent.go` — 0006's buffering
- `frontend/src/ai_pipeline_formatter.js` — `renderIncremental`, `planSplit`,
  `openFenceTail`, `setText`, lazy chunks
- `frontend/src/notes.js` — direct block event handling and legacy fallback
- `frontend.go` — `GetAISessionCache`, `GetAIPromptLog`, `ListAIPromptLogs`,
  `DeleteAIHistoryEntry`, `DeleteAISession`, `ClearAISessionHistory`
