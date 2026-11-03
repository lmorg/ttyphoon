# 0056 - One Bash tool executes commands in cloned terminal panes

Date: 2026-10-04

## Status

Accepted. Revised 2026-10-20; supersedes the original decision to expose separate
`interactive` and non-TTY `bash` tools.

## Context

The original two tools used different execution paths: `interactive` wrote into
the active shell, while `bash` spawned a non-TTY process in Go. Cloned terminal
panes now provide a TTY, inherited working directory/environment, and a one-shot
completion callback (ADR 0072). Bash can interpret both single commands and
multiline scripts inside that pane without modifying the source shell.

## Decision

- Expose one command tool, `bash`, using structured `{script, summary}` input,
  invocation approval by default, and denied subagent access by default. Remove
  `interactive` rather than keeping a second command execution route.
- Include the one-line purpose summary above the exact script in approval
  descriptions. Normalize whitespace and escape Markdown/HTML in that summary;
  it is explanatory text, never part of the executed command. Calls without a
  summary retain the script-only approval description for compatibility.
- Pass a shell-quoted `bash -c <script>` to the renderer's `CloneActivePane`,
  which calls tmux `ClonePane`. Shell syntax is always Bash. The clone inherits
  the source pane's current path and reported environment; there is no direct
  Go process or source-TTY fallback when cloning is unavailable.
- Let tmux own wrapper creation, completion authentication, cleanup, pane close
  controls, and the configured success auto-close timer. The tool waits for the
  existing completion callback; users can interact directly with the pane.
- Return JSON containing the captured terminal screen and process exit code.
  Setup/callback failures are model-visible `error` values; non-zero exits are
  failed command observations, not fatal agent-run errors. Output is a screen
  snapshot rather than an unbounded stdout/stderr log. It remains visible live
  in the terminal, but is no longer streamed live into the AI panel.
- Honour cancelled contexts before launching and while waiting. Cancelling a
  wait does not guarantee command termination; report that the pane may still
  be running. Late callbacks must not block after the wait has ended.

## Consequences

- Commands and scripts share one execution path and Bash syntax, while the
  user's original shell is not modified.
- Programs requiring a TTY are supported. The final screen snapshot may omit
  earlier output or contain terminal UI rather than a complete execution log.
- Only `bash` remains registered as a command runner; existing `bash` permission
  settings retain their tool ID. Old `interactive` settings no longer apply.
- The shell tool does not provide a general-purpose process sandbox; the
  existing tool-permission policy remains the invocation boundary.

## References

- `ai/tools/shell/bash.go` - clone-backed Bash invocation and observations
- `ai/tools/shell/bash_test.go` - synthetic callbacks, quoting, and cancellation
- `ai/tools/shell/bash_description.md` - model guidance
- `tmux/pane.go`, `tmux/interactive_action.go` - pane execution and completion
- `ai/agent/runtime_eino.go` - tool errors returned to the model
- ADR 0009 - reported shell environment state
- ADR 0051 - structured native tool inputs
