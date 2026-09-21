# 17. Notes LSP gaps and documentation drift

Date: 2026-09-06

## Status

Accepted — audit findings, 2026-09-06. Companion to ADR 0016.

## Context

An audit cross-checking `ai-docs/LSP-steps.md` and `ai-docs/LSP.md` against the
code found that the implementation has moved ahead of the documentation in some
areas, and that the documentation describes at least one feature that does not
exist. Anyone changing the LSP integration needs to know which is which.

## Decision

Record the following as verified, and treat ADR 0016 as the authoritative
description of current behaviour.

### Documentation is wrong: features exist that the docs deny

| Doc claim | Reality |
|---|---|
| "None enabled for MVP (LSP interactions are mouse-driven)" | Three F4 hotkeys are registered and active |
| "Existing key behavior inside LSP completion popup: None enabled for MVP" | Arrow keys, Enter, Tab, Escape, Ctrl/Cmd+Space all wired |
| Three hardcoded command palette entries | Palette entries are generated from the hotkey registry; none are hardcoded |
| LSP options menu implied minimal | Six core actions plus dynamically grouped code actions |

The "Deferred Work" and "LSP Hotkeys" sections of `LSP-steps.md` are therefore
stale, as is the "LSP interactions are primarily mouse-driven" limit in
`LSP.md` §10.

### Documentation is wrong: a feature is described that does not exist

`LSP.md` §5.12 describes semantic token rendering in detail — token colours by
type, modifier-driven styling such as dotted underline and strike-through, and a
stronger tint in no-wrap mode.

**That specific styling does not exist**, and never will in that form. The
description belongs to a removed hand-rolled overlay that wrapped every token in
the whole document in a `<span>`, which hung the DOM on large files.

As of 2026-09-06 semantic tokens are rendered again, but by Monaco (ADR 0018),
so the visual result is Monaco's theme-driven token styling rather than the
bespoke classes described in the doc. `LSP.md` §5.12 should be read as intent
only.

### Gaps the docs describe accurately

These are genuine current limitations, not drift:

- **`textDocument/references` is absent entirely** — no backend method, no
  bridge, no UI, no hotkey. Docs correctly list it only as a post-MVP suggestion.
- **WorkspaceEdit application is active-file only.** Both `code_actions.go` and
  `rename.go` filter edits through `editsForURI(uri)`; edits for other files are
  silently discarded. A rename touching several files updates only the open one,
  with no warning.
- **Definition uses the first location only.** No picker when a server returns
  several targets, and no navigation-back stack. Targets outside indexed Notes
  files are rejected with a warning.
- **Full document sync only.** No incremental sync.
- **Completion cap is a plain truncation** at 200 items, with no ranking or
  server-specific adaptation, so a wanted item can be cut off.

### Untested areas

`utils/lsp` has 22 test files with broad coverage, but the restart/backoff loop
has no test. Since it retries indefinitely, a regression there could produce a
silent hot loop.

## Consequences

- Do not trust `LSP-steps.md` or `LSP.md` for current behaviour; they are useful
  for original intent and for the manual test scripts in `LSP.md` §5–§7.
- Silent multi-file edit loss is the most user-hostile gap — a rename appears to
  succeed while leaving other files stale. Any fix should either apply the edits
  or tell the user what was skipped.
- Adding `references` requires the full vertical slice: `utils/lsp`, a bridge
  method, regenerated bindings plus the `notes.js` import (ADR 0013), UI, and a
  hotkey registration.
- Semantic tokens and inlay hints were both fetched-but-discarded until
  2026-09-06; see ADR 0018. When auditing an LSP feature, check that something
  actually *consumes* the response, not merely that the request exists.

## Reference

- `utils/lsp/semantic_tokens.go`, `code_actions.go`, `rename.go`,
  `completion.go`, `document.go`, `process.go`
- `frontend/src/notes.js` — LSP request orchestration; note the absent
  `NotesLspSemanticTokens` import
- `frontend/wailsjs/go/main/WApp.d.ts` — `NotesLspSemanticTokens` exposed but
  unused
- `ai-docs/LSP-steps.md`, `ai-docs/LSP.md` — historical intent
