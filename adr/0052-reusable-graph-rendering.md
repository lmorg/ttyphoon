# 0052 - Reusable graph rendering package

Date: 2026-10-02

## Status

Accepted.

## Context

AI tools and Notes tables need a shared way to turn tabular data into charts.
The application already defines its visual palette in `types`, while chart
rendering should be independent of AI and frontend integration details.

## Decision

- Add a reusable Go package at `utils/graphs`, backed by
  `github.com/go-analyze/charts v0.6.1`.
- Provide a table-shaped input with headings and rows, plus an explicit chart
  specification that selects the zero-based category column and numeric value
  columns. Do not guess which columns to plot.
  An optional `CategoryColumns` selection overrides the legacy singular
  `CategoryColumn`; join its cell values with `, ` in the supplied order.
  Validate every category index and reject overlap with any value column.
- Support table-compatible line, area, bar, horizontal bar, scatter, pie, and
  doughnut charts. Exclude specialized chart types whose input does not map
  naturally from ordinary tabular data, including radar, heatmap, candlestick,
  funnel, violin, and rendered-table charts.
- Require exactly one value series for pie and doughnut charts. Scatter charts
  use the selected category column for sample positions and numeric columns as
  Y series, matching the library's tabular scatter model.
- Return rendered bytes in SVG or PNG format, defaulting to SVG. Do not write
  files or introduce AI-tool or Notes UI integration in this change; consumers
  can use the shared package API.
- Treat blank numeric cells as missing chart values and report non-numeric or
  non-finite cells with their source row and heading.
- Keep chart dimensions at 800x600 through package constants rather than
  exposing dimensions as caller options.
- Expose title, axis/category labels, and series names as the initial caller
  options. Do not expose chart-library-specific advanced options or custom
  theme overrides in the initial API.
- Derive default chart colors from the project's `types` foreground,
  background, accent, and ANSI theme colors.

## Consequences

- AI tools and CSV/Markdown table consumers can share rendering and validation
  without depending on one another.
- Explicit column selection avoids surprising chart mappings and gives AI tools
  a schema that can describe what is being plotted.
- Specialized chart data models and UI integration remain possible follow-up
  work without expanding the initial API.

## Reference

- `utils/graphs` - shared graph input, theme mapping, and renderer
- `types/ansi_colours.go` - project theme palette
- `github.com/go-analyze/charts` - rendering backend
