# 0046 - Markdown images are container-capped

## Status

Accepted.

## Context

Notes View and Run already declared `max-width: 100%` for rendered images in
CSS. The shared Markdown post-processor subsequently wrote inline
`max-width: none` for images without a valid alt-text size suffix. Inline style
won the cascade, allowing large images to overflow both surfaces.

Optional alt suffixes such as `:20%` and `:2000px` also wrote inline maximums
without a container cap.

## Decision

Apply image sizing in the shared Markdown processor used by View and Run:

- Unsized or invalidly sized images use `max-width: 100%`.
- Percentage and pixel limits use `min(100%, requested-size)`, so explicit
  sizing may reduce an image but never exceed its container.
- Width and height remain `auto` to preserve aspect ratio.
- Existing viewport/pixel height limits remain unchanged.
- Keep the static View/Run CSS rule as a fallback before post-processing runs.

## Consequences

- Large Markdown images shrink to fit both View and Run surfaces.
- Small images retain their natural size because only the maximum is constrained.
- Explicit alt-text sizing remains supported but cannot create horizontal
  overflow.
- Fullscreen image display is unaffected because it uses a separate overlay.

## Reference

- `frontend/src/notes.css` - View/Run image fallback rule
- `frontend/src/markdown-utils.js` - shared inline image sizing
- `frontend/src/markdown-utils.test.js` - default and explicit sizing coverage
