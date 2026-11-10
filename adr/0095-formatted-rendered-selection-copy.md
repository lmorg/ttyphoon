# 0095 - Rendered selections share Go-backed clipboard formats

Date: 2026-11-09

## Status

Accepted and implemented.

## Context

Rendered Markdown selection copying previously flattened content to text.
Whole-table Formatted copy already published HTML, RTF, and tab-separated text,
but its conversion rules lived in the frontend. Selections need those same table
rules alongside headings, inline formatting, lists, code, and images. Exporting
the original Markdown would lose table sorting, SQL filtering, formula results,
and the user's partial selection.

## Decision

- `utils/copy_format` owns conversion from normalized rendered HTML to HTML,
  RTF, plain text, Markdown, and Slack HTML. `Convert` produces these formats; `HTML`, `RTF`, and
  `PlainText` expose individual representations. Core conversion does not
  depend on Wails, DOM access, or clipboard APIs.
- Notes captures the selection synchronously when opening its context menu,
  before menu interaction can move the selection. Native `copy` events capture
  the current selection. Preserve partial text and table selections and their
  semantic ancestor elements rather than expanding to whole tables.
- Capture the live DOM in display order. Exclude filtered/hidden rows, cell
  references, formula attributes, controls, modal line numbers, and other UI
  decorations. Read displayed cell values, never original formulas.
- Snapshot charts, Mermaid diagrams, and other SVG/canvas content as images.
  Inline PNG images in HTML using base64 data URLs and in RTF using
  `\pict\pngblip` with hexadecimal bytes. Image loading/rasterization failures
  retain alt text and the URL where available and show a warning after copying
  the remaining content. A conversion or clipboard failure reports an error.
- Use semantic HTML tags and neutral rich-text styling, not the current screen
  theme or syntax colors. Strip executable content, presentation attributes,
  unsafe links, and internal export metadata. Bound input size, nesting, and
  image dimensions.
- Selection plain text uses the same GitHub-flavoured Markdown representation
  as the custom `text/markdown` entry: headings, ordered/bullet lists, emphasis,
  links, quotes, pipe tables, images, inline code, and fenced code without
  language tags. Escape literal Markdown punctuation and entity-like text,
  preserve explicit line breaks, and keep emphasis delimiters inside surrounding
  whitespace. Underline uses inline `<u>text</u>` HTML, not `__text__` (which
  means bold); superscript/subscript also use inline HTML.
- Reuse the former Formatted table rules: HTML cell markup, RTF table rows and
  cell boundaries with a bold first row, and tab-separated displayed values.
  Whole-table plain text retains its trailing row newline. The existing
  `Copy table (Formatted)` action uses `CopyFormattedTable` on this shared
  pipeline and retains tab-separated plain text. Selection copy instead uses
  `CopyFormattedSelection`, even for a selection containing an entire table.
  Both actions share the same asynchronous queue, HTML, RTF, and Chromium custom
  data; only standard plain text differs. CSV and Markdown table-copy actions
  retain their encodings.
- `CopyFormattedSelection` formats content in Go and publishes standard HTML,
  RTF, plain text, and Chromium custom data in one clipboard write. The target
  application chooses its preferred format. Generate its Wails binding and
  import it through the stable wrapper (ADR 0013).
- The Chromium custom container holds `slack/html`, `text/html`, and
  `text/markdown`. Encode its entry count and UTF-16 key/value strings using
  Chromium's length-prefixed, four-byte-aligned Pickle layout. Publish it as
  `org.chromium.web-custom-data` on macOS, `Chromium Web Custom MIME Data Format`
  on Windows, and `chromium/x-web-custom-data` on Linux.
- Slack HTML follows the structure inspected in Slack's native clipboard:
  block wrappers with `data-block-type`, and table blocks with JSON
  `data-block-source` containing rows of `raw_text` or `rich_text` cells.
  Rich cells retain bold, italic, strike, code, and links. Generate this
  metadata from sanitized selected content, not copied Slack IDs or channel
  metadata. Keep tables separate from surrounding rich-text blocks, including
  when the selected HTML nests a table inside a div.
- Previously tried Slack design (before the hybrid text experiment): emit bold
  as `<b data-stringify-type="bold">text</b>`, matching
  Slack's native clipboard markup, rather than `<strong>` or Markdown
  delimiters. Structured table cells still use `style.bold`; standard HTML
  retains `<strong>` and Markdown retains its own emphasis syntax. Do not
  strip literal asterisks from user content.
- Slack HTML replaces empty or whitespace-only table cells, including headers,
  with `-` in both the HTML and structured table metadata. Other clipboard
  representations retain empty cells.
- Fragment-only same-document bookmark links (`#section`, including `#`) become
  ordinary formatted text in Slack HTML and table metadata. External links
  with fragments and all non-Slack representations retain their hyperlinks.
- Unwrap all unclassed divs in Slack HTML, preserving their children and order,
  while retaining Slack's classed block and rich-text section wrappers.
  Other clipboard representations are unchanged.
- The previously tried design kept lists as `ul`/`ol`/`li`, and converted
  `h1` through `h6` to paragraphs prefixed with the
  corresponding Markdown hashes (`#` through `######`), retaining inline
  formatting. Other representations retain their existing heading rules.
- That design preserved attribute-free `<p>` and `<em>` tags, including
  italic styles in structured table cells. Removing these tags experimentally
  did not resolve missing Slack list bullets, so that experiment was reverted.
- Current hybrid Slack design: outside tables, headings use hash-prefixed text
  with an underscored title (`## _Title_`); bold uses `*text*`; italic/emphasis
  and underline use `_text_`. Lists use literal `- ` or `N. ` markers with
  HTML line breaks, preserving ordered-list starts and nested indentation.
  Remove list-layout whitespace and duplicate trailing breaks so consecutive
  list items, including after nested lists, have one line break, not a blank
  line. Ignore whitespace-only siblings when separating list-item paragraphs.
  Inline spans hold literal formatting markers; paragraphs and Slack's own
  wrappers remain HTML. Inline code and other unlisted elements retain HTML.
  Heading titles use their displayed text, avoiding nested emphasis markers.
  Add two explicit HTML line breaks before and after each non-table heading
  so the hash-prefixed title has a blank line on each side.
- Tables retain the previous HTML and `data-block-source` metadata, including
  native bold/italic styling, empty-cell placeholders, and bookmark handling.
  Formatting inside table cells is not converted to text markers. This hybrid
  experiment changes only `slack/html`, not standard HTML, RTF, plain text, or
  the custom `text/markdown` representation. Slack paste remains a manual check.
- HTTP(S) images in Slack HTML use their original source URL as literal
  `[fig N: URL]` references, with two HTML line breaks before and after.
  Number references in selection order, starting at one for each clipboard
  conversion, including images in table cells and their structured metadata.
  Without S3 configuration, local/generated images retain inline PNG handling.
  Read original image metadata before stripping it from exported HTML; never
  expose internal `data-copy-source` attributes in either HTML representation.
- Optional Social/S3 uploads give local/generated images shareable URLs for
  Markdown and Slack while successful HTML/RTF exports retain inline PNGs.
  Configuration, consent, authentication, object lifecycle, limits, and failure
  behavior are defined separately in
  [ADR 0096](0096-social-s3-clipboard-image-uploads.md).
- The custom Markdown representation uses pipe tables, preserving selected
  row/cell order, empty cells, inline formatting, and displayed formula values.
  Headerless partial selections get an empty Markdown header rather than
  treating the first data row as a heading. Standard plain text uses this
  Markdown for selections, and retains tab-separated tables for the explicit
  Formatted table action. The package's `PlainText` output retains its legacy
  encoding; its `Markdown` output is the strict Markdown representation.
  Legacy `ClipboardSetHTML` and
  `ClipboardSetFormatted` callers retain their original format sets.
- Use this path for AI output's Copy selection, rendered Notes View/Run
  selection copy, the Markdown modal's new Copy selection action, and native
  Copy/Cmd+C/Ctrl+C on rendered Markdown selections. Leave Monaco, editable
  controls, raw-code copy, and block-copy buttons on their existing paths.
- AI Copy prompt and Copy response also use the shared selection formatter,
  publishing Markdown plain text alongside HTML, RTF, and Chromium custom
  representations. Prompt copy renders only the stored user prompt. Response
  copy captures all top-level text answer blocks for the targeted job, without
  thoughts, tools, or nested/subagent output. Capture loaded blocks from the live
  DOM so table order, filters, and evaluated values survive. Load and render
  historical answers that have not yet been displayed without expanding their
  disclosure or mutating the screen; report loading/rendering errors explicitly.

## Consequences

- Selection copying reflects what the user selected and currently sees, not
  a reconstructed source document. Diagram/image content is atomic when
  selected; ordinary text and table cells remain partial.
- Rasterization and clipboard publication are asynchronous. Snapshots survive
  selection loss, and copy requests publish in invocation order.
- External images that cannot be read because of network or browser
  cross-origin restrictions degrade explicitly to text/URL rather than
  silently disappearing.
- The screen document, current filters, table state, and selection are not
  mutated by exporting.
- Slack's private HTML format is a compatibility implementation, not a
  documented import contract. Its receiver may impose table limits or change
  parsing behavior. Inline PNGs remain in HTML/RTF, but advertising Slack HTML
  does not upload image attachments to Slack; image-only table cells use alt
  text in Slack's structured cell data. Receiving/pasting into ttyphoon is
  unchanged.

## Verification

- Go tests cover semantic HTML, RTF controls/Unicode, table boundaries and
  multiline values, Markdown-style text and nested lists, inline PNGs, and
  invalid input.
- Frontend tests cover partial selections, table state and computed values,
  diagram rasterization, warning/error reporting, native copy, menu snapshots,
  Notes View/Run/AI integration, and modal menus.
- Browser verification renders a real Mermaid flowchart and an SVG chart image
  into inline PNGs and exercises a native browser copy event.
- Additional Go tests cover Slack table JSON and block boundaries, rich cell
  styles/links, empty cells, metadata removal, separate Markdown tables,
  Unicode/surrogate pairs, Pickle alignment, and platform-specific clipboard
  format names. A native macOS round-trip independently decodes the published
  container. These protocol checks do not replace a manual paste into Slack.
- Markdown tests cover literal punctuation/entity escaping, underline, headings,
  lists, quotes, links, code, images, line breaks, and pipe tables. Clipboard
  tests verify that selection plain text equals the Markdown representation
  while Formatted table plain text retains tabs and all rich formats match.
  Frontend tests distinguish the two actions and verify shared write ordering.

## References

- [ADR 0096: Social S3 clipboard image uploads](0096-social-s3-clipboard-image-uploads.md)
- [ADR 0028: Go-backed Notes tables](0028-go-backed-notes-table-sorting.md)
- [ADR 0054: Notes table graphs](0054-persisted-graphs-for-notes-tables.md)
- [ADR 0013: Wails bindings](0013-wails-binding-checklist.md)
- `utils/copy_format` - format conversion and tests
- `frontend/src/copy-formatted.js` - rendered selection capture and image export
- `frontend/src/notes.js`, `frontend/src/markdown-modal.js` - copy actions
- `window/frontend/main/clipboard.go` - Wails and clipboard boundary
