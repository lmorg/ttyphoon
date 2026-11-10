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
  controls, raw-code copy, block-copy buttons, and AI Copy prompt/response on
  their existing paths.

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

- [ADR 0028: Go-backed Notes tables](0028-go-backed-notes-table-sorting.md)
- [ADR 0054: Notes table graphs](0054-persisted-graphs-for-notes-tables.md)
- [ADR 0013: Wails bindings](0013-wails-binding-checklist.md)
- `utils/copy_format` - format conversion and tests
- `frontend/src/copy-formatted.js` - rendered selection capture and image export
- `frontend/src/notes.js`, `frontend/src/markdown-modal.js` - copy actions
- `window/frontend/main/clipboard.go` - Wails and clipboard boundary
