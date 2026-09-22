# 0039 - Scheduled removal of the legacy markdown compatibility layer

## Status

Proposed. Earliest execution ~2027-03 (six months after 0038).

Completes 0037. Supersedes the compatibility clauses of 0037 and 0038 D.

## Context

0037 replaced the linear markdown AI stream with addressable sqlite blocks, and
0038 fixed the scope defects that migration introduced. Both deliberately left a
compatibility layer in place:

- **Legacy markdown files.** `session-log.{ws}.{sid}.{pid}.md` are no longer
  written (0037 step 12) but existing files are still read, listed, merged with
  sqlite prompts, and deleted alongside sqlite rows.
- **The `aiResponseStream` transport.** Every block emit is mirrored as a
  markdown chunk on the old event, and the frontend suppresses the mirror once
  it has seen a typed block for that run.
- **Markdown framing helpers.** Fences, blockquote prefixes and sub-agent labels
  are still generated, solely to feed that mirror.

The only thing keeping any of this alive is **markdown files already on users'
disks from before the cutover**. Nothing creates new ones. As those sessions are
deleted over time the layer becomes dead weight, and it is substantial: it is
roughly half of `session_log.go`, a third of `ai_pipeline_formatter.js`, and a
duplicated argument threaded through every emit helper.

Carrying two transports also caused the 0038 defects. Every emit site has to
remember to gate both, and one that forgets fails silently.

## Decision

Remove the compatibility layer in one change, once the precondition below holds.
Record the full inventory now, while the reasoning is fresh, because in six
months the coupling will not be obvious from the code alone.

### Precondition

Legacy files live in `~/Documents/{app.DirName}/`. Before starting:

```bash
ls ~/Documents/ttyphoon/session-log.*.md 2>/dev/null | wc -l
```

Removal is safe when the remaining files belong only to sessions the user is
willing to lose the *transcript* for. The authoritative prompt/response pair is
always in `sessions_{n}.llm_response`; only the rendered transcript is lost.

There is no automatic migration, deliberately: importing legacy markdown into
`stream_blocks` would duplicate it on disk and require parsing the retired
framing back out, which is lossy (0038 D).

### Prerequisite: emit the request header as a block

**This is the one item that is not a deletion, and it blocks several others.**

0037 specified that the request header "becomes a `request` block". That was
never implemented. `StreamBlockRequest` is declared in `stream_blocks.go` and
never emitted; the header is still built as markdown by
`buildSessionLogRequestPrefix` and shipped through `AIJobStart.Title`.

So before `buildSessionLogRequestPrefix` can be deleted:

1. Emit the prompt, command line and output block as a `request` block at
   `SESSION_LOG_START_JOB`.
2. Render it in the formatter from `kind === 'request'`.
3. Drop `Title` from `AIJobStart`, and the `Query`, `CommandLine` and
   `OutputBlock` fields from `SessionLogContext` once nothing else reads them
   (`summarizeRequestHeading` still needs `Query` for `stream_prompts.heading`).

### Removal inventory

#### 1. Legacy markdown files — `ai/agent/sessiondb/session_log.go`

Delete: `sessionLogDir`, `sessionLogPendingPath`, `sessionLogPromptPath`,
`sessionLogPromptFilesGlob`, `appendSessionLog`, `writePromptLogHeader`,
`buildSessionLogRequestPrefix`, `writeSessionOutput`, `head`, `maxLines`,
`buildSessionLogFinalizeSuffix`, `listPromptLogMetas`, `promptLogFilenameRx`,
`readRequestHeadingComment`, `requestHeadingCommentRx`, `GetSessionLog`,
`GetPromptLog`.

Collapse: `listPromptLogMetasMerged` → `ListStreamPromptMetas` directly;
`ListPromptLogs` keeps its name and loses the merge. Strip the file-removal
halves of `ClearSessionLog` and `DeletePromptLog`, leaving the sqlite deletes.

Then drop the now-unused `os`, `path/filepath`, `regexp` and `app` imports.

#### 2. The `aiResponseStream` transport

- `AIStreamChunk`; the `SESSION_LOG_APPEND_CHUNK` state and its emit; the
  `streamed` and `sequence` fields of `sessionLogState`; `AIJobFinish.FinalSequence`.
- `ai/ui.go` — `emitAIResponseChunk`, and the `streamCallback` argument threaded
  through `RunLLMWithMessageStream` / `runLLMWithMessageStream` /
  `runBoundedWindow` once nothing consumes it.
- `frontend/src/notes.js` — the `EventsOn("aiResponseStream")` handler,
  `aiBlockStreamRunId` and every assignment to it, `appendAIText`.

#### 3. Markdown framing helpers — `ai/agent/runtime_eino.go`

Delete: `formatToolCallMarkdown`, `formatToolOutputMarkdown`,
`formatToolErrorMarkdown`, `formatToolSummaryNoticeMarkdown`,
`formatToolSummaryFailureMarkdown`, `summariserStreamOpenMarkdown`,
`summariserStreamCloseMarkdown`.

On `aiStreamEmitter`, delete `fn`, `pending`, `inThinking`, `lastEmit`,
`flushTimer`, `emitLegacyText`, `emitLegacyTextLocked`, `flush`, `flushLocked`
and `aiStreamEmitInterval`. The emitter becomes a thin wrapper over the block
writer, and the 100ms reasoning coalescing moves to the writer's existing 250ms
flush.

Collapse the emit helpers' duplicated `legacy` parameter:
`emitAIStreamToolBlock` / `emitAIStreamToolBlockID` /
`emitAIStreamToolBlockIDWithLabel` lose it, and
`EmitAIStreamBlockWithLegacy` + `EmitAIStreamLegacy` fold into
`EmitAIStreamBlock`. `AIStreamBlockEmitter.Emit` drops its mirror call.

**Do not delete `formatToolPermissionRequestMarkdown` or
`formatUserQuestionRequestMarkdown`.** Despite the naming they are not framing:
they build the interactive `ttyphoon://` links that *are* the content of
`question` blocks. They currently pass the same string as both content and
legacy, which makes them look like framing helpers. Keep them; drop the
duplicate argument.

#### 4. Sub-agent framing

- `ai/subagent/subagent.go` — `Request.StreamPrefix`, `Request.StreamSuffix`,
  `Request.FormatStreamChunk`, and `Quote`, plus the prefix/suffix emission in
  `Run`.
- `ai/tools/subagents/subagent.go` — the `legacy := fmt.Sprintf(...)` blockquote
  construction.
- `ai/agent/summariser.go` — the `streamPrefix` / `streamSuffix` fallback branch
  taken when no block writer exists.

#### 5. Wails bindings

Delete `GetAISessionCache` and `GetAIPromptLog` from `frontend.go`, their
imports in `notes.js`, and their entries in the generated `WApp.js` and
`WApp.d.ts`. Regenerate rather than hand-editing if the toolchain allows.

#### 6. Frontend — `ai_pipeline_formatter.js`

The one-shot flat render path goes entirely: `setText`, `renderTextAsMarkdown`,
`buildLazyChunks`, `primeLazyChunks`, `processLazyChunk`, `lazyChunkObserver`,
`streamText`, `legacyRender`, and `appendChunk`.

`appendChunk` is the only caller of `renderCurrentStream`, which is the only
caller of the pipelined-section machinery, so that goes too: `parseSections`,
`SECTION_REGEX`, `MAX_SECTION_HEADER_LEN`, `createSectionCache`,
`resetSectionCache`, `sectionCache`, `sectionLengths`, `buildSectionShell`,
`patchMarkdownSection`.

**Keep** `renderIncremental`, `planSplit`, `openFenceTail`, `closeFenceFor` and
`canStartNewBlock` — the block path renders markdown blocks through them.

In `notes.js`, `setAIFinalOutput` loses its last caller and goes with them.

#### 7. CSS — `notes.css`

Remove `.notes-ai-legacy`, `.notes-ai-legacy::before`, `.notes-ai-lazy-chunk`,
`.notes-ai-lazy-chunk-content`, `.notes-ai-lazy-spinner*` and `.notes-ai-section`.

**Keep** `.notes-ai-heading` — reused by the tool-call heading — and
`.notes-ai-stable` / `.notes-ai-tail` / `.notes-ai-batch`, which
`renderIncremental` depends on.

#### 8. Tests

Delete: `session_log_delete_prompt_test.go`, `session_log_prefix_test.go`,
`session_log_mixed_test.go`; `TestFormatToolCallMarkdown_UsesTildeFences`,
`TestFormatToolOutputMarkdown_UsesTildeFences`,
`TestAIStreamEmitter_WrapsReasoningAsBlockquote`,
`TestAIStreamEmitter_ReopensBlockquoteAfterText`,
`TestWriteToSessionLog_DoesNotCreateMarkdownFile` (vacuous once nothing writes
files).

Frontend: the `aiResponseStream` handler tests, `delivers legacy stream chunks
directly without a global cursor`, and `flags legacy transcripts and clears the
flag for block renders`.

Several AI-panel tests drive rendering through `aiResponseStream`; rewrite them
against `aiStreamBlock` rather than deleting them, or panel coverage drops
sharply.

### Suggested order

Deleting bottom-up keeps the tree compiling at each step:

1. Request block prerequisite.
2. Frontend: legacy handler, flat render path, CSS, bindings imports.
3. Wails bindings.
4. Go: framing helpers and the emitters' `legacy` parameter.
5. Go: `aiResponseStream` transport and `streamCallback` plumbing.
6. Go: markdown file helpers.
7. Tests, in step with each of the above.

## Consequences

- One transport, one gating predicate. The 0038 class of defect — a second emit
  path that forgets to gate — becomes unrepresentable.
- Transcripts for pre-cutover prompts are lost. The prompt and final response
  survive in `sessions_{n}`; only the rendered intermediate output goes.
- Orphaned `session-log.*.md` files remain on disk once `ClearSessionLog` stops
  removing them. They are inert. Deleting user data automatically is not worth
  the risk; document the path instead.
- `~1000 lines` come out across both languages, and every emit helper loses a
  duplicated argument.
- **Lesson: date-stamp compatibility layers when you add them.** This one was
  load-bearing for exactly one reason — files already on disk — but that reason
  was spread across seven layers of code, and nothing in those layers said so.

## Reference

- `ai/agent/sessiondb/session_log.go` — file helpers, `AIStreamChunk`,
  `SESSION_LOG_APPEND_CHUNK`
- `ai/agent/runtime_eino.go` — `format*Markdown`, `aiStreamEmitter`, emit helpers
- `ai/agent/summariser.go`, `ai/subagent/subagent.go`,
  `ai/tools/subagents/subagent.go` — sub-agent and summariser framing
- `ai/ui.go` — `emitAIResponseChunk`
- `frontend.go` — `GetAISessionCache`, `GetAIPromptLog`
- `frontend/src/ai_pipeline_formatter.js` — `setText`, section machinery
- `frontend/src/notes.js` — `aiResponseStream` handler, `setAIFinalOutput`
- 0037 (the migration), 0038 (the scope fixes this completes)
