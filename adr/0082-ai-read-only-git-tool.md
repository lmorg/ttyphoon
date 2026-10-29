# 82. AI Git inspection uses a permission-gated read-only action whitelist

## Status

Accepted and implemented on 2026-10-29.

## Context

The AI agent can inspect files and run Bash, but common Git questions benefit
from an explicit tool that exposes repository-aware operations without granting
arbitrary command execution. Git action policy belongs in `utils/git` (ADR 0071),
and ClonePane provides the interactive terminal execution path.

## Decision

- Register an internal structured AI tool named `git` with actions `status`,
  `diff`, `log`, `show`, `branches`, and `remotes`.
- The tool is enabled by default, with invocation permission `askPermission` and
  subagent permission `deny`.
- Execute every action through the Git package's interactive runner and
  ClonePane. Do not accept arbitrary Git flags or shell commands.
- Validate revisions, repository-relative paths, and log limits in `utils/git`.
  Restrict commands to read-only Git subcommands and bound returned output.
- Use only remote names for the `remotes` action; do not expose remote URLs,
  which may contain credentials.

## Consequences

- Users approve each Git inspection invocation; the tool appears enabled in AI
  settings by default.
- Subagents cannot use the interactive tool because they cannot obtain user
  approval.
- The allowlist is intentionally narrow. Any added action must be audited for
  side effects and parameter injection in `utils/git` before it is exposed.

## Reference

- `ai/tools/git/git.go` - structured AI tool and ClonePane bridge
- `ai/tools/git/description.md` - model-facing action contract
- `utils/git/action.go` - read-only command allowlist and argument validation
- ADR 0071 - Git package owns action execution policy