# 3. Resume the agent after max-step exhaustion

Date: 2026-09-06

## Status

Accepted. Transient model stream disconnect recovery is specified separately in
ADR 0040 and reuses this record's checkpoint continuation mechanism.

## Context

Eino's ReAct agent is constructed with `MaxStep: r.agent.MaxIterations()`, sourced
from `config.Config.Ai.MaxIterations` (default 10). On exhaustion it returns:

```
[GraphRunError] exceeds max steps
```

This was surfaced to the user as a hard failure. Any work already performed was
lost from the model's perspective, even though tool side effects (files written,
commands run) had persisted to the workspace.

Simply raising `MaxIterations` is not a fix: it trades one arbitrary cliff for a
larger one, and a bigger budget increases the risk of exceeding the model's
context window.

Note that "max steps" is a **harness** limit, not a provider one. See ADR 0011.

## Decision

Wrap the run in a bounded continuation loop.

- `RunLLMWithMessageStream` (outer) drives continuations.
- `runLLMWithMessageStream` (inner) performs a single bounded Eino run.
- Only `exceeds max steps` triggers a continuation; all other errors return
  immediately.
- The number of retries is capped by `config.Config.Ai.MaxContinuations` before
  asking the user, through the same `ttyphoon://ai-user-question` hyperlink
  flow used for other interactive questions, whether to **continue** or **finish
  up**.
- **Continue** resets the continuation counter and opens another bounded window.
- **Finish up** emits the exact continuation summary that would have been sent
  to the model into the AI panel, then completes the run without returning the
  max-step error.
- The overall request timeout (ADR 0004) spans **all** windows, providing a hard
  upper bound on wall-clock time.

Each continuation appends a message carrying the previous window's output and
instructing the model not to repeat completed actions, and to verify existing
work before proceeding. Progress is surfaced in the panel as
`**Continuing after max steps (n/m)**`.

## Consequences

- Long tasks complete instead of failing at an arbitrary step boundary.
- Users can explicitly extend or conclude a long task instead of hitting a
  silent hard stop at the configured continuation count.
- Continuations inherit prompt-scoped tool permissions (ADR 0002), so the user is
  not re-prompted for consent they already gave in this run.
- Detection is **string matching** on the error text. This is fragile: an Eino
  upgrade that rewords the error will silently disable continuation. If Eino ever
  exports a typed error, switch to `errors.Is`.
- Session history is written only after the run completes, so the continuation
  prompt relies on the accumulated in-memory response rather than the session DB.
  A checkpoint-before-exhaustion mechanism would be more robust if this proves
  insufficient.

## Amendment (2026-09-16): the outer loop had no stream callback, so the question hung silently

Two bugs shipped together and compounded each other:

1. **Off-by-one on the continuation count.** The boundary check was
   `if continuation >= MaxContinuations`, checked using the *current*
   (0-indexed) loop variable after that continuation had already run. With
   `MaxContinuations: 3` this let a 4th continuation run (visible in logs as
   `Continuation 4 of 3`) before asking the user - one full extra continuation
   window past the configured limit. Fixed by checking the *next* 1-indexed
   continuation number instead: `if nextContinuation >= MaxContinuations`.

2. **The outer loop's `ctx` never carried the stream emitter.** Only
   `runLLMWithMessageStream` (the inner, per-continuation call) did
   `ctx = withAIStreamCallback(ctx, emitter)` - and that reassignment is local
   to that function call, discarded the moment it returns. The outer loop in
   `RunLLMWithMessageStream` kept using its own, never-enriched `ctx` for both
   the `**Continuing after max steps (n/m)**` progress marker and, critically,
   `RequestUserQuestion`. `emitAIStreamToolProgress` silently no-ops when the
   context has no emitter (by design, so it's safe to call from contexts that
   never stream), so the continue/finish-up question was built but never sent
   to the panel - the user had nothing to click, and `RequestUserQuestion`
   blocked on its answer channel forever. Combined with bug 1, the visible
   symptom was: agent quietly grinds past the configured continuation limit,
   then hangs with no visible prompt.

   Fixed by attaching a stream callback to the outer `ctx` once, at the top of
   `RunLLMWithMessageStream`, before the loop starts.

## Amendment (2026-09-17): continuation state is visible-output only, not a structured checkpoint

Reviewing the continuation flow showed that it preserves less state than the
phrase "continuation summary" suggests:

- `RunLLMWithMessageStream` keeps an in-memory `continuationMessages` slice for
  the current prompt.
- Each exhausted bounded window returns only the assistant content accumulated
  from the outer stream. Reasoning content is forwarded to the panel/log emitter
  but is not appended to the returned string.
- On max-step exhaustion, the next window receives a new `schema.UserMessage`
  containing the fixed instruction "Continue the original task..." plus the
  previous window's returned visible output.
- Every bounded window also reloads prior completed session history from
  SQLite via `sessiondb.ActiveSessionEntries`, but the current in-flight prompt
  is not written to SQLite until the entire run completes in `ai/ui.go`.

Before the 2026-09-17 structured-checkpoint work, continuations preserved the
original prompt, prior completed session history, prompt-scoped tool
permissions, and visible assistant output from the current run. They did **not**
reliably preserve the ReAct scratchpad, structured assistant tool calls, tool
results/observations, or reasoning content. If a window spent most of its budget
exploring via tools and emitted little final assistant content before exhausting
`MaxStep`, the next continuation might not know enough about what was already
done and could repeat directory reads, file reads, or grep searches.

This is acceptable as a minimal recovery loop, but it is not a best-practice
agent harness for long-running work. The next design should add a per-window
structured checkpoint that captures:

- assistant tool calls and arguments;
- tool results/observations, possibly truncated or summarized;
- assistant visible content;
- a compact task-state summary, such as completed work, pending work, files
  touched/read, facts learned, and verification already performed.

The next continuation should be seeded from that structured checkpoint rather
than only from the previous visible output. Prior completed session history
should remain separate from the current in-flight checkpoint; using SQLite
session history for the current prompt would require checkpointing before the
run completes and careful handling of cancellation/partial output.

Implementation should also introduce a test seam around the continuation loop.
Today `RunLLMWithMessageStream` calls the concrete private
`runLLMWithMessageStream`, which makes max-step continuation behavior hard to
unit-test without a live Eino agent. Extracting a small bounded-window runner
interface/function would allow tests to prove: max-step errors append the right
checkpoint, the user prompt appears exactly at the configured boundary,
"finish up" returns cleanly, and "continue" resets only the continuation
window counter.

### Implementation plan

1. Extract a small seam for "run one bounded window" so the continuation loop
  can be tested without constructing a live Eino ReAct agent.
2. Add focused unit tests for the existing control-flow contract: max-step
  errors append continuation state, non-max-step errors return immediately,
  the user question appears exactly at the configured continuation boundary,
  `finish up` returns cleanly, and `continue` resets only the continuation
  window counter.
3. Introduce an in-memory per-window checkpoint type. Initially it should
  capture assistant visible output and whatever structured events the current
  runtime can reliably observe without changing provider behavior.
4. Extend runtime/tool execution instrumentation so the checkpoint can include
  assistant tool calls, tool arguments, tool results/observations, and compact
  facts such as files read, searches run, files modified, commands run, and
  verification already performed.
5. Replace the current raw `Previous run output` continuation payload with a
  compact, structured state message derived from the checkpoint: completed
  work, pending work, facts learned, tool observations, and output tail.
6. Keep prior completed session history separate from the current in-flight
  checkpoint. Do not rely on `ActiveSessionEntries` for current-prompt
  recovery unless partial-run persistence is explicitly introduced.
7. Consider SQLite-backed partial-run checkpointing only after the in-memory
  harness is testable and stable; persistence needs cancellation/crash
  semantics and should be a separate change.

### Implementation status (2026-09-17)

The first pass of the structured continuation harness is implemented:

- `einoRuntime` has a `boundedWindowRunner` seam so the outer continuation loop
  can be unit-tested without a live Eino ReAct agent.
- Focused tests cover appending continuation state, prompting at the configured
  boundary, `finish up`, `continue` resetting the bounded continuation window,
  and real `einoAgentTool.InvokableRun` observation capture.
- Each bounded window gets an in-memory `continuationCheckpoint` attached to
  the context. It currently records visible assistant output and generic tool
  observations (`Name`, `Arguments`, `Output`, `Error`), with each field capped
  at 4000 characters.
- Tool observations are captured at the Eino tool wrapper boundary, after
  permission failures, delegate tool errors, normal tool returns, and summarized
  oversized tool returns.
- The next continuation is now seeded by
  `formatContinuationCheckpointMessage(checkpoint)`, which includes the fixed
  continuation instruction plus structured sections for tool observations and
  visible assistant output.

The second pass adds the semantic observation contract and checkpoint compactor:

- `aitypes.ToolObservationProvider` lets native tools return structured
  metadata for continuation checkpoints without changing the normal `Call`
  output the model sees in the current bounded window.
- `aitypes.ToolObservation` includes semantic fields for inputs, outputs,
  files read, files modified, directories listed, searches run, commands run,
  counts, status, summary, and error.
- `einoAgentTool.InvokableRun` prefers semantic observations when a tool
  implements `ToolObservationProvider`, and falls back to generic observations
  for MCP and other tools that do not.
- Semantic observation providers are implemented for `readFiles`,
  `readDirectory`, `grep`, `writeFile`, `patchFile`, `insertLines`,
  `generateImage`, and `commandLine`.
- The continuation formatter emits semantic sections for files, directories,
  searches, commands, counts, errors, outputs, and visible assistant output.
- Each field is capped at 4000 characters, and the total continuation
  checkpoint payload is capped at 20000 characters with an explicit truncation
  marker.

This is intentionally still an in-memory checkpoint for the current prompt, not
SQLite partial-run persistence. Remaining work is to add MCP-specific adapters
only where generic server/tool/arguments/output metadata is insufficient, and,
only if needed, add crash/cancellation recovery via persisted partial-run
checkpoints.

## Reference

- `ai/agent/runtime_eino.go` — `RunLLMWithMessageStream`, `isMaxStepError`
- `ai/agent/aitypes/aitypes.go` — `ToolObservationProvider`,
  `ToolObservation`
- `ai/tools/file/*.go`, `ai/tools/image/generate.go`,
  `ai/tools/shell/cmdline.go` — native tool observation providers
- `ai/ui.go` — final `AppendActiveSessionEntry` call after the full run returns
- `ai/agent/sessiondb/session_db.go` — `ActiveSessionEntries`, `loadEntriesTx`
- `config/defaults.yaml` — `AI.MaxIterations`, `AI.MaxContinuations`

