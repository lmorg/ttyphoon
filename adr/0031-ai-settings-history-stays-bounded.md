# 0031 - AI Settings history stays bounded regardless of what produced it

## Status

Accepted.

## Context

Opening AI Settings calls `GetAISessionManagement` -> `sessiondb.GetFrontendState`,
which loads both the session list and the active session's recent history for
rendering in the modal. Two independent problems made this open hang for
workspaces with a long AI history:

1. `loadSessionMetasTx` ran a `SELECT COUNT(1) FROM session_<id>` — a full
   table scan — for every session, on every single open, to populate
   `EntryCount`. This cost scaled with total historical rows across the
   workspace and was recomputed from scratch every time the modal opened.

2. History items are rendered directly into `innerHTML` in
   `renderAISessionHistory()`. A stray base64 image data URL previously sent
   through the generic "ask AI about this document" path (see amendment to ADR
   0029 below) got persisted verbatim as `output_block`. Rendering a multi-KB
   string with no whitespace as a single DOM text node is pathologically slow
   for a browser layout engine, since there is nowhere to break the line. This
   froze the modal even for a workspace with very few prompts, as long as one
   of them referenced an image this way.

## Decision

**Maintain `entryCount` incrementally, not by scanning.** `sessions_meta` gained
an `entryCount` column, updated in the same transaction as
`AppendActiveSessionEntry` (+1) and reset to 0 by `ClearActiveSession`.
`loadSessionMetasTx` now reads the column directly. A database created before
this column existed is migrated once via `migrateEntryCountColumn`: it detects
the missing column, adds it, and backfills every session with the old
full-scan `COUNT(1)` exactly one time. Every read after that is O(1) per
session.

**Cap history fields at render time, regardless of source.** `Prompt`,
`CommandLine` and `OutputBlock` are passed through `truncateHistoryField`
(4000 chars) before being returned to the frontend. This is deliberately
independent of *how* a field got large — it protects against any future
regression that stores an oversized value, not just the one case identified
here. It does not touch what is stored on disk or what the model saw; it only
bounds what the Settings modal renders.

## Consequences

- Opening AI Settings is O(sessions) instead of O(total history rows) after
  the one-time migration.
- A pre-existing giant `output_block` in someone's database can no longer
  freeze the modal, without needing a manual database edit.
- `Response` (the full LLM response) is intentionally left untruncated: the
  frontend does not currently render it in the Settings modal. If that
  changes, it will need the same treatment.

## Follow-up: per-entry deletion

The original toolbar `[Clear]` button erased the whole active session's
per-prompt log files at once (`ClearAILog`/`ClearActiveSessionLog`). It was
relocated into this same "Active Session Transcript" list, but as a
per-item eraser button rather than a single global action, so a single
oversized or unwanted entry can be removed without discarding the rest of the
history.

`DeleteActiveSessionEntry(workspace, entryID, limit)` deletes one row from the
active session's history table by `id` and decrements `entryCount` (floored at
0), reusing the same maintained column rather than reintroducing a scan.
`DeletePromptLog(workspace, sessionID, promptID)` removes that entry's
corresponding per-prompt markdown log file — the row `id` and the log's
`promptID` are the same value, since `AppendActiveSessionEntry`'s returned row
id is what gets passed as `PromptID` when the log is finalized.

`WApp.DeleteAIHistoryEntry(entryID)` composes both calls. The frontend renders
one eraser button per transcript item (`data-action="erase"`,
`data-entry-id`), styled like `.notes-ai-settings-session-delete`.

## References

- `ai/agent/sessiondb/session_db.go` - `migrateEntryCountColumn`,
  `loadSessionMetasTx`, `truncateHistoryField`, `loadFrontendHistoryTx`,
  `DeleteActiveSessionEntry`
- `ai/agent/sessiondb/session_log.go` - `DeletePromptLog`
- `frontend.go` - `DeleteAIHistoryEntry`
- `ai/agent/sessiondb/session_db_entrycount_test.go`
- `ai/agent/sessiondb/session_db_history_truncation_test.go`
- `frontend/src/notes.js` - `renderAISessionHistory`
