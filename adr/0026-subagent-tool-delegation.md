# 0026 — Sub-agent tool delegation

## Status

Accepted. Supersedes the unimplemented delegation hook described in ADR 0006.

## Context

`subagents.Subagent.Call` asserted its agent to a `delegateToolRunner` and, on
success, handed `RunSubagentWithTools` to `subagent.Run`:

```go
if runner, ok := t.agent.(delegateToolRunner); ok {
    subagentRequest.RunWithTools = runner.RunSubagentWithTools
}
```

**Nothing implemented that method.** The assertion always failed, `RunWithTools`
stayed nil, and `subagent.Run` took its other branch — a single `chatModel.Stream`
call with no tools bound. Every sub-agent was a bare LLM turn.

Everything around it was already built and inert:

- `SubagentToolNames()` computed the delegable set and `Description()`
  **advertised it to the model** — sub-agents were told they could use tools they
  could never call.
- A whole permission scope (`subagentTools`, `SetToolAllowedInSubagent`,
  `DefaultPermissions.Subagents` on ~15 tools) plus a user-facing "Allow in
  subagents" toggle configured a policy nothing consulted at run time.

The failure mode was the worst kind: a sub-agent asked to read a file could not,
so it answered from the prompt alone — plausible, ungrounded, and silent.

## Decision

Implement `(*Agent).RunSubagentWithTools`, building a second `einoRuntime`
restricted to the delegable toolset.

`einoRuntime` already supported this: `toolsConfig()` honours a non-nil
`r.tools` field and otherwise falls back to the agent's full set. Only the
caller was missing.

```go
runtime := &einoRuntime{agent: agent, tools: agent.subagentToolSet()}
```

Design points:

- **Isolated context.** Unlike `runLLMWithMessageStream`, the sub-agent path does
  not load `sessiondb` history. A sub-agent sees only its system prompt and task.
- **The system prompt is now passed.** The old signature was
  `RunSubagentWithTools(ctx, prompt, emit)`, so `Request.SystemPrompt` — the
  whole point of the `delegate` vs `report` distinction — would have been
  dropped. Widened to `(ctx, systemPrompt, prompt, emit)` across the interface,
  the `RunWithTools` field, and the call site.
- **Tool progress routes to the sub-agent's emitter.** `withAIStreamCallback`
  binds the run's emitter into the context, so `einoAgentTool.InvokableRun`
  writes tool calls into the sub-agent's own buffered block rather than
  interleaving into the parent stream (preserving ADR 0006).
- **No permission stalls.** Delegable tools are `ToolStateAlways` by
  construction (ADR 0002 update), and `RequestToolPermission` returns nil
  immediately for that state, so a sub-agent can never block on a prompt no one
  can answer.
- **Partial work survives the step limit.** A max-step error with content already
  accumulated returns that content instead of discarding the run.
- **Toolless fallback retained.** If the user disables every sub-agent tool,
  `Call` leaves `RunWithTools` nil rather than failing, matching the previous
  behaviour. The runner's own no-tools error is then only a defensive guard.

`SubagentToolNames` and `subagentToolSet` now share
`toolDelegableToSubagentLocked`, so the advertised list and the bound list cannot
drift.

## Consequences

- Sub-agents can genuinely read files, grep, and run the other always-allowed
  tools; the configured permissions finally take effect.
- Each delegation constructs its own chat model and react agent. Acceptable —
  every sub-agent is a separate conversation — but it is not free per call.
- Sub-agent runs get no max-step continuation, unlike the main agent (ADR 0003).

## Lessons

- **An unimplemented interface assertion fails silently and forever.**
  `x.(iface)` returning false is indistinguishable from "feature off". The
  compile-time guard is one line and belongs at the assertion site:

  ```go
  var _ delegateToolRunner = (*agent.Agent)(nil)
  ```

- **Advertising a capability in a tool description is a promise to the model.**
  `Description()` listed tools for two release cycles that no sub-agent could
  invoke, which shapes planning and produces confident, ungrounded output.
- **Check what a callback signature drops.** `RunWithTools(ctx, prompt, emit)`
  compiled perfectly while silently discarding the system prompt that
  distinguishes `delegate` from `report`.

## Reference

- `ai/agent/subagent_runner.go` — `RunSubagentWithTools`, `subagentToolSet`,
  `runSubagentStream`
- `ai/agent/runtime_eino.go` — `einoRuntime.tools`, `toolsConfig`
- `ai/subagent/subagent.go` — `Request.RunWithTools`
- `ai/tools/subagents/subagent.go` — `delegateToolRunner`, `subagentToolNames`
- Tests: `TestSubagentToolSet_MatchesSubagentToolNames`,
  `TestRunSubagentWithTools_ErrorsWhenNoToolsDelegable`, and the
  `var _ delegateToolRunner` assertion
- ADR 0002 — sub-agent tool eligibility · ADR 0006 — output buffering
