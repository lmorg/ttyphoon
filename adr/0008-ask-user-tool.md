# 8. `askUser` tool for blocking clarification

Date: 2026-09-06

## Status

Accepted

## Context

Ttyphoon's AI entry points are one-shot: `askAI` runs the ReAct loop once and
returns. There is no second conversational turn, so an agent that needs a missing
fact has no way to obtain it and must guess.

A general "chat with the user" facility was rejected as it invites the model to
defer work rather than investigate. The need is narrower: unblock on a specific
missing input.

## Decision

Add an `askUser` tool in its own package, `ai/tools/userquestion`, following the
standard tool layout (registration via `init()` → `agent.ToolsAdd`, embedded
`description.md`, `DefaultPermissions`).

Input schema:

```json
{
  "question": "Which environment should I deploy to?",
  "choices": ["prod", "staging", "dev"]
}
```

`choices` is optional. The question and its options are rendered in the panel
using the `ttyphoon://ai-user-question` scheme described in ADR 0007, and the
answer is returned to the model as the tool result.

The tool description constrains usage to cases where the agent is genuinely
blocked and cannot proceed safely — not for routine uncertainty.

## Consequences

- The agent can obtain a required fact instead of guessing.
- The call blocks a goroutine until answered, cancelled, or timed out.
- Answers are transient: they are not persisted to tool settings or session state.
- **Risk:** an over-eager model could use this instead of investigating with
  read-only tools. If that emerges, tighten the description or set the default
  invocation permission to `askPermission`.

## Reference

- `ai/tools/userquestion/userquestion.go`
- `ai/tools/userquestion/description.md`
- `ai/agent/user_question.go` — `RequestUserQuestion`, `ResolveUserQuestionRequest`
- ADR 0007 for the link/round-trip mechanism
