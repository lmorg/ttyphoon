# 0041 - Rerun failed AI prompts from persisted checkpoints

## Status

Proposed.

Builds on 0003, 0037, 0038 and 0040.

## Context

A transient provider disconnect is now retried automatically (0040), using an
in-memory continuation checkpoint. That helps while the original request is
still alive, but it does not help after the request finally fails, the process
exits, or the user wants to rerun it later.

The failed run does leave useful durable state:

- the session history row stores the original prompt and final error text;
- `stream_prompts` and `stream_blocks` store the request, reasoning, tool calls,
  tool outputs, summaries, sub-agent output, and visible text;
- the current session history UI exposes the stable `entryId` used to delete a
  prompt.

What is missing is a durable, structured **checkpoint contract** and a user-facing
rerun operation. Starting the original prompt again currently creates a fresh
context. It does not tell the model which tools already ran, what they returned,
or which work was completed before the disconnect.

Blindly replaying the original request is unsafe. It can repeat side-effecting
tools such as file writes, issue creation, comments, transitions, or terminal
commands.

## Decision

Add **Rerun from checkpoint** as a new AI request linked to the failed request.
Never mutate or overwrite the failed history row, and never replay its tool calls
automatically.

### 1. Persist a rerunnable checkpoint

Add a per-workspace SQLite table:

```sql
CREATE TABLE IF NOT EXISTS ai_run_checkpoints (
    sessionId INTEGER NOT NULL,
    promptId INTEGER NOT NULL,
    runId INTEGER NOT NULL,
    status TEXT NOT NULL, -- failed | retryable | rerun
    failureKind TEXT NOT NULL DEFAULT '',
    failureMessage TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    created TEXT NOT NULL,
    updated TEXT NOT NULL,
    rerunPromptId INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (sessionId, promptId, runId)
);
```

The checkpoint is written transactionally when a run finalizes with an error.
The existing `stream_blocks` rows are the detailed evidence; `summary` is the
bounded model-facing representation generated from the same
`continuationCheckpoint` used by 0040. Do not copy every stream block into a
second JSON blob.

Store only bounded data in `summary`, using the existing field and total limits.
The original prompt remains in `session_{n}` and is not duplicated here.

### 2. Define retryable failures

Initially mark these retryable:

- transient stream disconnects after automatic retries are exhausted;
- provider/network timeouts, unless the request context itself expired;
- provider 5xx-style transport failures.

Do not offer rerun for:

- user cancellation;
- explicit tool permission denial as the final result;
- successful requests;
- invalid configuration or authentication failures, unless the provider
  reports them as transient and the user explicitly chooses rerun later.

Keep the distinction in the stored `failureKind`; do not infer it from the UI's
error text.

### 3. Add a linked rerun operation

Add a binding such as:

```go
RetryAIHistoryEntry(entryID int64) error
```

The backend must, in one transaction:

1. load the active session entry and its checkpoint;
2. verify the checkpoint is eligible and has not already been rerun, unless an
   explicit "rerun again" action is added later;
3. create a new request using the original prompt, command line, and output
   block;
4. mark the checkpoint as `rerun` and store the new prompt ID in
   `rerunPromptId`;
5. return the new prompt/run context to the UI.

The new request is then executed through the normal AI entrypoint. The original
failed entry remains visible and auditable.

Prefer a new field on the session history row or a small relation table for
`sourcePromptId`/`retryOfPromptId` if the UI needs to show lineage. Do not encode
lineage in prompt text.

### 4. Seed the new run safely

The new run's model context contains:

1. the original user prompt;
2. a bounded system/user checkpoint message:

```text
This is a recovery run for a previous interrupted attempt.
Do not repeat side-effecting actions merely because they appear in the old log.
Inspect the checkpoint below, verify the current workspace state, and continue
only with work that is still necessary.

[checkpoint summary]
```

3. the current workspace/session history according to normal history limits.

The checkpoint summary includes completed observations, tool outputs/summaries,
files changed, commands run, visible assistant output, and the failure reason.
It does **not** include executable tool calls for automatic replay.

The model must verify state before repeating an action. Tool permissions are
fresh for the new invocation; a previous approval is not silently reused.

### 5. Frontend behaviour

Add a `Retry` action to eligible AI Settings history entries. The action is
available only when the backend reports checkpoint eligibility.

The UI should:

- keep the failed entry and its transcript unchanged;
- show the new request as a separate live run;
- disable the action while the rerun is being created;
- refresh history and stream prompt metadata after creation;
- show a clear notification when the checkpoint is unavailable or no longer
  retryable.

Do not make clicking the old prompt itself rerun it; history selection remains
read-only.

### 6. Deletion and cleanup

Deleting a prompt or session must delete its checkpoint transactionally with the
history row and stream blocks, just as 0011/0037 require for stream state.
Deleting the source failed prompt must not delete the linked rerun prompt; the
rerun is an independent audit record. If the source is deleted, retain the
rerun's `sourcePromptId` as an orphaned lineage reference or set it to zero by
explicit policy, but never cascade-delete the rerun.

Clear active history removes all checkpoints in the session.

### 7. Migration and compatibility

There is no useful migration for old failed runs because their structured
checkpoint was never persisted. Existing historical prompts remain readable but
are not rerunnable until a new run produces a checkpoint.

Legacy markdown transcripts are irrelevant to rerun eligibility. The checkpoint
comes from SQLite stream state; do not parse old markdown to reconstruct one.

## Suggested implementation order

1. Add `ai_run_checkpoints` schema and migration.
2. Persist checkpoint/failure classification during finalization.
3. Add transactional load/create/mark-rerun sessiondb methods.
4. Add a runtime entrypoint that accepts a checkpoint seed without replaying
   tools.
5. Add Wails bindings and generated wrappers.
6. Add the Settings history Retry action and lineage display.
7. Add deletion tests, retry safety tests, and a full integration test proving a
   failed run followed by rerun creates two independent prompt records.

## Tests required

- Transient failure after a tool observation persists a bounded checkpoint.
- Non-retryable failure does not expose Retry.
- Retry creates a new prompt and never mutates the failed row.
- Retry cannot execute the same recorded tool call automatically.
- A second retry is rejected unless explicitly supported.
- Deleting the source leaves the rerun intact.
- Deleting the session removes both prompts and checkpoints.
- The checkpoint summary is capped at existing continuation limits.
- Workspace/session scoping prevents retrying another workspace's entry.

## Consequences

- Failed runs become recoverable without pretending tool side effects can be
  rolled back.
- The original failure remains a durable audit trail.
- SQLite stores one bounded checkpoint summary plus the existing block evidence,
  not a second copy of every block.
- Reruns consume a new prompt/history slot and may still fail independently.
- Rerun safety depends on the model verifying state; the system deliberately
  does not attempt automatic tool replay.

## Reference

- `ai/agent/runtime_eino.go` — `continuationCheckpoint`, transient retry
- `ai/agent/sessiondb/session_db.go` — history transactions
- `ai/agent/sessiondb/stream_blocks.go` — durable stream evidence
- `ai/ui.go` — request execution and finalization
- `frontend/src/notes.js` — AI Settings history actions
- 0003 — continuation checkpoints
- 0037 — addressable stream blocks
- 0040 — transient stream retry
