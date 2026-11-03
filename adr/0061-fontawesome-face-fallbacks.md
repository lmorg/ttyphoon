# 0061 - Font Awesome uses separate regular and solid faces

## Status

Accepted and implemented on 2026-10-09.

## Context

The frontend had a single Solid `@font-face` in `style.css` pointing at an OTF
asset. Bundled Regular and Solid WOFF2 files are now available. Importing the
upstream stylesheets wholesale would also introduce shared family variables and
helper selectors that the frontend does not use, while both faces initially
used the same family name.

## Decision

- Define minimal faces with distinct family aliases: `Font Awesome Regular` at
  weight 400 and `Font Awesome Solid` at weight 900.
- Load the bundled WOFF2 files from `frontend/src/assets/fonts/FontAwesome/` and
  import both face declarations through `frontend/src/style.css`.
- Use Regular first, Solid second, then each call site's existing fallback
  fonts. Keep the Solid face's 900 descriptor as font metadata, but request
  normal weight (400) at every icon use site so icons are not synthetically
  emboldened.
- Do not include the upstream shared variables or helper classes; raw Unicode
  icon usage needs only the face declarations.

## Consequences

- Regular glyphs are preferred where present, and missing glyphs can fall back
  to the Solid face before system fonts.
- Frontend call sites must use the shared aliases in the same order and must not
  request bold weight for icon text.
- Vite resolves and emits both WOFF2 assets as part of the frontend build.

## Reference

- `frontend/src/assets/fonts/FontAwesome/regular.css` and `solid.css` - face
  declarations
- `frontend/src/style.css` - central imports
- `frontend/src/notes.css`, `frontend/src/inputbox.css` - static font stacks
- `frontend/src/popup_menu.js`, `frontend/src/style-utils.js` - dynamic/generated
  font stacks
- `frontend/src/notes.test.js`, `frontend/src/style-utils.test.js` - regressions
- ADR 0045 - bundled terminal fonts (separate terminal font policy)
