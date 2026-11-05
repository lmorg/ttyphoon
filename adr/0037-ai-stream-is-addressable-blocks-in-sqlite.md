# 0037 - The AI stream is a set of addressable blocks persisted in SQLite

## Status

Accepted and implemented. Reconciled against the code on 2026-11-04.

The addressable-block migration is complete; it is not an outstanding feature.
Later decisions refine its implementation:

- [0038](0038-typed-stream-blocks-must-honour-panel-scope.md) implements
  workspace/live-history gating and structured restore.
- [0039](0039-remove-legacy-markdown-compatibility-layer.md) removes the
  duplicate live transport and all pre-cutover Markdown transcript compatibility.
- [0042](0042-large-ai-stream-footprint-and-ui-responsiveness.md) owns subsequent
  performance work, including its explicitly outstanding optimizations.
- [0050](0050-nested-subagent-output-uses-typed-blocks.md) preserves typed
  ownership through nested subagent execution.
- [0088](0088-ai-prompt-status-and-historical-disclosures.md) records prompt
  outcomes and historical intermediate-output disclosures.

Supersedes 0006. Amends 0005.

## Context

Before this migration, the panel consumed one linear Markdown stream. Main-agent
text, reasoning, parallel tool output, summaries, subagents, questions and
permission requests all appended to the same sink.

Each producer supplied its own fences or blockquote prefixes. Atomic individual
writes did not make those multi-chunk constructs atomic: reasoning could arrive
inside another producer's open code fence. Buffering each subagent's whole
response, as in ADR 0006, mitigated one producer but sacrificed live feedback
and did not solve tool or summarizer interleaving.

Per-prompt Markdown files also required filesystem scans and heading extraction
for the prompt menu. File deletion could diverge from session-history deletion,
and loading a transcript required a full-document parse.

## Decision

### 1. Addressable blocks and scoped ownership

Represent output as ordered typed blocks, each with its own destination:

| Field | Meaning |
|---|---|
| `RunID` | SQLite-allocated run identity. |
| `BlockID` | SQLite-assigned block row ID, serialized as a decimal string. |
| `ParentID` | Owning tool-call or subagent block, or empty for a root block. |
| `Kind` | Text, thinking, tool call/output/error/summary, subagent, notice or question. |
| `Label` | Separate display metadata, such as a tool name. |
| `Ordinal` | Position allocated by the shared run writer. |
| `Status` | Block lifecycle: `open` or `closed`. |
| `Content` / `Delta` | Stored body / incremental content, without cross-block framing. |

The original design proposed a per-block `Seq` revision. The implemented event
does not contain it: there is no global sequence cursor or gap-recovery timer.
The panel applies lifecycle events and deltas directly to the identified block.

The run owns a shared typed writer, reused across bounded model windows and
continuations. Rolling text and thinking blocks are keyed by both kind and
owning scope. A tool call owns its output, error and summary blocks; nested
subagents share the writer while retaining their own parent scope.

Opening a non-thinking block closes current reasoning in that scope, not in
sibling scopes. Parallel producers do not append framing to a shared string.

### 2. Presentation belongs to the frontend

The formatter maintains independent DOM roots and render queues per block.
Children attach to their parent's child container, including when a child's
event arrives before the parent's event.

- Raw tool-call/output/error/summary payloads render as text in code containers,
  not as Markdown.
- Thinking uses a dedicated DOM structure; its blockquote appearance is
  presentation, not prefixes inserted into stored content.
- Text and other Markdown-capable blocks use incremental rendering scoped to
  their own roots.
- Subagent responses and nested tool activity retain their typed hierarchy.
- Question and permission blocks retain actionable `ttyphoon://` links.

The runtime still supports callback-only consumers when there is no typed
writer. That API is not a second panel transport and does not restore old logs.

### 3. Transport and prompt lifecycle

`aiStreamBlock` is the sole live panel content transport. Open/close events and
delta updates identify their owning block; ordinary updates do not resend a
full accumulated snapshot. `aiJobStart` and `aiJobFinish` identify the run.

Workspace and live/history gating applies to emitted content, not persistence.
Lifecycle notifications still reach the active workspace while its user reads
history, so the UI can track completion without appending to the selected
historical prompt.

**Explicit retained deviation:** the live prompt/command/output header still
travels in `AIJobStart.Title`. Although `StreamBlockRequest` is declared and the
formatter recognizes request blocks, the backend does not emit one. Replacing
the header with persisted request data is a separate refactor, not a missing
piece of addressable output storage or old-log compatibility. Preserve this
header until a replacement is wired through live rendering, restore and export.

### 4. SQLite persistence and deletion

The per-workspace database owns:

- `stream_run_ids`: durable run ID allocation, seeded above pre-existing IDs.
- `stream_prompts`: session/prompt/run identity, heading, timestamps, outcome
  status and block count.
- `stream_blocks`: block identity, hierarchy, ordering, lifecycle and content.

In-flight blocks use prompt ID zero; finalization transactionally assigns the
session-history prompt ID and records the outcome. Block appends accumulate
pending deltas and flush on the configured stream interval and on close.
The database enables WAL and a busy timeout.

Prompt deletion removes stream rows in the same transaction as the history row.
Session deletion and clear-history likewise remove the relevant stream data in
their existing transactions. Direct transcript-clear operations delete stream
rows without deleting independently stored session prompts/final responses.

The session-history response remains independently persisted; the UI transcript
is not its only copy.

### 5. Structured restore and lazy history

`ListPromptLogs` reads SQLite prompt metadata only.

- `ListAIStreamBlocks(sessionID, promptID)` supplies a metadata-only block index.
- `GetAIStreamBlockContent(sessionID, promptID, blockID)` hydrates individual
  bodies.
- `ListAILiveStreamBlocks(workspace)` selects the active run, or the newest
  finalized prompt when no run is active.

The formatter creates shells before hydration and uses an `IntersectionObserver`
to load nearby blocks. Open blocks and the selected historical final answer
hydrate without waiting for intersection. Historical intermediate content
hydrates when its disclosure opens. Already-loaded blocks are not refetched
merely because they intersect again.

This avoids parsing a whole flat transcript at startup. It does **not** imply a
strict viewport-sized memory bound: loaded content remains resident until the
view is cleared or replaced. Further performance work belongs to ADR 0042.

Historical prompt text is read from session history and timestamps from prompt
metadata. See ADR 0088 for final-answer placement and status-menu behaviour.

### 6. Retired Markdown transcripts

New runs do not write Markdown transcript files. Following ADR 0039, old
`session-log.*.md` files are no longer discovered, read, merged, rendered or
deleted by transcript cleanup.

Those files remain untouched on disk. Pre-cutover transcripts are intentionally
unavailable in the panel; their independently stored session-history records
are not removed. There is no migration or flat-rendering fallback.

## Consequences

- Parallel output cannot corrupt another block's Markdown framing.
- Subagents can stream live while keeping nested output independently
  addressable.
- Persistence remains separate from panel visibility and selection.
- Prompt menus and restore use one SQLite-backed transcript model.
- The frontend and backend must agree on block kinds, identity and lifecycle;
  binding changes must follow ADR 0013.
- The block migration's completion does not claim that all performance work is
  finished or that the live request header is already persisted as a block.

## Verification

Existing tests cover concurrent scoped writers, persistent IDs, nested
parent/child output, prompt finalization and deletion, metadata-only queries,
lazy hydration, typed printing/export, historical disclosures, ignoring retired
files and clearing stale output when a workspace has no typed transcript.

For the ADR 0039 cleanup on 2026-11-04, the application and AI Go package tests,
280 Notes tests, and the frontend build passed. This is focused validation,
not a claim that every repository-wide test or vet diagnostic is clean.

## References

- `ai/agent/stream_writer.go`, `stream_writer_test.go` - typed writer and ownership
- `ai/agent/runtime_eino.go`, `subagent_runner.go` - emitter and nested scopes
- `ai/agent/sessiondb/session_db.go`, `stream_blocks.go`, `stream_blocks_delete.go`
  - schema, allocation, queries, finalization and deletion
- `ai/agent/sessiondb/session_log.go`, `session_log_test.go` - lifecycle and
  SQLite-only prompt listing
- `frontend.go` - block metadata/content bindings
- `frontend/src/ai_pipeline_formatter.js`, `notes.js`, `notes.test.js` -
  rendering, hydration, restore and panel lifecycle
