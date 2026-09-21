# 0025 — Sub-agent tool robustness and unique notification IDs

## Status

Accepted.

## Context

An audit of `subagents.Subagent.Call` — the tool that fans out `delegate` and
`report` sub-agents — found four defects, two of which are reachable directly
from model-generated input.

### Nil request elements panic the backend

`Call` decodes into `[]*requestT`. A JSON `null` array element decodes to a **nil
pointer**, and the first field access dereferences it:

```
input: [null, {"name":"a","prompt":"b"}]
panic: runtime error: invalid memory address or nil pointer dereference
```

Tool arguments come straight from an LLM, so `[null]` is not a hypothetical.
Every other malformed-input case was already handled by returning a message; this
one crashed the process.

### Notification IDs collided

`DisplaySticky` minted IDs from `time.Now().UnixMilli()`. `Call` raises one
sticky per request in a tight loop, so all of them landed in the same
millisecond. Measured: **8 stickies produced 1 unique ID.**

The ID is the frontend's key and the handle for `CloseNotification`, so with
parallel sub-agents:

- the "Running subagent: X" notifications collapsed into a single entry;
- the first sub-agent to finish emitted `terminalNotificationClose` for the
  shared ID, dismissing the notification while others were still running.

### delete() matched on ID, not identity

```go
if (*notifications)[i].id == nt.id {
```

With duplicate IDs this evicted whichever notification came first in the slice
rather than the one being closed, leaking the others permanently. Confirmed by
test: `delete removed the wrong notification`.

### Redundant close and a fatal marshal error

`Call` had both `defer sticky.Close()` and a trailing `sticky.Close()`. `Close`
is idempotent, so this was harmless but misleading. Separately `Call` ended with
`return resp.json()`, propagating any marshal error as a non-nil tool error —
which aborts the whole agent run, contrary to ADR 0001.

## Decision

- Skip nil request elements, recording the same validation message used for blank
  names and prompts.
- Hoist trimming and validation out of the goroutine. Invalid entries no longer
  raise a sticky at all, and the sticky label is trimmed.
- Mint notification IDs from a strictly increasing atomic
  (`newNotificationID`), seeded from the wall clock so IDs stay time-ordered and
  the existing frontend contract is unchanged.
- Compare pointers in `notifyT.delete`, which is correct regardless of ID
  uniqueness.
- Drop the redundant `sticky.Close()`, and return marshal failures as a tool
  message rather than an error.

## Consequences

- Parallel sub-agents each get their own sticky that closes independently.
- Malformed tool input degrades to a message; no input shape panics.
- `Call`'s validation paths are now unit-testable without a renderer: invalid
  requests never reach `t.agent`.

## Lessons

- **Wall-clock timestamps are not identifiers.** Any loop that creates more than
  one item per millisecond will collide. If a value is used as a key, mint it
  from a counter.
- **Match on identity, not on a value that is supposed to be unique.** The
  ID-based `delete` was only correct while the ID invariant held, and it failed
  silently when that invariant broke.
- **Decoding into a slice of pointers admits `null`.** `[]*T` from untrusted JSON
  needs a nil check per element; `[]T` would not.
- **Prove the test fails first.** Removing the nil guard reproduced the panic and
  removing the ID fix reproduced `got 1 unique notification IDs for 8 stickies` —
  both tests pin real defects rather than passing incidentally.

## Reference

- `ai/tools/subagents/subagent.go` — `Call`
- `ai/tools/subagents/subagent_test.go` — malformed input, per-request errors,
  concurrent `responsesT` store
- `window/backend/renderer_webkit/notifications.go` — `newNotificationID`,
  `DisplaySticky`, `notifyT.delete`
- `window/backend/renderer_webkit/notifications_test.go`
- ADR 0001 — tool failures must not abort the agent run
- ADR 0006 — parallel sub-agent output buffering
