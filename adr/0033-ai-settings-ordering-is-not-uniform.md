# 0033 - AI Settings ordering is deliberately not uniform

## Status

Accepted.

## Context

Two lists in the AI Settings modal both come from `sessiondb` and both look
like "most recent first", but the underlying rule for each is different, and
getting either one wrong reintroduces a bug that was just fixed:

1. **Session list.** `loadSessionMetasTx` originally ordered by
   `active DESC, updated DESC, tableId DESC`, which pinned the currently active
   session to the top regardless of its actual `updated` timestamp. Selecting
   a session also bumped its `updated` value via `setActiveSessionTx`. Net
   effect: merely looking at an old session moved it to the top of the list,
   which is not what "last updated" means to a user - only running a prompt in
   a session (or clearing it) is an update.

2. **Active session transcript (history).** `loadEntriesTx` queries
   `ORDER BY id DESC` but then reverses the result back to ascending order
   before returning, because its *other* caller, `ActiveSessionEntries`, feeds
   chronological conversation history back to the LLM for context
   reconstruction - it would be actively wrong for that consumer to receive
   newest-first order. `loadFrontendHistoryTx` was forwarding that same
   chronological order straight to the frontend, so the transcript displayed
   oldest-first, and `sessions.slice(0, 12)` in the frontend was keeping the
   oldest 12 entries once a session grew past 12, not the newest.

## Decision

**Sessions: order by `updated` alone, and stop treating "select" as
"update".** `loadSessionMetasTx` now orders by `updated DESC, tableId DESC`
with no `active` tiebreak. `setActiveSessionTx` only flips the `active` flag;
it no longer writes `updated`. Only `AppendActiveSessionEntry` (running a
prompt) and `ClearActiveSession` touch `updated`.

**History: reverse only at the frontend-facing boundary.** `loadEntriesTx`'s
chronological order is unchanged, since `ActiveSessionEntries` depends on it.
`loadFrontendHistoryTx` reverses its own copy of that slice before building
`FrontendHistoryItemT`, so the two callers of `loadEntriesTx` get opposite,
independently-correct orders from the same underlying query.

## Consequences

- Switching sessions to look at old history no longer reorders the session
  list or perturbs `updated`.
- The transcript list, and its `slice(0, N)` truncation in the frontend, now
  keep the newest entries instead of the oldest.
- Any new consumer of `loadEntriesTx` must decide which order it needs and
  not assume the function's return order is "the" order - it has two
  independently-correct meanings depending on what's asked of it.

## References

- `ai/agent/sessiondb/session_db.go` - `loadSessionMetasTx`,
  `setActiveSessionTx`, `loadEntriesTx`, `loadFrontendHistoryTx`,
  `ActiveSessionEntries`
- `ai/agent/sessiondb/session_db_ordering_test.go` -
  `TestSetActiveSession_DoesNotChangeOrderingOrUpdatedField`
- `ai/agent/sessiondb/session_db_history_order_test.go` -
  `TestGetFrontendState_HistoryIsNewestFirst`
