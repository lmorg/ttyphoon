# 0079 - Document printing uses isolated snapshots and platform backends

Date: 2026-10-24

## Status

Accepted. Consolidates printing decisions implemented on 2026-10-23 and
previously recorded under ADR 0024's menu-dismissal discussion.

## Context

Printing the application webview directly included the file sidebar, Tools
panel, terminal, and sometimes the context menu used to select Print. Scroll
containers clipped documents to the visible viewport, and Monaco's virtualized
DOM did not contain the complete source. The screen theme was unsuitable for
paper or PDF output.

Wails' macOS print implementation also hardcodes landscape orientation and
zero margins. A CSS portrait hint did not override that native default. The
application needs the native print dialog, including Save as PDF, without
maintaining a Wails fork.

These concerns are separate from dismissing menus before dispatching their
selection. That ordering remains governed by
[ADR 0024](0024-dismiss-menu-before-dispatching-selection.md).

## Decision

### Print Sources

- Markdown View renders the complete document body through the shared Markdown
  processing pipeline, with frontmatter stripped as in the live View.
- Edit reads the complete editor buffer, including unsaved Monaco changes,
  rather than the visible editor DOM or a stale hidden textarea. A separate
  filename line precedes the source: `$PROJECT` files use their project-relative
  path; other categories use the basename. Filenames and source are inserted as
  text, not HTML.
- Jupyter Run offers Print but renders the document through the Markdown View
  pipeline. Runtime output and controls are excluded, and the active tab does
  not change.
- Markdown modals offer Print in both body and table context menus. They retain
  their original Markdown so printing is independent of scrolling, table
  filtering, and other changes to the displayed modal. Current callers are Git
  worktree information, Git branch/tag information, and View command meta.
- The AI panel prints top-level assistant `text` blocks as Markdown. Thinking,
  tool calls/results/errors/summaries, user-prompt headings, and nested subagent
  text are excluded. Offscreen historical output is fetched by block ID without
  loading excluded blocks or changing the displayed history. Legacy transcripts
  print only classified `Final Answer` sections; unclassified transcripts are
  not printed. With no assistant output, show a notification instead of an empty
  print dialog.
- Other document modes retain their window-print path and discard any retained
  document snapshot before using it.

### Isolated Print Document

`frontend/src/notes-print.js` prepares a body-level `#notes-print-root` from the
chosen source. It is hidden and positioned offscreen during normal use. Print
media shows the snapshot and hides every other body child, excluding application
panels, menus, and tooltips.

Render Markdown afresh rather than copying a scrolled or collapsed live view.
Expand native `details` elements in the snapshot only. Remove fixed-height and
overflow clipping, wrap long source lines and table cells, constrain images and
SVGs to the page width, repeat table headers, and discourage page breaks between
headings and their following content. The live document layout, selection,
scroll position, collapsed sections, and screen theme remain unchanged.

Close shared tooltips and wait for menu dismissal animations before printing.
Wait for Markdown processing, image decoding, font readiness, and layout; asset
waiting is bounded so an unavailable resource cannot indefinitely prevent the
dialog from opening. Concurrent requests during preparation share one operation.

### Snapshot Lifetime

Do not use the print binding's promise or an unqualified `afterprint` event as
universal evidence that the document is no longer needed. Wails' fallback may
return before its native dialog closes, while preview completion may precede a
later PDF render.

Ignore `afterprint` while the print binding is pending. After it resolves, a
subsequent event may remove the snapshot. If the platform emits no suitable
event, retain the hidden snapshot until the next request replaces it. Failed
requests discard the snapshot. This prevents preview cleanup from removing the
content needed for Save as PDF.

### Print Styling

Keep print styling in `frontend/src/notes.css` under print media, independent of
the live application theme:

- Use the user-supplied iTerm Display-P3 palette through CSS `color(display-p3
  ...)` values. Foreground, bold, and source text use black; ANSI and link colours
  retain their supplied components. Request exact print colour adjustment.
- Capture each element's original computed font size before enabling print
  media. The current text reduction is `3pt`, with a `1pt` minimum. Code blocks,
  their `code` elements, and highlighted token descendants use `4pt`, making code
  one additional point smaller. Subtract from captured sizes rather than nested
  `em` values so reductions do not compound and highlighted spans cannot reset
  the size.
- Checked and unchecked checkbox outlines use `--fg` with transparent
  backgrounds. Checked boxes retain an `--fg` tick sized `2pt` above the inherited
  checkbox font size.
- Current page styling requests portrait orientation, `5mm 0` page margins,
  and a `5mm` inner gutter. The experimental first-page top-margin override is
  currently disabled. Clone box decoration across fragmented pages. These are
  stylesheet preferences, not native-backend constants.
- Mermaid SVG colours remain those chosen by the diagram renderer rather than
  being recoloured through CSS.

### Platform Print Backend

`WApp.WindowPrint` delegates to `utils/printing.Print(ctx)`:

- macOS uses an app-owned Objective-C bridge, requiring cgo and macOS 11 or
  later. Find the main window's `WKWebView` through Cocoa's public view hierarchy;
  do not depend on Wails' private window context or modify Wails.
- Copy `NSPrintInfo` rather than mutate shared settings. Select portrait, fit
  horizontal pagination, keep vertical pagination automatic, and disable
  centering. Native margins stay zero so the print CSS controls spacing.
- Set the native print view's frame to `webView.bounds`. WebKit initially
  returns a zero-sized print view; omitting this initialization leaves no useful
  drawing area for PDF output.
- Start printing on the main thread with
  `runOperationModalForWindow:delegate:didRunSelector:contextInfo:`. Signal the
  waiting Go caller from the completion delegate while leaving the main queue
  available for WebKit. Synchronous `runOperation` stalled the native PDF fixture
  even after its frame was corrected.
- Wait for native operation completion before resolving the macOS binding.
  Cancelling the dialog is not an application error; setup errors reject the
  existing binding promise. A cancelled input context is rejected before
  invoking a backend.
- Non-macOS builds delegate to `runtime.WindowPrint(ctx)`. macOS builds without
  cgo report that native printing is unavailable rather than silently restoring
  Wails' landscape implementation.

## Consequences

- All supported document print actions reuse preparation, pagination, theme,
  font sizing, and the native dialog instead of implementing separate exporters.
- Printing can include complete unsaved source and unloaded historical AI output
  without mutating the visible document or including tool/reasoning data.
- macOS orientation and operation lifetime are application-owned, without a
  Wails fork. Linux and Windows retain the existing Wails printing behavior.
- Print snapshots may remain allocated, but hidden, until the next request on
  platforms that do not emit a reliable completion event.
- Browser print emulation alone is insufficient to verify saved native PDFs.
  Preserve the native output regression alongside frontend lifecycle tests.

## Verification

Frontend tests cover source/filename safety, complete unsaved Monaco output,
Run-to-View routing, modal printing, AI exclusion and lazy history loading,
snapshot lifecycle, menu dismissal, print styling, and font-size capture.

Go tests cover cancelled contexts and a missing native application. Linux and
Windows fallbacks are cross-compiled. An opt-in macOS test saves a PDF from a
hidden WebKit window through the production operation helper, without opening a
print dialog. PDFKit checks all 120 paragraphs, start/end markers, multiple
portrait pages, and exclusion of application controls. Run in a WindowServer
session:

```sh
TTYPHOON_TEST_NATIVE_PRINT=1 go test ./utils/printing -run TestNativePrintPDFContainsPreparedDocument -v
```

The verified fixture produced five portrait pages with 5,930 characters.
Physical printer and interactive dialog behavior still require manual checks.

## Reference

- [Print preparation](../frontend/src/notes-print.js) and [tests](../frontend/src/notes-print.test.js).
- [Print styles](../frontend/src/notes.css).
- [Notes print actions](../frontend/src/notes.js) and [integration tests](../frontend/src/notes.test.js).
- [Markdown modal](../frontend/src/markdown-modal.js) and [tests](../frontend/src/markdown-modal.test.js).
- [AI output selection](../frontend/src/ai_pipeline_formatter.js).
- [Printing package](../utils/printing/printing.go) and [macOS bridge](../utils/printing/printing_darwin.m).
- [Native PDF regression](../utils/printing/printing_darwin_test.go) and [fixture](../utils/printing/testdata/print_pdf.m).
- [ADR 0014: verification baseline](0014-test-baseline-and-verification.md).
- [ADR 0024: menu dismissal](0024-dismiss-menu-before-dispatching-selection.md).