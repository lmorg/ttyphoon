# 0034 - Pin-to-viewport toolbars use scroll-synced transform, not `sticky`

## Status

Accepted.

## Context

Rendered code and quote blocks in the AI panel (`pre`/`blockquote` inside
`#notes-ai-output`) cap their height and scroll independently
(`overflow-y: auto`) when their content is long. Each block also grew a
hover-revealed action toolbar (copy-to-clipboard, and a new send-to-terminal
button reusing the same `SendToTerminal` Go binding Jupyter mode's "Send to
terminal" button already uses - see `runCodeBlockInTerminal`).

The toolbar needs to stay visible at the top of the block regardless of how
far the block itself has been scrolled. The first implementation used
`position: sticky` on the toolbar wrapper (inserted as the block's first
child, given `height: 0; overflow: visible` so it wouldn't reserve layout
space). This positioned correctly, but caused the AI panel's translucent
`--darken-background-overlay` backgrounds to visibly flicker darker/lighter
whenever the toolbar's hover-triggered opacity changed (e.g. moving the mouse
over any code block) - reproducible simply by passing the cursor over a code
block. `position: sticky` requires the browser to continually recompute the
element's constraint against its nearest scrolling ancestor, and combined with
an opacity transition on its children, this appears to force a repaint of the
enclosing translucent compositing layers in the Wails/WKWebView renderer.

## Decision

Use `position: absolute` (anchored to the block's own positioned box, as
before sticky existed) plus a scroll listener that sets a compositor-only
`transform: translateY(scrollTop)` on the toolbar:

```js
const syncActionsOffset = () => {
    actions.style.transform = `translateY(${block.scrollTop}px)`;
};
syncActionsOffset();
block.addEventListener('scroll', syncActionsOffset, { passive: true });
```

`transform` is GPU-composited and doesn't force the layout/constraint
recalculation that `sticky` does, so the toolbar stays pinned to the visible
top of the block's own scroll area without the repaint side effect.

## Consequences

- Any future "stays put while its container scrolls" UI in this codebase
  should default to this scroll-listener + `transform` pattern over
  `position: sticky`, at least inside the AI panel's translucent-background
  chrome, until there's reason to believe the underlying WKWebView issue is
  fixed.
- The toolbar's DOM position (first child vs last child of the block) no
  longer matters for layout, since `position: absolute` takes it out of flow
  either way - unlike the `sticky` version, which needed to be the first
  child to have a sensible static position.

## Amendment (2026-09-17): avoid a second nested darken layer

The apparent background flicker also exposed a separate compositing problem:
`#notes-main` already paints `var(--darken-background-overlay)`, while
`#notes-ai-output` independently painted the same translucent overlay over its
child area. The two layers are visually close enough to be mistaken for one
background, but a WebView repaint triggered by mouse movement can make the
double-darkening obvious.

The AI output element no longer paints its own darken overlay. The parent
`#notes-main` is now the single owner of that layer, preventing nested alpha
compositing and keeping hover/scroll repaints from changing the perceived
background intensity.

## References

- `frontend/src/notes.js` - `ensureRenderedBlockCopyButton`,
  `initRenderedBlockCopyButtons`
- `frontend/src/notes.css` - `.notes-copy-block-actions`
- ADR 0013 (Wails binding checklist) - `SendToTerminal` is the same binding
  reused here and by Jupyter mode's `runCodeBlockInTerminal`
