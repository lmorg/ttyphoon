# 1. Tool failures must not abort the agent run

Date: 2026-09-06

## Status

Accepted

Updated 2026-10-08: invocation-error recovery is centralized for structured
native tools and string-input adapters.

## Context

The Eino ReAct runtime treats a non-nil `error` returned from
`einoAgentTool.InvokableRun` as a **node failure**. That fails the whole graph
run, which propagates out of `RunLLMWithMessageStream` and is rendered by
`askAI` in `ai/ui.go` as a terminal error notification.

The LLM never sees the failure and gets no opportunity to correct itself.

Three separate bug reports in one session traced back to this single cause:

1. An MCP server rejected arguments with
   `invalid params: validating "arguments": unexpected additional properties ["limit"]`.
   A recoverable schema mistake killed the entire run.
2. Clicking **Deny** on an in-panel tool permission prompt cancelled the agent
   instead of telling the model it was refused.
3. A tool in `disabled` or `denied` state aborted the run rather than being
   skipped.

In every case the correct outcome is the same: the model should be told what
happened and allowed to continue.

## Decision

Recoverable tool failures are returned as **tool output with a `nil` error**.
Only genuinely terminal conditions return a non-nil error.

Terminal (still return an error):

- The agent invocation context's `ctx.Err()` is `context.Canceled` — the user
   pressed stop
- The agent invocation context's `ctx.Err()` is `context.DeadlineExceeded` —
   the request timeout elapsed
- runtime/init failures that mean no further progress is possible

Recoverable (return `(message, nil)`):

- MCP protocol and schema validation errors
- tool permission denials
- disabled or denied tool state
- malformed JSON arguments
- ordinary errors returned from native `CallStructured` or string-input `Call`,
  including input validation, filesystem/network failures, and tool-local
  timeouts or cancellation while the agent invocation context remains live

The Eino adapter is the recovery boundary. Concrete tools may return normal
Go errors without swallowing them or guessing whether Eino will abort. After
recording the original failure in the typed stream and continuation observation,
`InvokableRun` returns model-visible failure text with a nil framework error.
The text includes the tool name, original error, and any partial output, and
guides the model to correct its input or choose another tool. Since a failed
operation may have partially succeeded, it also tells the model to check the
existing state before retrying. No automatic blind retry is performed.

Check the invocation context before dispatch and after the call; a delegate
that ignores cancellation and returns success cannot keep a stopped run alive.
A returned `context.DeadlineExceeded` alone is not proof that the agent timed
out: with a live invocation context it can represent a recoverable tool-local
timeout.

To distinguish them, `ai/agent/agent.go` defines a sentinel:

```go
package main

import "errors"

func main() {
	var ErrToolPermissionRefused = errors.New("tool call refused")

}
```

`InvokableRun` converts anything matching `errors.Is(err, ErrToolPermissionRefused)`
into model-visible text. `mcpTool.Call` performs the equivalent check against
`ctx.Err()` before swallowing an error.

The returned message should tell the model what to do next, not merely state the
failure — e.g. *"Do not retry this tool call; continue the task without it, or
tell the user what you need."*

## Consequences

- A refused or malfunctioning tool degrades the run instead of ending it.
- Cancellation semantics are preserved; the stop button still works.
- Tool implementations can return ordinary errors. New adapter/framework
   paths must preserve this classification rather than forwarding invocation
   errors into the graph as terminal node failures.
- Notifications still surface the failure in the UI, so nothing becomes silent.
- Network-free integration testing exercises the actual Eino ReAct loop: a
   scripted model receives a failing tool's observation, issues corrected
   arguments, receives success, and produces its final answer.

## Reference

- `ai/agent/runtime_eino.go` — `einoAgentTool.InvokableRun`
- `ai/agent/mcp_tool.go` — `mcpTool.Call`
- `ai/agent/agent.go` — `ErrToolPermissionRefused`
- Tests: `TestEinoAgentTool_InvokableRun_DeniedToolDoesNotAbortRun`,
  `TestEinoAgentTool_InvokableRun_CancellationStaysTerminal`
- Tests: `TestEinoReActModelReceivesToolFailureAndRetries`,
  `TestStructuredToolInvocationErrorIsModelVisibleAndRetryable`,
  `TestStructuredToolLocalTimeoutDoesNotCancelAgent`,
  `TestStructuredToolExpiredAgentDeadlineStaysTerminal`
