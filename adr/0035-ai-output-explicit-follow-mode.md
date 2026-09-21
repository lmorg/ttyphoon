# 0035 - AI output uses explicit follow mode

## Status

Accepted.

## Context

The AI output panel receives content incrementally while an agent is running.
The old behavior continuously chased `scrollHeight`, which kept the newest
content visible but prevented the user from reading older output: new content
could move the viewport back to the bottom while the user was scrolling up.

The panel already had a `Latest` button and an `aiStickToBottom` state, but the
state was inferred from whether the viewport happened to be near the bottom.
That meant follow mode could be re-enabled implicitly, and it did not cleanly
represent user intent. A user trying to scroll away from the live tail should
not have their choice overridden by the next streamed chunk.

## Decision

Treat auto-follow as an explicit mode:

- The panel starts in follow mode.
- While follow mode is enabled, streamed output and delayed render passes may
  chase the bottom as content grows.
- A user wheel or touch scroll pauses follow mode immediately.
- A user scrolling/dragging far enough away from the bottom also pauses follow
  mode through the scroll handler.
- Once paused, new content is appended without changing the user's viewport.
- Merely returning near the bottom does **not** resume follow mode.
- The `Latest` button is the explicit resume action: it enables follow mode,
  cancels stale retry state, and scrolls to the newest output.
- Historical prompt navigation continues to use an explicit forced-bottom
  action where appropriate.

This is implemented in `frontend/src/notes.js` by keeping
`state.aiStickToBottom` false after user scrolling, removing the implicit
`isAIOutputNearBottom()` fallback from final-output updates, and adding wheel,
touch, and scroll handling that calls `pauseAIAutoScroll()`. Existing delayed
bottom-chase callbacks already check `state.aiStickToBottom`, so pausing the
state stops them safely.

## Consequences

- Users can read older AI output while the agent continues streaming.
- The live tail remains the default for users who do not interact with the
  output scroll surface.
- Returning to the latest content is deliberate and discoverable through the
  existing `Latest` button.
- The AI panel may accumulate unseen output while follow mode is paused; the
  visible `Latest` affordance is the indicator that newer content exists.
- This policy is separate from the scroll-synced transforms used to pin code
  and quote block toolbars (ADR 0034).

## References

- `frontend/src/notes.js` - `pauseAIAutoScroll`, `setAIFinalOutput`,
  `requestAIScrollToBottom`, `scrollAIOutputToBottom`, AI output scroll handler
- `frontend/src/notes.css` - `.notes-ai-scroll-bottom`
- ADR 0034 - code/quote block toolbar positioning inside scrollable blocks
