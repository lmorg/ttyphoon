# 0039 - Remove the legacy Markdown transcript compatibility layer

## Status

Accepted and implemented, 2026-11-04. The user confirmed that backward
compatibility with pre-cutover Markdown logs is no longer required.

The duplicate live `aiResponseStream` transport was removed on 2026-09-23.
This change removes the remaining old-file compatibility paths. It completes
the compatibility removal scheduled by ADR 0037 and ADR 0038.

## Context

ADR 0037 replaced linear Markdown AI transcripts with addressable SQLite blocks.
Historical `session-log.{workspace}.{sessionId}.{promptId}.md` files remained
readable through a second backend and frontend path, even though new requests
no longer wrote those files.

The original removal inventory also included functionality that still serves
current requests. It must not be used as a verbatim deletion checklist:

- `AIJobStart.Title` still carries the live prompt, command and output header.
  `StreamBlockRequest` exists but is not emitted by the backend.
- Callback-only runtime streaming still works without a typed block writer.
- The file-index indicator and structured-document viewer share spinner CSS
  with the former legacy transcript renderer.

None of these current dependencies requires reading an old transcript.

## Decision

- List prompts exclusively from `stream_prompts`; do not scan or merge Markdown
  files into the prompt menu.
- Restore and display AI output exclusively through typed stream block metadata
  and per-block content hydration. Remove the flat concatenation endpoint and
  the legacy `GetAISessionCache` and `GetAIPromptLog` Wails bindings.
- Remove the formatter's `setText`, `appendChunk`, legacy section parser,
  one-shot transcript renderer, and transcript-only CSS. Keep incremental
  Markdown rendering and viewport-driven hydration for typed blocks.
- Remove the old-file prompt/timestamp fallback and DOM-based prompt menu
  collector. Current historical prompt text comes from session history and
  timestamps from SQLite prompt metadata.
  Panel initialization and workspace restoration use the same stored-prompt
  rendering path as explicit prompt-menu selection, including the full prompt,
  timestamp, and collapsed intermediate output.
- A workspace with no typed transcript clears the panel rather than retaining
  the previous workspace's output or attempting to render an old file.
- Clear/delete operations remove SQLite stream rows only. Do not read, migrate,
  or delete retired Markdown transcript files.
- Preserve the current live request header, runtime callback API, interactive
  question/permission links, shared spinners, and typed printing/export support.
  Moving the live request header into a persisted request block is a separate
  refactor, not a prerequisite for dropping old-log compatibility.

## Consequences

- Pre-cutover Markdown transcripts are no longer available in the AI panel.
  Their original files remain untouched in `~/Documents/ttyphoon/` (the shared
  application documents directory). They can be removed manually if desired.
- Stored prompts, command/output context, and final responses remain in the
  session history database. This change does not recreate panel transcripts
  for those older entries.
- New and existing SQLite-backed transcripts retain typed rendering, historical
  disclosures, lazy loading, and printing/export.
- Regression tests cover ignoring retired files, preserving them during
  transcript deletion, SQLite prompt ordering and deletion, empty-workspace
  clearing, typed rendering, and the shared loading styles.

## References

- [ADR 0037](0037-ai-stream-is-addressable-blocks-in-sqlite.md)
- [ADR 0038](0038-typed-stream-blocks-must-honour-panel-scope.md)
- [ADR 0088](0088-ai-prompt-status-and-historical-disclosures.md)
- `ai/agent/sessiondb/session_log.go`, `stream_blocks.go`
- `frontend.go`, generated Wails bindings
- `frontend/src/ai_pipeline_formatter.js`, `notes.js`, `notes.css`
