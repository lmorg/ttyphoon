# 12. System prompts state their context and constraints

Date: 2026-09-06

## Status

Accepted

## Context

Prompts under `ai/prompts/` are embedded markdown expanded via `os.Expand`, with
`$SYSTEM_PROMPT` and `$USER_PROMPT` shared by all of them. `system.md` carries
global rules; each entry point adds its own file.

Reviewing `ask.md` — the general-purpose entry point — exposed a structural gap.
`explain_cmd.md` and `explain_doc.md` both inject context automatically
(`$COMMAND_LINE`, `$COMMAND_OUTPUT`), so the model always has material to work
from. `ask.md` receives **no** automatic context, yet said nothing about it. A
model reading it cold had no signal that it must use tools to ground itself, nor
that there would be no follow-up turn.

## Decision

Each prompt states what its situation actually is, and global rules stay in
`system.md` rather than being duplicated.

For `ask.md` specifically:

- state that no context is supplied automatically, and that tools should be used
  rather than guessing;
- state that the response is one-shot, so ambiguity should be resolved by stating
  an interpretation and answering — not by asking and stopping;
- use consistent terminology with sibling prompts ("query", not "question");
- avoid non-actionable filler such as "helpful".

The one-shot instruction reflects a real property of the harness: `askAI` runs the
loop once and returns, so a clarifying question ends the turn with no way for the
user to reply. Where genuine clarification is unavoidable, the `askUser` tool
(ADR 0008) provides a blocking round trip instead.

Cross-cutting behaviour belongs in `system.md`, including the
`[output summarised by subagent]` marker and guidance on `delegate` / `report`.

## Consequences

- The general-purpose prompt no longer implies context it does not receive.
- Global rules have one home; per-entry-point files stay small.
- Prompt files are behaviour. Changing them alters agent conduct without any
  compiler or test signal, so treat edits with the same care as code.

## Reference

- `ai/prompts/ask.md`, `system.md`, `explain_cmd.md`, `explain_doc.md`
- `ai/prompts/prompts.go` — `os.Expand` variable substitution
