# 83. AI skills support independent manual and model invocation

## Status

Accepted and implemented on 2026-10-30.

## Context

TTYphoon discovers `SKILL.md` files and supports both a Skills menu and automatic
selection by a model. Skills need a way to opt out of either invocation path
without losing the other. The frontmatter fields `user-invocable` and
`disable-model-invocation` are supported by several agent hosts, but are not
part of the core Agent Skills frontmatter specification.

## Decision

- Treat omitted `user-invocable` as `true`; when false, hide the skill from the
  Skills menu and reject its explicit slash command.
- Treat omitted `disable-model-invocation` as `false`; when true, omit the skill
  from the model's automatic selection catalog. It remains available for manual
  invocation unless `user-invocable` is also false.
- Use standard `name` as the sole skill command ID. Do not support separate
  `function-name` or `metadata.function-name` aliases.
- Preserve the menu as an explicit path. Automatic selection considers only
  model-invocable skills, and the shared prompt builder activates selected MCP
  entries before the primary agent run.
- Keep `allowed-tools` behavior as implemented: it starts referenced MCP
  servers; it does not restrict the agent's other available tools.

## Consequences

- A skill may be menu-only, model-only, available through both paths, or
  unavailable through either path.
- Skill names, slash commands, and model-routing IDs cannot diverge. Existing
  skill files that relied on `function-name` must move that command ID into
  `name` (and use it as the skill directory name).
- TTYphoon supports useful host extensions while keeping core skill fields
  (`name`, `description`, `license`, `compatibility`, `metadata`, and
  `allowed-tools`) compatible with the Agent Skills format.

## Reference

- `ai/skills/skills.go` - frontmatter parsing and invocation filters
- `ai/prompts/prompts.go` - automatic routing and explicit skill selection
- `window/backend/renderer_webkit/ai.go` - Skills menu filtering
- `docs/AI-Skills.md` - user-facing file format and behavior