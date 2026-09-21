# 2. Tool permission scopes are per-invocation or per-prompt

Date: 2026-09-06

## Status

Accepted

Supersedes the earlier three-link model (`prompt` / `session` / `deny`).

## Context

The AI panel renders consent links when a tool in `approval` state is invoked.
The original model offered three choices, which had two problems:

- **Asymmetric consequences.** Both allow options continued the run, but *Deny*
  aborted the agent entirely (see ADR 0001).
- **Session scope leaked.** `toolPermissionAllowedSession` survived
  `ResetToolPermissions`, so a decision made for one user prompt silently
  applied to later, unrelated prompts.

There was also no way to allow a single invocation without granting the tool for
everything that followed.

## Decision

Four links, spanning two independent axes (allow/deny × invocation/prompt):

| Link | `decision` value | Stored | Effect |
|---|---|---|---|
| Allow this invocation | `allow-once` | no | allows now, re-prompts next time |
| Allow for this prompt | `allow-prompt` | yes | allows for the remainder of the run |
| Deny this invocation | `deny-once` | no | denies now, re-prompts next time |
| Deny for this prompt | `deny-prompt` | yes | denies for the remainder of the run |

Rules:

1. **Session scope is removed.** `toolPermissionAllowedSession` no longer exists.
2. **"Prompt" means the whole agent run, including continuations** (ADR 0003),
   but excluding new user-generated prompts.
3. **Consent never mutates the persistent tool settings.** These decisions write
   only to `agt.toolPermissions`. The `toolStates` map behind the `A`/`S`/`P`/`D`
   badges in AI settings is untouched.
4. **Denials do not abort the run** — they return via `ErrToolPermissionRefused`.

`ResetToolPermissions()` clears the entire map and is called **once per user
prompt**, from the outer `RunLLMWithMessageStream`. It must not be called from
the inner per-continuation function, or prompt scope would be lost on every
continuation.

## Consequences

- Users can grant narrow, single-use access.
- Decisions cannot leak across user prompts.
- **Implementation trap:** once-decisions are deliberately not persisted, so the
  resolver must `return` directly after handling a decision. The previous code
  looped back to re-read the stored decision; with nothing stored that becomes an
  infinite re-prompt loop. Only *waiters* blocked on a concurrent request for the
  same tool loop back — which is correct, as they must prompt for their own
  invocation.
- The frontend needs no per-decision knowledge: `markdown-utils.js` forwards the
  `decision` query parameter verbatim, so new values work without JS changes.

## Update — sub-agent tool eligibility

Sub-agents have no user to prompt. A tool left at `approval` (or `session`,
`prompt`) would either stall or silently fail if delegated, so `Subagents:
"allow"` alone is not sufficient to qualify.

`SubagentToolNames()` now requires **both**:

1. allowed in sub-agents (per-tool default, or the user's override), and
2. an effective state of exactly `ToolStateAlways`.

Anything that can ask for permission is therefore excluded, including a tool the
user has set to `session` — that grant came from an interactive prompt and does
not extend to a non-interactive context.

Three tools are additionally hard-blocked and cannot be enabled at all:

| Tool | Reason |
|---|---|
| `askUser` | Sub-agents cannot surface a question to the user |
| `delegate` | Prevents recursive delegation |
| `report` | Prevents recursive delegation |

`SetToolAllowedInSubagent` rejects attempts to enable these, and
`ShowToolSubagentMenu` now reports that refusal instead of discarding the error
and appearing to do nothing.

Note this list is what `Description()` advertises to the model, so tightening it
also stops the agent planning around tools a sub-agent could never run.

## Reference

- `ai/agent/agent.go` — `RequestToolPermission`, `formatToolPermissionRequestMarkdown`,
  `ResetToolPermissions`
- Tests: `TestRequestToolPermission_DecisionScopes`,
  `TestResetToolPermissions_ClearsPromptScopedDecisions`
- `ai/agent/tools.go` — `SubagentToolNames`, `toolAllowedInSubagentLocked`,
  `SetToolAllowedInSubagent`, `subagentForbiddenTools`
- Tests: `TestSubagentToolNames_OnlyIncludesAlwaysAllowedTools`,
  `TestSubagentForbiddenToolsCannotBeEnabled`
