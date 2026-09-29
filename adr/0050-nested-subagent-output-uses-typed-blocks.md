# 0050 - Nested subagent output uses typed stream blocks

Date: 2026-09-29

## Status

Accepted.

## Context

The outer subagent tool opened a typed `subagent` block, but
`runSubagentStream` replaced the run's callback with a callback-only emitter.
Nested tool calls, outputs, errors and summaries consequently fell back to
plain text and were appended to the subagent response. Tool progress notices
also used the writer's root-level notice block. This bypassed the typed block
hierarchy and the panel's tool-output disclosure behavior.

## Decision

- Reuse the parent run's typed writer in the nested runtime; do not create a
  second writer with colliding block IDs or independent ordering.
- Scope the nested runtime's block parent to its owning `subagent` block.
- Keep assistant response text in that subagent block, and emit nested tool
  calls, outputs, errors, summaries and progress notices as typed children of
  their owning tool-call block.
- Preserve callback-only behavior when no typed writer is available.

## Consequences

- Nested tool activity is addressable, persisted, and rendered with the same
  block semantics as top-level tool activity.
- Tool output and summaries inside subagents participate in the panel's
  collapsed-by-default presentation.
- Each subagent continues to have an independent visible response block while
  sharing the run's block identity and ordering source.
- Previously persisted subagent blocks that flattened nested tool activity do
  not contain enough boundaries to reconstruct typed tool blocks reliably.
  New runs use the typed hierarchy; no lossy migration is attempted.

## Reference

- `ai/agent/subagent_runner.go` - nested emitter construction and response text
- `ai/agent/runtime_eino.go` - child emitter and notice parenting
- `ai/tools/subagents/subagent.go` - subagent block context
- `ai/agent/stream_writer_test.go` - persisted nested block hierarchy
- `adr/0026-subagent-tool-delegation.md` - subagent tool execution