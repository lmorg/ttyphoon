# AI Skills

TTYphoon skills are `SKILL.md` files with YAML frontmatter and Markdown instructions. Skills can be invoked from the Skills menu, selected automatically by the AI, or both.

## Locations

TTYphoon searches each XDG config directory for:

- `ttyphoon/agent-skills/<skill-directory>/SKILL.md`
- `ttyphoon/skills/<skill-directory>/SKILL.md`

This supports user-level and system-level XDG configuration directories. Skill directories are discovered one level below `agent-skills/` or `skills/`.

## Frontmatter

The standard Agent Skills fields `name` and `description` are used. TTYphoon also recognizes `license`, `compatibility`, `metadata`, and `allowed-tools`. The following host-specific fields control invocation:

```yaml
---
name: review-changes
description: Review code changes for correctness and regressions. Use when inspecting a diff or reviewing a patch.
user-invocable: true
disable-model-invocation: false
---

# Review changes

Inspect the change and report actionable findings first.
```

`name` is the skill's command ID. TTYphoon does not support a separate command alias: use the same value for `name`, the skill directory, and the slash command. This matches the portable Agent Skills naming convention.

`user-invocable` defaults to `true`. Set it to `false` to hide the skill from the Skills menu and reject its explicit `/name` command. The model may still select it automatically unless automatic invocation is disabled.

`disable-model-invocation` defaults to `false`. Set it to `true` to omit the skill from automatic model selection. It remains available through the menu and `/name` unless `user-invocable` is false.

| `user-invocable` | `disable-model-invocation` | Menu and slash command | Automatic selection |
| --- | --- | --- | --- |
| omitted or `true` | omitted or `false` | Available | Available |
| `false` | omitted or `false` | Hidden | Available |
| omitted or `true` | `true` | Available | Excluded |
| `false` | `true` | Hidden | Excluded |

## Automatic Selection

For ordinary AI requests, TTYphoon asks the configured model to choose the smallest applicable set of skills from their IDs, names, and descriptions. It does not send full skill instructions during this routing call. The model returns a JSON array of skill IDs; TTYphoon accepts only IDs in the available catalog, removes duplicates, and falls back to the normal prompt if selection fails or returns no skills.

TTYphoon then adds selected skill instructions to the main request and starts any MCP servers listed by the skill. The Skills menu remains available; choosing a skill explicitly bypasses automatic selection.

## Supporting Files

An active skill can refer to files beside its `SKILL.md`, such as `references/guide.md` or `scripts/check.sh`. TTYphoon does not inject these files automatically. When the skill instructions refer to one, the model can use `readSkillResource` with the skill's `name` and a path relative to that skill directory. Use `.` or a relative directory path to list immediate contents; pass a file path to read it. Optional line bounds support reading part of a large file.

The tool only exposes skills selected for the current request. It rejects absolute paths, `..` traversal, and symlinks that resolve outside the skill directory. Text reads are limited to 1 MiB; binary resources are not returned by this tool. For a script, inspect its contents first, then use Bash with the absolute path returned by the tool if the skill instructions call for executing it. Bash retains its normal permission behavior. The resource scope is removed when the AI request completes.

## TTYphoon Extensions

`variables` defines fields shown in the skill's input dialog. Values can be referenced in the Markdown body as `$VARIABLE_NAME`.

`allowed-tools` is parsed as a space-separated list. Currently, `mcp(server-name)` starts the named MCP server from TTYphoon's MCP configuration. This field does not restrict the agent's other available tools.

The invocation-control fields are supported by TTYphoon and some other hosts but are not part of the portable Agent Skills core frontmatter specification. Check the target agent's documentation before relying on them elsewhere.
