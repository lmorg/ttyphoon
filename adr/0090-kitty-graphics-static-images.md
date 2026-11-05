# 0090 - Kitty graphics: static images

Date: 2026-11-05

## Status

Implemented static-image subset. **Not full Kitty graphics compatibility.**
The supported/deferred lists below are the implementation contract; update
them whenever the implementation changes.

## Context

Applications can send raster images through the Kitty graphics protocol using
`ESC _ G ... ESC \`. Previously these sequences reached the mxAPC parser and
reported unknown commands. `test/kitty-graphics.sh` also wraps its image in tmux
DCS passthrough, which previously was not decoded.

Image cells used by mxAPC/sixel replace text and only represent one drawable
element per cell. Kitty placements must preserve underlying text, overlap,
have independent image/placement IDs, and support reusable transmitted images.
Reusing the image cache is appropriate; treating placements as text-cell
elements is not.

## Decision

### Implemented

- [x] Kitty APC dispatch, separate from existing mxAPC commands.
- [x] Direct transmission (`t=d`) and `a=t`, `a=T`, `a=p`, `a=q`, `a=d`.
- [x] RGB (`f=24`), RGBA (`f=32`, default), and PNG (`f=100`).
- [x] Alpha transparency and zlib compression (`o=z`).
- [x] Chunked uploads (`m=1`/`m=0`), with first-chunk metadata retained and no
  rendering/reply until the final chunk has been validated.
- [x] Image IDs (`i`) and placement IDs (`p`), including anonymous placements,
  image reuse, placement replacement, and image retransmission replacement.
- [x] Native pixel sizing or cell-based sizing (`c`, `r`); one specified
  dimension preserves aspect ratio, and two dimensions fit/centre the image
  without distortion.
- [x] Source cropping (`x`, `y`, `w`, `h`) and within-cell offsets (`X`, `Y`).
- [x] Cursor policy (`C=1` preserves it; the default advances columns/rows).
  Movement beyond screen bounds is implementation-defined by the protocol:
  ttyphoon caps vertical advancement at one screen height and clamps the final
  column, avoiding arbitrarily large scrollback creation from one placement.
- [x] Z-order (`z`), including above text, below text, and below non-default
  cell backgrounds for values below `-1073741824`.
- [x] Basic scrolling/scrollback and window resizing: placements track their
  original terminal row, clip at pane edges, and disappear when that row is
  erased/evicted. Folded rows hide their anchored placements.
- [x] Alternate-screen isolation; leaving the alternate screen discards its
  placements while transmitted named image data remains reusable.
- [x] Deletion selectors `a/A`, `i/I` (optionally `p`), `c/C`, `p/P`, `q/Q`,
  `x/X`, `y/Y`, `z/Z`, and `r/R`. Lowercase retains reusable image data;
  uppercase releases data once no placements reference it. Deletion aborts
  an incomplete upload.
- [x] Synchronous query validation and ASCII `OK`/error replies through the
  originating terminal's PTY, with quiet policies `q=1` and `q=2`.
- [x] Kitty-only `ESC P tmux; ... ESC \` passthrough, including doubled ESCs
  and up to four envelopes. This is not general-purpose DCS execution.
- [x] Explicit cleanup on terminal reset/close and tmux pane exit.

### Deferred / not supported

- [ ] Files, temporary files, or shared-memory transmission (`t=f`, `t=t`,
  `t=s`, `S`, `O`); terminal output never requests filesystem reads.
- [ ] Image numbers (`I`) and their newest-image/deletion semantics.
- [ ] Unicode placeholders / virtual placements (`U`).
- [ ] Parent-relative placements (`P`, `Q`, `H`, `V`).
- [ ] Animation, frame composition, timing, loops, and frame deletion.
- [ ] Paletted formats (`f=8`, `f=16`) and image usage hints (`N`).
- [ ] Extended feature queries (`e`); the ordinary image-validation probe is
  supported, but no full-protocol capability response is advertised.
- [ ] Splitting a placement into independently scrolling fragments when only
  part of an image crosses a scrolling region.
- [ ] Recalculating existing pixel sizes when font metrics change; cell anchors
  and viewport clipping update, but the initial pixel size is retained.
- [ ] Restoring image payloads from persisted textual scrollback history.
- [ ] Guaranteed graphics-query-before-DA discovery through tmux control mode.

Unsupported controls/formats/actions return `ENOTSUP`, rather than claiming
success. Invalid input uses `EINVAL`, oversized input uses `E2BIG`, missing
image IDs use `ENOENT`, and exhausted storage uses `ENOSPC`. Errors are logged
even when the client requests no protocol replies.

### Architecture and bounds

`utils/kittygraphics` validates controls, accumulates chunks, validates dimensions,
decodes/decompresses bytes, and normalizes the result to PNG. Limits:

- 4096 encoded bytes per chunk; bounded APC/DCS envelopes consumed through ST
  even when oversized, preventing payload bytes from leaking into terminal text.
- 64 MiB maximum transfer and decompressed data; at most 16,777,216 pixels.
- 128 MiB accounted decoded-pixel + PNG storage per terminal, 128 images,
  512 placements, and 32,768 columns/rows per placement.
- Unplaced images are evicted oldest-first; displayed images are not silently
  evicted to admit a new upload.

The per-terminal placement collection holds row references and does not modify
`Cell.Char`, `Cell.Element`, or `Cell.Sgr`. The existing WebKit image cache stores
validated PNGs; an optional `types.GraphicsRenderer` interface supplies image
loading and pixel-aware drawing. Renderer compact operation 11 carries cell
anchors, pixel size/offsets, source rectangle, and pane clipping. Existing image
operation 9 and mxAPC/sixel behaviour are unchanged.

Rendering separates image/background/text layers only when a visible Kitty
placement exists. Background/text separation operates on copied SGR state.

### tmux and discovery

`%output` exposes the pane's original bytes, including explicitly wrapped
passthrough. Ttyphoon unwraps these locally and writes responses through that
same pane's PTY; it does not route replies via the currently selected pane.
No server-wide tmux options or terminal identity variables are changed.

As documented in ADR 0080, tmux can answer a subsequent DA query before its
control client processes the graphics probe. Local synchronous ordering cannot
fix that. Consequently static rendering and graphics-only acknowledgements
are supported through this path, but automatic discovery using graphics + DA
is not guaranteed through tmux. Cooperating clients can explicitly enable
graphics and use `q=2` where responses are not wanted.

## Consequences

- Static inline graphics work without pretending to be Kitty or implementing
  risky filesystem access.
- The checkerboard example is an executable regression fixture for direct and
  tmux-wrapped RGB output.
- Protocol/placement/canvas tests cover decoded pixels, alpha, compression,
  chunking, malformed/oversized input, IDs, deletion, resource quotas, layering,
  cropping, offsets, pane clipping, scrollback, alternate screens, cleanup,
  split input, and direct query ordering.
- Protocol decoding is synchronous on the terminal parser, preserving response
  order; image decoding may briefly delay parsing within the enforced limits.
- Validation includes actual tmux 3.8 control-mode byte preservation and an
  isolated browser canvas check of alpha blending, source cropping, pixel
  offsets, and pane clipping. This does not claim a native-window end-to-end
  test or guaranteed tmux discovery ordering.

## Reference

- [Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/)
- [ADR 0080 - Reporting terminal capabilities](0080-reporting-terminal-capabilities.md)
- [Example](../test/kitty-graphics.sh)
