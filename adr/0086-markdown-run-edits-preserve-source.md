# 0086 - Markdown Run edits preserve source

Date: 2026-11-03

## Status

Accepted

## Context

Markdown Run renders the document as HTML and also contains interactive code
cells and tables. Editing a rendered node and converting its HTML back to
Markdown would lose source formatting and could rewrite unrelated content.

## Decision

- Offer `Edit` from the rendered-content context menu only for Markdown Run.
- Offer it for parsed headings, paragraphs, and entire lists with a resolvable
  source span.
- Map a list as one block, including its nested lists. Do not match individual
  de-indented list-item tokens against the original indented source: failed
  matches can shift mappings for all following paragraphs and headings.
- Right-clicking a link inside an editable block also offers `Edit`, with the
  existing native hyperlink menu retained through `Link actions`.
- Double-clicking one of those blocks starts editing in Run. Neither View nor
  Run collapses headings on double-click; collapse/expand uses the context menu.
- Headings have no hover underline or pointer cursor in either View or Run.
- Both View and Run offer `Collapse section` on right-click. It collapses the
  clicked heading, or the nearest preceding heading for content and blank-space
  clicks, using the existing collapsible-section wrapper. Link and image menus
  retain access to this action. Before the first heading there is no section to
  collapse, so the action is omitted.
- Right-clicking a collapsed heading offers `Expand section` instead, restoring
  its content through the existing expansion helper in both View and Run.
- Seed a plain textarea with the original Markdown for that block. Do not edit
  rendered HTML or serialize DOM back into Markdown.
- Size the inline textarea to its content on opening and every input event,
  including wrapped lines and a trailing empty line. Grow and shrink without a
  row cap, fixed minimum height, or manual resize handle.
- Inline Run textareas use the bespoke spellchecker and shared text-assistance
  suppression settings, including native spellchecking, autocorrect, writing
  suggestions, and Grammarly opt-outs. Detach spellchecking on Cancel,
  successful Save, or removal during a render/reload; a failed Save keeps it
  attached while the editor remains open.
- On Save, replace only that source span in the complete editor buffer, save via
  the existing Notes save path, and rerender Run from the updated Markdown.
- If Run's source snapshot is stale when editing starts, refresh from the live
  editor buffer without saving or reloading the file from disk. Resume editing
  only when exactly one block of the same element type has the original source
  text; changed, removed, or ambiguous targets leave the refreshed view visible.
- Cancel leaves the source unchanged. Code cells, tables, runtime output, and
  blocks without a reliable source mapping do not get this action.
- Preserve separators after the selected block so edits do not consume adjacent
  Markdown structure. Account for stripped frontmatter when calculating source
  offsets, and map blockquote text only when Marked provides an exact raw span.
- Clicking outside the inline editor leaves it open; Save or Cancel resolves
  the edit without a separate open-editor state flag.

## Consequences

- Markdown syntax entered in the editor is interpreted by the normal renderer on
  save, including new headings, paragraphs, and fenced code blocks.
- The edit path reuses the main Monaco source model and ordinary save behavior,
  preserving document-wide undo/change tracking instead of maintaining a second
  Markdown representation.
- The source mapper depends on Marked token raw text matching a source span; when
  that cannot be established, the menu action is omitted rather than risking an
  edit to the wrong text.

## References

- `frontend/src/notes.js` - Run rendering, source mapping, context menu, and save
- `frontend/src/notes.css` - inline editor presentation
- `frontend/src/notes.test.js` - frontmatter offsets, duplicate headings, entire
  lists, README nested lists, link menus, double-click, save, rerender, and
  cancel coverage
- ADR 0021 - navigation targets the live editor surface