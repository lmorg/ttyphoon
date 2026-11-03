# 0053 - AI graph tool uses the shared tabular renderer

Date: 2026-10-02

## Status

Accepted.

## Context

ADR 0052 added a reusable table-to-chart API. The AI agent needs to produce
charts using the same validation, supported chart types, dimensions, output
formats, and theme mapping as future CSV and Markdown table consumers.

## Decision

- Register a native structured tool named `createGraph` in
  `ai/tools/graphs` and implement it using `utils/graphs.RenderTable`.
- Accept headings, string-valued rows, an explicit chart type, and zero-based
  category/value column indices. Keep the AI tool input aligned with CSV data;
  blank numeric cells are missing values, while invalid numeric cells are
  reported with row and heading context.
- Save generated PNG files under `~/Documents/<application>/.images/` using
  unique filenames. The tool always produces PNG; output format is not a
  caller-selectable input.
- Return a standalone file-backed Markdown image reference in the tool output.
  Do not put image bytes or data URLs in the AI stream or session log.
- Allow the tool without an additional invocation prompt; it only renders
  user-supplied data locally and writes a generated chart file.

## Consequences

- The chart appears inline in the AI conversation and remains available to
  later responses through its file path.
- AI tools and Notes table adapters share chart behavior through `utils/graphs`
  rather than reimplementing rendering.
- Image cleanup follows the existing `.images` lifecycle and is not performed
  by the graph tool.

## References

- `ai/tools/graphs/graph.go` - native tool implementation
- `ai/tools/graphs/description.md` - input and capability guidance
- `utils/graphs/graphs.go` - shared renderer
- ADR 0029 - file-backed images in AI output
- ADR 0052 - reusable graph renderer