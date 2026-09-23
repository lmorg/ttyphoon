# 0043 - Adjusted terminal cells centre glyphs

## Status

Accepted.

## Context

`TypeFace.AdjustCellHeight` changes the terminal cell height after the font's em
height has been measured. The canvas renderer uses a `top` text baseline, so it
previously drew every glyph at the cell's top edge. Positive height adjustments
therefore added all spacing below the text instead of distributing it around the
glyph.

Cell backgrounds, underlines, strike-throughs, images, and terminal geometry use
the complete adjusted cell dimensions and should retain those anchors.

Canvas uses `source-over` compositing. If a font paints outside its allocated
cell, anti-aliased pixels from adjacent glyphs overlap and blend, making repeated
characters appear brighter or the wrong colour even during a clean full-frame
render.

## Decision

The font controller owns a vertical glyph offset alongside its measured cell
size. After the adjusted cell height has been clamped, it calculates:

```text
glyphOffsetY = ceil((cellHeight - measuredEmHeight) / 2)
```

The terminal canvas renderer adds this offset only to glyph `fillText` and
`strokeText` coordinates. It does not move cell backgrounds or decorations.

Before drawing text, the renderer clips the canvas to the command's allocated
cell rectangle. A normal character receives one cell, a wide character receives
two, and a shaped run receives its complete run width. The clip applies to both
the glyph fill and search-result outline; backgrounds and decorations remain
outside it.

Rounding with `ceil` places text half a pixel below the mathematical centre when
the available spacing is odd, matching the preference for the lower pixel when
pixel-perfect centring is impossible. Calculating from the final cell height also
keeps negative adjustments and the minimum one-pixel cell-height clamp coherent.

## Consequences

- Positive `AdjustCellHeight` values distribute added space above and below text.
- The default adjustment of 3 pixels moves glyphs down by 2 pixels.
- Terminal row dimensions and backend resize calculations are unchanged.
- Search-result outlines stay aligned because fill and stroke use the same offset.
- Oversized and anti-aliased glyph pixels cannot blend with neighboring cells.
- Wide characters and ligature runs retain their complete allocated width.
- Decorations remain anchored to the cell rather than the glyph.

## Reference

- `config/config.go` - `TypeFace.AdjustCellHeight`
- `frontend/src/font.js` - cell measurement and glyph offset
- `frontend/src/font.test.js` - odd, even, zero, and negative adjustment coverage
- `frontend/src/terminal.js` - clipped glyph drawing
- `frontend/src/terminal.test.js` - adjusted row-coordinate and clipping coverage
