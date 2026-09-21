# 7. Interactive `ttyphoon://` links for agent-to-user round trips

Date: 2026-09-06

## Status

Accepted

## Context

Some agent operations need a decision from the user mid-run: tool consent, and
(later) direct questions from the model. A native modal or context menu would
block the agent goroutine and sit outside the panel's scrollback, losing the
context the decision relates to.

The panel already renders agent output as markdown, so interaction can be
expressed as markdown links.

## Decision

Use a custom URI scheme rendered inline in the panel, with a Go-side registry of
pending requests keyed by an opaque ID.

The pattern, shared by tool permissions and user questions:

1. Go allocates a request ID and a buffered `chan string`, stored in a
   mutex-guarded package-level map.
2. Go emits markdown links of the form
   `ttyphoon://<kind>?request=<id>&<answer-param>=<value>`.
3. The requesting goroutine blocks on `select` over the channel and `ctx.Done()`.
4. `markdown-utils.js` intercepts the scheme, cancels default navigation, and
   dispatches a `CustomEvent` carrying the parsed query parameters.
5. `notes.js` marks the links resolved and calls the Wails binding.
6. Go looks up the ID, deletes it from the map, and sends on the channel.

Established conventions:

- **Resolution is idempotent.** A stale or already-resolved ID returns `nil`, not
  an error. Streamed content is re-rendered, so the same link can be clicked
  twice or survive a re-render; treating that as an error produced spurious
  `Failed to resolve …` notifications.
- **The frontend forwards parameters verbatim.** Adding or renaming a decision
  value requires no JavaScript change.
- **Resolved links are visually marked** — dimmed and struck through, with the
  chosen option highlighted — and have their `href` removed.
- **Cancellation is honoured.** The waiting goroutine also selects on `ctx.Done()`
  and removes its pending request on timeout or stop.

## Consequences

- Interaction stays inline, in context, and in the scrollback.
- The blocking goroutine is bounded by the request timeout (ADR 0004).
- **Every new binding must be added to the generated Wails files *and* imported in
  `notes.js`.** A missing import throws
  `ReferenceError: Can't find variable: …` at click time, which looks like a
  backend failure but is purely a wiring error. See ADR 0010.
- Pending requests are process-global, not per-agent. Acceptable because IDs are
  unique and single-use.

## Reference

- `ai/agent/agent.go` — write-permission request registry
- `ai/agent/user_question.go` — user-question request registry
- `frontend/src/markdown-utils.js` — scheme interception
- `frontend/src/notes.js` — event listeners and binding calls
