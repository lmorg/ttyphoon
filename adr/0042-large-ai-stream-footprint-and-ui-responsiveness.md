# 0042 - Large AI streams must preserve UI responsiveness

## Status

In progress. Delta-only events, frame-coalesced block rendering, direct typed
delegate streaming, live-panel scroll coalescing, and thinking DOM
specialization are implemented. Raw-block specialization and SQLite connection
reuse remain outstanding.

## Context

A long-running agent caused the AI panel to lock up. The addressable block stream
introduced in 0037 is safer logically, but its current hot path still scales
poorly for large outputs.

### Confirmed cost centres

#### 1. Every event carries the full block content

`aiStreamBlock` events contain both `delta` and `content`. The writer appends to
`h.block.Content` on every token/chunk and copies that entire string into the
event value. For a block of size N delivered in K updates, the transport copies
roughly O(NK) bytes before the browser even renders it. The frontend then keeps
another accumulated `entry.text` and appends each event's delta.

The full `content` snapshot was added for suppression recovery in 0038, but it is
unnecessary for every live delta. A snapshot is needed only when a block is
reloaded from SQLite or explicitly resynchronised.

#### 2. Per-update rendering reparses a growing block

`appendBlock()` queues a render for every event. For markdown/thinking blocks,
`renderIncremental()` receives the entire accumulated `entry.text`; the open
blockquote tail is reparsed repeatedly as it grows. If K reasoning chunks arrive,
this creates substantial repeated parsing and DOM work.

Raw tool blocks avoid markdown parsing, but still update the DOM for every event
and pin `scrollTop` on every event.

#### 3. Multiple async queues can retain stale work

Each block has a promise chain. A fast producer can enqueue many closures before
the browser catches up. Each closure captures its own `text` string, so memory
and work grow while the UI is busy. The chain eventually drains, but the user
experiences a frozen panel first.

#### 4. Main-panel scroll chasing is independent of block rendering

`appendAIText` and the typed block handler both call the AI output scroll logic.
The chase uses repeated animation frames and layout reads. During continuous
streaming this competes with parsing, DOM insertion, syntax/image/table work and
input handling.

#### 5. SQLite is serialized with the same global session lock

Block flushes are batched at 250ms, which is better than one row per token, but
`withStreamDB` takes the global sessiondb mutex and opens/closes the workspace
SQLite handle for each flush. This can contend with history reads and prompt
management during a long run.

The timer previously did **not** gate IPC: `Append` immediately emitted one
`aiStreamBlock` event per provider delta and merely deferred the SQLite write.
Fast streams could therefore fill Wails' main-thread IPC path even while the
database was batched. The persisted flush now emits the accumulated delta only
after a successful write; `Close` emits its final unflushed delta together with
the closed status. This bounds the usual live-event rate to four per second per
open block without dropping content.

## Decision

Optimize in this order, keeping block ownership and persistence semantics intact.
Do not solve UI pressure by dropping durable stream data.

### A. Separate deltas from snapshots

Change `AIStreamBlock` transport semantics:

- `open` event: metadata only, `delta` empty, `content` empty;
- live update: `delta` only, `content` empty;
- `closed` event: `status=closed`, `delta` empty, `content` optional and only
  included if an explicit recovery snapshot is requested.

The frontend appends deltas and never treats an empty content field as a reason
to rebuild. SQLite remains the recovery source. Add a `GetAIStreamBlockContent`
read after suppression/re-entry rather than copying N bytes into every event.

This changes the hot-path transport from O(NK) copied content to O(N) delta data.

Implementation note: live delta and close events now clear `content`. The
formatter still accepts non-empty `content` only for explicit SQLite hydration
snapshots used by historical/live restore.

### B. Coalesce frontend updates per animation frame

`appendBlock()` should update a per-block pending delta string and schedule one
render with `requestAnimationFrame` (or the existing `nextFrame`) rather than
extend a promise chain for every event. At most one render per block per frame
runs. If the producer outruns the browser, deltas coalesce in one buffer instead
of creating K queued closures.

Keep the full accumulated text only for markdown parsing state or recovery, and
consider a bounded render cadence for very large blocks (for example one frame
or 50ms, whichever is later).

Implemented. Each block now has one active render loop and a `needsRender` flag.
Bursts collapse into one follow-up render after `nextFrame()` rather than one
promise closure per event. A synthetic 1,000-update reasoning burst verifies
that markdown parse count remains frame-bounded.

### B1. Stream delegate output directly into typed blocks

Implemented after delegate was identified as the worst multiplier. Previously
each parallel sub-agent accumulated a `strings.Builder`, copied it to a string,
created a second full quoted legacy string, then emitted one giant typed delta.
It now opens its own typed `subagent` block and streams into it on the sub-agent
client's existing interval. Buffering remains only as a compatibility fallback
when no typed block writer exists.

Typed sub-agent streams explicitly disable the legacy prefix/suffix framing;
the client still flushes pending content on return. The backend block handle no
longer retains a second accumulated `Content` copy in addition to its pending
SQLite batch.

### C. Make raw blocks cheap

Tool input/output/error/summary blocks are raw `<pre><code>` content. Append
new delta text directly to the existing text node or `code` node; do not assign
`textContent = fullText` on every event. Pin scroll at the same coalesced render
boundary, not per event.

### D. Make thinking rendering structurally incremental

Implemented. Typed thinking blocks create one stable `<blockquote>`, a fixed
`Thinking:` label, and one text node. Streaming appends only the new delta with
`Text.appendData`; snapshots replace that text node's data without rebuilding
the quote.

Thinking content remains plain text for its entire lifetime, including close and
SQLite hydration. Marked and full markdown post-processing are never invoked for
thinking blocks, so nested `> ` markers and other markdown syntax remain literal
and cannot create deeply nested DOM. Final-answer `text` blocks continue through
the incremental Marked pipeline.

Regression tests verify zero Marked calls for large live bursts, closed thinking
blocks, and restored thinking blocks; stable blockquote identity across updates;
and continued Markdown parsing for final text blocks.

Final-answer `text` blocks also remain append-only plain text while their status
is `open`. The formatter keeps one text node and appends deltas without invoking
Marked. When the backend closes the block, the accumulated final response enters
the incremental Markdown pipeline once. Restored text blocks are already closed,
so they continue to render directly as Markdown.

This distinction is load-bearing: making thinking structural removed reasoning
reparses, but an answer containing nested blockquotes could still reproduce the
same main-thread lock while its `text` block streamed. No token-streaming path
now invokes Marked for thinking or final-answer content.

The live legacy mirror was also removed after it was found to retain a second
formatter and transport path alongside typed blocks. `aiStreamBlock` is now the
only live panel event: the panel no longer supplies a legacy callback, the
backend does not emit `aiResponseStream`, and the frontend does not register its
listener. Typed tool/subagent helpers no longer build duplicate fenced or quoted
strings before discarding them. Old markdown files are still supported only by
the one-shot historical restore path.

### E. Reduce main-panel layout work

Coalesce `scrollAIOutputToBottom()` calls with the same frame scheduler used by
block updates. Do not start or extend a separate chase for every block event.
Keep explicit follow mode: if the user is reading history, do not force the panel
back to the bottom.

Implemented. Live legacy and typed block handlers call
`scheduleAIStreamScrollToBottom`, which permits one `scrollHeight`/`scrollTop`
operation per animation frame regardless of event count. The longer bottom
chase remains reserved for one-shot restores and asynchronous lazy-layout
expansion. A 50-event burst regression verifies one panel scroll write.

### F. Keep the SQLite connection hot during a run

Introduce a run-scoped writer/transaction or cached workspace `*sql.DB` with
careful close/lifecycle handling. Batch block updates by run and commit on the
existing 250ms cadence, block close, and run finish. Do not hold the global
session mutex during slow frontend or model work.

This is lower priority than A-E because the reported symptom is UI lock-up, but
it reduces backend contention for large streams and history reads.

## Memory policy

- Do not retain both a full event snapshot and a full frontend block string.
- Do not import legacy markdown into `stream_blocks` (0039).
- Historical block hydration may retain one copy per hydrated block. Add an
  eviction policy only after measuring real long-history usage; eviction must not
  discard SQLite content.
- Bound or coalesce pending frontend deltas so producer speed cannot create an
  unbounded promise queue.

## Instrumentation before/with implementation

Add debug counters/timing, disabled or sampled by default:

- block delta events received;
- bytes in deltas versus bytes in snapshots;
- pending render count and maximum pending bytes;
- render duration by block kind;
- markdown parse count and duration;
- SQLite flush duration and batch bytes;
- dropped/coalesced frontend render passes.

Use a synthetic large-stream test fixture rather than a live provider for
regression tests. Test at least: 1MB raw output, 1MB reasoning, 1000 updates,
parallel tool blocks, suppressed-and-reloaded blocks, and user scrolling during
streaming.

## Suggested implementation order

1. Add counters and a synthetic stream benchmark/test.
2. Remove full `content` from live delta events (A).
3. Replace per-event promise chains with frame coalescing (B).
4. Optimize raw block text updates and thinking DOM updates (C/D).
5. Coalesce main-panel scrolling (E).
6. Profile SQLite and add a run-scoped connection/batch if still material (F).

## Consequences

- The frontend becomes frame-bounded instead of token-bounded.
- Stream recovery relies explicitly on SQLite snapshots, which is the intended
  persistence boundary from 0037/0038.
- Live events become smaller and cheaper to serialize, transport, parse and
  garbage-collect.
- More runtime state lives in the formatter scheduler, so tests must cover
  coalescing and final flush on close.
- The optimization must not silently drop deltas or alter block ordering.

## Reference

- `ai/agent/stream_writer.go` — block content copies and 250ms SQLite flush
- `ai/agent/runtime_eino.go` — typed block event emission
- `frontend/src/ai_pipeline_formatter.js` — `appendBlock`, `renderIncremental`,
  block queues and live rendering
- `frontend/src/notes.js` — `aiStreamBlock`, scroll chasing and markdown post-processing
- `ai/agent/sessiondb/stream_blocks.go` — block persistence and recovery reads
- 0037 — addressable SQLite stream blocks
- 0038 — workspace/prompt scope and block recovery
- 0040 — transient stream retry
