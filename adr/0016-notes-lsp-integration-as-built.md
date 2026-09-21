# 16. Notes LSP integration as built

Date: 2026-09-06

## Status

Accepted — records verified implementation state, audited 2026-09-06.

## Context

`ai-docs/LSP-steps.md` and `ai-docs/LSP.md` describe the Notes LSP integration,
but both predate later work and describe it as an MVP with constraints that no
longer hold. This record captures what the code actually does, verified against
`utils/lsp/`, `frontend.go`, `frontend/src/notes.js`, and the hotkey registry.

Doc drift and outstanding gaps are recorded separately in ADR 0017.

## Decision

Record the following as the verified baseline for future LSP work.

### Session model

One server process per `SessionKey{WorkspaceRoot, LanguageID}`
(`utils/lsp/manager.go`). Multiple languages can be active in one workspace.
Activation requires a `Notes.LSP.<language>` argv entry; language id is resolved
from Jupyter language metadata. Unconfigured file types silently have no LSP.

### Protocol coverage

25 methods implemented across `utils/lsp/`:

- **Lifecycle** — `initialize`, `initialized`, `shutdown`, `exit`,
  `$/cancelRequest`
- **Sync** — `didOpen`, `didChange`, `didSave`, `didClose`
- **Server notifications** — `publishDiagnostics`, `window/logMessage`,
  `$/progress`, dispatched through one unified listener
- **Language features** — `hover`, `completion`, `definition`, `formatting`,
  `rangeFormatting`, `codeAction`, `prepareRename`, `rename`, `documentSymbol`,
  `signatureHelp`, `semanticTokens/full`, `inlayHint`, `codeLens`,
  `resolveCodeLens`
- **Workspace** — `workspace/symbol`, `workspace/executeCommand`,
  `didCreateFiles`, `didRenameFiles`, `didDeleteFiles`

`textDocument/references` is **not** implemented anywhere.

### Document sync

Full-document sync only. `didChange` sends `contentChanges: [{text: <whole
document>}]` with no `range` field. Incremental sync remains unimplemented.

### Position encoding

`initialize` advertises `general.positionEncodings` for UTF-16 and UTF-8 plus the
legacy `offsetEncoding` field, parses the server's selection, and defaults to
UTF-16 when unspecified. Conversion is applied on **every** feature path — both
outbound request coordinates and inbound coordinates returned to the frontend
(diagnostics, definitions, symbols, semantic tokens, inlay hints, code lens).
Surrogate pairs are handled explicitly in `position_encoding.go`.

### Reliability

- Crash detection in the read loop triggers `restartWithBackoff`: 500ms initial,
  ×2 multiplier, capped at 30s, with **no maximum retry count**.
- `Stop()` performs a graceful `shutdown` request (500ms timeout) followed by an
  `exit` notification.
- Startup and attach failures surface as `NOTIFY_ERROR` notifications, not
  log-only, deduplicated per workspace+language.
- Completion responses are capped at 200 items and filtered for empty labels.

### Invocation surfaces

LSP actions are **not** mouse-only. Three hotkeys are registered in
`config/defaults.yaml` and wired through the renderer hotkey registry:

| Binding | Function | Description |
|---|---|---|
| `F4::F4` | `LspOptionsMenu` | LSP: options menu... |
| `F4::f` | `LspFormatDocument` | LSP: format document |
| `F4::j` | `LspJumpToSymbol` | LSP: jump to symbol... |
| `F4::w` | `LspJumpToWorkspaceSymbol` | LSP: jump to workspace symbol... |

These reach the frontend as runtime events (`notesRunLspFormatDocument`, etc.).
Because `commandPaletteItems()` enumerates `hotkeys.List()`, they also appear in
the command palette automatically — there are no hardcoded LSP palette entries.

The editor LSP options menu offers six core actions (format document, go to
symbol, go to workspace symbol, signature help, code lens, rename symbol) plus
code actions grouped dynamically when the cursor position has any.

The completion popup is fully keyboard-driven: `ArrowUp`/`ArrowDown` to move,
`Enter`/`Tab` to commit, `Escape` to dismiss, `Ctrl`/`Cmd+Space` to trigger, with
`Backspace`/`Delete` re-requesting.

### Diagnostics rendering

Rendering is deferred while typing: the handler returns early if a change timer
is pending or if less than `LSP_DIAGNOSTIC_RENDER_IDLE_MS` (220ms) has elapsed
since the last input. The current file's diagnostics cache and visible chrome are
cleared on input and reload before repaint.

## Consequences

- The MVP framing in `ai-docs/LSP-steps.md` and `LSP.md` is obsolete for hotkeys,
  keyboard navigation, and invocation surfaces. Treat this ADR as authoritative
  and those documents as historical.
- Adding an LSP action means registering a hotkey function in
  `renderer_webkit/hotkeys.go`, which gets command-palette presence for free.
- Full sync means every keystroke burst sends the whole document. Acceptable at
  current file sizes; the first thing to revisit if large files feel slow.
- Unlimited restart backoff means a permanently broken server retries forever at
  30s intervals without a terminal "giving up" notification.

## Reference

- `utils/lsp/` — 22 test files covering transport framing, correlation, encoding
  conversion, workspace edits, symbols, and an in-memory fake-server integration
  test
- `frontend.go` — `NotesLsp*` bridge methods, `notesLspServerFor`,
  `notifyLspStartError`
- `frontend/src/notes.js` — request orchestration and overlay rendering
- `window/backend/renderer_webkit/hotkeys.go`, `command_palette.go`
- `config/defaults.yaml` — `Hotkeys.Functions` F4 bindings
