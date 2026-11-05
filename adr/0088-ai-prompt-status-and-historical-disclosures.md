# 0088 - Persist AI prompt outcomes and collapse historical stream details

Date: 2026-11-04

## Status

Accepted

## Context

The AI panel can navigate between the live stream and stored prompts. Historical
prompts use the same formatter as live output, so thinking, tool activity, and
intermediate responses can obscure the final answer. The prompt menu also
always offers Live output and has no persisted way to distinguish successful,
failed, and cancelled runs.

## Decision

- Store each stream prompt's outcome as `running`, `success`, `failed`, or
  `cancelled`. New prompts begin as `running`; finalization records the outcome.
- Classify a run as cancelled when its error wraps `context.Canceled`; other
  errors are failures. Completed databases without an outcome column migrate
  finished prompts to `success` and unfinished prompts to `running`.
- Include the outcome in prompt metadata. Prefix menu labels with `💭`, `✅`,
  `‼️`, or `🚫` for live, successful, failed, or cancelled prompts. Keep the
  menu's selection tick separate from the label.
- Mark live stream metadata as active so restoring the panel can distinguish an
  in-flight run from the newest completed prompt.
- Offer Live output only while a run is active. On completion, leave live mode
  and select the completed prompt; if the user was viewing a different prompt,
  load the completed prompt instead.
- When a historical prompt is selected, show its heading, user prompt, and start
  time first. Put intermediate stream blocks in one closed native
  `<details>/<summary>` disclosure, followed by the final response outside it.
  Load intermediate content when expanded and keep the final response visible.
- Keep live output rendering unchanged. Its completion continues to honor the
  final-answer reveal behavior in ADR 0035.
- Finalized AI Markdown also supports the same heading-section collapse/expand
  context-menu actions as Notes View/Run. Sections stay within their Markdown
  block, independently of historical stream disclosures. Streaming batch
  wrappers are removed during final processing so sections can span batches;
  repeated processing does not nest duplicate section wrappers. Actively
  growing output keeps its cheap streaming path until finalized.
- AI-panel table filtering, graphs, and column widths use a shared identity
  containing workspace, session ID, prompt ID, stream block ID, and the table's
  position within that Markdown block. Identical headings in different prompts
  or tables cannot share settings. In-flight blocks use their run ID until the
  prompt ID is assigned; completion carries live filter, graph, and width
  settings into the finalized prompt identity. Other Notes surfaces retain their
  existing keys.

## Consequences

- The prompt menu reflects run outcomes across application restarts, while
  legacy completed prompts are treated as successful because their prior
  outcomes were not stored.
- Historical prompts show their original request and start time, and defer
  loading all streamed content until the user expands the single disclosure.
- The live stream remains immediately visible and follows the existing scroll
  behavior; historical disclosure behavior is applied when navigating to a
  prompt.

## References

- `ai/agent/sessiondb/stream_blocks.go` - prompt outcome and start-time metadata
- `ai/agent/sessiondb/session_log.go` - lifecycle events and outcome propagation
- `ai/ui.go` - success, failure, and cancellation classification
- `frontend/src/notes.js` - prompt selection, status labels, and run transitions
- `frontend/src/ai_pipeline_formatter.js` - historical disclosures and lazy content
- `frontend/src/notes.css` - disclosure presentation
- `frontend/src/notes.test.js` - menu, lifecycle, and historical rendering tests
- ADR 0035 - final-answer reveal behavior
- ADR 0037 - typed stream persistence
- ADR 0049 - collapsed tool output
