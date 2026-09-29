# 0049 - Collapse AI tool output by default

Date: 2026-09-29

## Status

Accepted.

## Context

The AI panel displays thoughts, tool inputs, tool outputs, summaries and errors
as typed stream blocks. Large tool output can obscure the model's reasoning and
the actions being taken, even though the complete stream must remain available
in SQLite and in the panel when requested.

## Decision

- Keep thoughts visible. Hide tool-call input by default, retaining its heading
  as a control that toggles the input.
- Collapse `tool-output` and `tool-summary` child blocks by default. Keep
  `tool-error` blocks visible; summary-generation failures use `tool-error` too.
- Add a Markdown-styled checkbox to the owning tool-call block. It is unchecked
  and captioned `Running` with a spinner while the tool-call block is open;
  unchecked and captioned `Summarising` while a summary is streaming; checked and
  captioned `Successful` when output or a summary completes, and unchecked and
  captioned `Error` when a tool error occurs. The status row uses the smaller
  Markdown font size.
- Render a tool name in the clickable heading as a literal inline `<code>`
  element, without Markdown parsing or backtick characters.
- Initialize completion state from persisted block metadata, before lazy content
  hydration, so restored completed summaries do not appear to still be running.
- Clicking the checkbox reveals or collapses output and summaries without
  changing its status. Revealing an open summary shows its existing live stream,
  including deltas received after expansion.
- Do not change stream persistence or typed-block transport. Live and historical
  views use the same formatter behavior.

## Consequences

- The live panel is less dominated by raw tool output while retaining one-click
  access to the full result.
- Errors remain immediately visible, and SQLite continues to retain all blocks
  independently of whether their content is expanded in the panel.

## Reference

- `frontend/src/ai_pipeline_formatter.js` - block hierarchy and output disclosure
- `frontend/src/notes.css` - output status control
- `frontend/src/notes.test.js` - output, summary, streaming and error behavior
- `adr/0037-ai-stream-is-addressable-blocks-in-sqlite.md` - typed block storage