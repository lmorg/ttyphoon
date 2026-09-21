# 1. Tool failures must not abort the agent run

Date: 2026-09-06

## Status

Accepted

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

- `context.Canceled` — the user pressed stop
- `context.DeadlineExceeded` — the request timeout elapsed
- runtime/init failures that mean no further progress is possible

Recoverable (return `(message, nil)`):

- MCP protocol and schema validation errors
- tool permission denials
- disabled or denied tool state
- malformed JSON arguments

To distinguish them, `ai/agent/agent.go` defines a sentinel:

```go
var ErrToolPermissionRefused = errors.New("tool call refused")
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
- **Care required:** any new `return "", err` added to the tool call path
  re-introduces this class of bug. Classify the error first.
- Notifications still surface the failure in the UI, so nothing becomes silent.

## Reference

- `ai/agent/runtime_eino.go` — `einoAgentTool.InvokableRun`
- `ai/agent/mcp_tool.go` — `mcpTool.Call`
- `ai/agent/agent.go` — `ErrToolPermissionRefused`
- Tests: `TestEinoAgentTool_InvokableRun_DeniedToolDoesNotAbortRun`,
  `TestEinoAgentTool_InvokableRun_CancellationStaysTerminal`
