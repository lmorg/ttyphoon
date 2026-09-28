# 0048 - Save rendered code blocks to disk

## Status

Accepted.

## Context

Rendered Markdown code blocks and AI output code blocks already supported copy and
terminal actions, but there was no way to save a block directly to a file. AI
output also contains tool calls, tool output and thinking blocks, where adding
extra actions to every block would increase menu and DOM work during streaming.

## Decision

Add `Save code...` to code-block context menus.

- Markdown View exposes the action for every rendered code block.
- AI output exposes it only for closed `text` blocks, which represent the final
  response. Tool, thinking, notice and subagent blocks do not expose it.
- The code text comes from the rendered `pre code` element, preserving the
  displayed block content without Markdown fences.
- The language class supplies the default extension, with `txt` as the fallback.
- A Wails `SaveCodeDialog` opens with the current project root as its default
  directory. The selected path is then written through the existing `SaveFile`
  binding.
- The AI streaming path does not attach or process this action. It becomes
  available during the final closed-block processing pass.

## Consequences

- Users can save a rendered snippet without first copying it into an editor.
- The project root is the initial save location, but the native dialog still
  allows choosing another location.
- AI tool and reasoning output remain free of the extra action and its associated
  interaction surface.

## Reference

- `frontend.go` - `SaveCodeDialog`
- `frontend/src/notes.js` - code-block extraction and context menus
- `frontend/src/ai_pipeline_formatter.js` - block kind/status data attributes
- `frontend/src/notes.test.js` - View and final-AI menu coverage
