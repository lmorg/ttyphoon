# 84. AI skills can read selected supporting files

## Status

Accepted and implemented on 2026-10-31.

## Context

TTYphoon loaded only each skill's `SKILL.md` body. Skills could mention sibling
references or scripts, but no tool resolved those paths relative to the skill
directory. The general project file reader is scoped to the current project and
must not be broadened to arbitrary XDG configuration directories.

## Decision

- Record the source directory for each loaded skill.
- When assembling a request, grant a request-scoped map of only the selected
  skills' directories. Clear the scope for ordinary prompts and after each run.
- Expose a read-only `readSkillResource` tool that lists a selected skill
  directory or reads a relative UTF-8 text file, optionally by line range.
- Resolve symlinks and reject any path that escapes the selected skill root.
  Limit each file read to 1 MiB and reject binary files.
- Do not execute scripts in the resource reader. The agent may use the existing
  Bash tool with the returned absolute path when the skill instructs it to do so;
  Bash permissions remain unchanged.

## Consequences

- Skill references can be loaded on demand without exposing unselected skills
  or granting general access to XDG configuration directories.
- Text references and scripts can be inspected. Binary assets are not returned
  by this tool.
- Skill-resource access follows the active request's context and disappears
  after that request completes.

## Reference

- `ai/skills/skills.go` - skill source-directory tracking
- `ai/agent/skill_resources.go` - request-scoped resource directory context
- `ai/tools/skillresource/read.go` - scoped listing and text reader
- `ai/prompts/system.md` - model guidance
- `docs/AI-Skills.md` - user-facing behavior and limits