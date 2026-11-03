# 0054 - Persisted graphs for Notes tables

Date: 2026-10-02

## Status

Accepted.

## Context

CSV and Markdown tables share Go-backed SQL filtering and a SQLite-backed cache
key composed from workspace, document path, table identity, and a hash of the
table headings. A surface-wide table index can shift between View and Run when
unrelated tables are introduced, so it is not a stable cross-mode identity.
Users need a graph to remain visible when switching between modes and when
reopening a table.

## Decision

- Add graph configuration to the existing cached table state rather than
  creating a second key scheme or a frontend-only preference.
- Reuse the SQL table-state namespace and key: workspace, document path, a
  per-heading occurrence index, and table headings. Keep the surface-wide index
  only for the in-memory tablecore registry. Read legacy global-index cache keys
  as a migration fallback.
- Add one `Create graph...` action to the context menu for right-clicks anywhere
  on Markdown and CSV tables. It opens the existing Notes prompt modal with a
  chart-type dropdown and multi-select column and row fields, defaulting to all
  columns and rows. When a graph is already configured, label the action
  `Modify graph...` and prefill that same modal from the persisted graph spec.
  The modal title and confirmation button distinguish creation from modification.
- Keep graph creation in the existing table context menu alongside `Copy`,
  `Copy table (CSV)`, `Copy table (Markdown)`, `Copy table (Formatted)`,
  `Clear sorting`, and `SQL filter…`.
  Heading and body-cell right-clicks use the same menu; table-level handlers must
  not stop propagation and hide the other actions. Run-mode row/column editing
  and the word-wrap toggle remain available where applicable.
- Preserve cell line breaks when copying tables: CSV copy converts `<br>` tags
  to newlines, allowing the CSV serializer to quote multiline fields; Markdown
  copy retains the literal `<br/>` tag; Formatted copy publishes HTML, an RTF
  table, and tab-separated plain text with a trailing row newline in one
  clipboard write. All three copy options exclude SQL-filtered rows and retain
  display order. The tab-separated fallback was manually verified to enable
  native table pasting into Slack with the rich representations (ADR 0028).
- Right-clicking a rendered graph offers `Copy graph (PNG)` with the copy icon,
  `Modify graph...` with the same graph icon as table creation, a separator,
  and the existing `Remove graph` action. Modification opens the owning table's
  prefilled graph modal; removal clears the rendered image and persisted spec.
- PNG copying asynchronously requests `format: png` through
  `NotesTableRenderGraph`, which reuses the Go graph renderer and the same SQL
  filter/selected-row intersection as SVG display. The binding returns a PNG
  data URL in `pngDataURL`, passed to `TerminalCopyImageDataURL` for a native
  clipboard write. The default/normal display response remains `svg`.
  Copying changes neither the persisted graph spec nor the displayed SVG.
  Rendering failures do not touch the clipboard; rendering and clipboard errors
  are reported through terminal notifications. Keep the generated Wails result
  model in sync with the additional response field.
- The first selected column is the category; selected numeric columns after it
  become value series. Pie and doughnut use the first selected numeric series.
- Persist selected source-row indices in the graph specification. Empty indices
  mean all rows.
- Render the chart as SVG below the table's horizontal-scroll wrapper and, when
  present, below the SQL filter footer.
- Clicking the graph opens the shared fullscreen image overlay with an inline
  SVG sized from its viewBox.
- Apply the SQL filter through the Go table engine before intersecting it with
  selected source rows. With no SQL filter, render the selected rows from the
  whole table.
- Persist the chart specification, not rendered SVG bytes. Re-render it from
  the current table data after restore, sorting, or filtering.

## Consequences

- Graph settings are shared across View/Run and Markdown/CSV render paths for
  the same cached table identity.
- Editing table data regenerates the graph from the current table rather than
  displaying stale image bytes.
- The cache TTL follows the existing table state cache policy.
- Graph-action regressions cover menu order/icons, modification from graph and
  table menus, PNG copying without state mutation, error reporting and removal.
  A backend test decodes the exported PNG and verifies nonempty dimensions.

## References

- `frontend_tables.go` - graph state persistence and SVG rendering bindings
- `utils/notes/table_state.go` - shared SQLite cache key and state
- `frontend/src/notes.js` - table menu, SQL-filter row selection, and graph view
- `utils/graphs/graphs.go` - table-to-chart renderer
- ADR 0028 - Go-backed Notes table sorting and filtering
- ADR 0052 - reusable graph rendering package
- ADR 0053 - AI graph generation tool
