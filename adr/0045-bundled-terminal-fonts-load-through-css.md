# 0045 - Bundled terminal fonts load through CSS

## Status

Accepted.

## Context

The bundled Fira Code Nerd Font stylesheet was linked from `frontend/index.html`,
but its `@font-face` URLs requested `FiraCodeFontMono-*.ttf` while the packaged
assets are named `FiraCodeNerdFontMono-*.ttf`. The declarations also used the
non-standard `format('ttf')` hint. Vite therefore could not resolve or bundle the
faces and canvas silently fell back to another monospace font.

The font controller also constructed a new face from `local("Fira Code")`. On a
host with plain Fira Code installed, that dynamic face could compete with the
bundled Nerd Font under the same family name, making the selected glyph set
dependent on the machine.

## Decision

- Keep the stylesheet link in `frontend/index.html`; its relative path is valid.
- Point each `@font-face` declaration at the exact
  `FiraCodeNerdFontMono-<Weight>.ttf` filename and declare it as `truetype`.
- Use `font-display: block` so initial terminal metrics do not settle on fallback
  glyphs while the bundled face is loading.
- Load the configured canvas family with `document.fonts.load`. Do not inject a
  same-named `FontFace` from a `local(...)` source.
- Continue using the CSS family alias `Fira Code`, which matches
  `types.DefaultMono` and configured `TypeFace.FontName` values.

## Consequences

- Vite resolves and emits every referenced Nerd Font face.
- The canvas and DOM use the same declared family and assets.
- A locally installed plain Fira Code no longer overrides the bundled Nerd Font.
- Custom system fonts still work because `document.fonts.load` also resolves
  installed families and the configured `monospace` fallback remains available.
- The unused Retina face remains unreferenced and is not bundled.

## Reference

- `frontend/index.html` - bundled font stylesheet link
- `frontend/src/assets/fonts/FiraCodeNerdFont/firacode.css` - face declarations
- `frontend/src/font.js` - canvas font loading and measurement
- `frontend/src/font.test.js` - CSS font-set loading coverage
- `types/consts.go` - default terminal font family
