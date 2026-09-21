# 18. Semantic tokens and inlay hints render through Monaco providers

Date: 2026-09-06

## Status

Accepted

## Context

Semantic token styling existed once and was removed because it hung the DOM on
large files. The old editor was a textarea plus a hand-rolled highlight overlay,
so styling a token meant wrapping it in a `<span>` — for the **entire document**,
rebuilt on every repaint. Cost grew linearly with file size and thrashed layout.

That architecture is gone. Notes is now Monaco-only (Monaco is default-on;
`localStorage['notes-editor-monaco'] = '0'` opts out), and Monaco virtualises
rendering — only visible lines exist in the DOM.

Both features were nevertheless left in a fetched-but-discarded state:

- **Semantic tokens** — backend implemented, `NotesLspSemanticTokens` exposed in
  the generated bindings, but `notes.js` never imported it. No consumer at all.
- **Inlay hints** — worse: `requestLspInlayHints()` fetched hints over IPC on
  every debounced change into `state.lspInlayHints`, then called
  `renderLspInlayHints()`, which was an empty stub whose comment claimed
  "rendered by Monaco providers" when no such provider existed. Every fetch was
  discarded, and the `.notes-lsp-inlay-hint` CSS had nothing to style.

## Decision

Both features are supplied to Monaco as language providers, registered in
`configureLsp` alongside the existing signature help, formatting, definition,
rename, and code action providers.

Token styling therefore rides Monaco's tokenisation pipeline, colouring spans it
already creates per visible line. Cost is proportional to the viewport, not the
file, so the original DOM-hang failure mode cannot recur.

Two constraints shaped the implementation:

1. **The legend arrives asynchronously.** Monaco requires `getLegend()`
   *synchronously* at registration, but the token legend only comes from the
   server's initialize response. `ServerProcess` now captures it
   (`SemanticTokensLegend()`), and the provider registration is deferred until
   the first token response, guarded by an `lspGeneration` counter so a
   `disposeLsp()` during the await cannot register a stale provider.

2. **Monaco columns are UTF-16.** The backend now passes the server's
   relative-encoded token stream through raw rather than decoding it to absolute
   positions. When the server negotiated a non-UTF-16 encoding,
   `convertSemanticTokensToUTF16` re-encodes offsets **while remaining in
   relative encoding**, recomputing each delta against the *converted* position
   of the previous token. Converting positions independently would corrupt every
   subsequent delta on the line.

Monaco caches provider results, so `refreshInlayHints()` fires an
`onDidChangeInlayHints` emitter after a debounced document change.

### Token colours mirror the markdown palette

Monaco's default theme rules covered only five tokens, so semantic tokens landed
mostly on the default foreground and looked flat. `applyTheme()` now derives its
rules from the same palette `getMarkdownCodeStyles()` uses in `style-utils.js`,
so code reads identically in the editor and in rendered markdown:

| Role | Colour |
|---|---|
| comment | `fg`/`bg` mix at 45% |
| keyword, modifier, literal, built-in, tag | `accent` |
| string, regexp, escape | `green` |
| number, variable, enum member | `cyan` |
| type, class, struct, interface, enum, namespace | `blue` |
| function, method, macro | `yellow` |
| parameter | `magenta` |
| property, operator, delimiter | `fg` |

Functions and parameters were initially `fg`, matching hljs, but read as washed
out against surrounding code. Both palettes were changed together — `hljs-title`
/ `hljs-title.function_` to yellow and `hljs-params` to magenta — so the editor
and rendered markdown stay in step. `.hljs-title.class_` keeps its blue by CSS
specificity (0,2,0 beats 0,1,0).

The same rule names serve both the syntactic (TextMate-style) and semantic token
vocabularies, so one table drives both. Semantic highlighting is opt-in and must
be enabled in two places: `semanticHighlighting: true` in the theme data and
`'semanticHighlighting.enabled': true` in the editor options.

## Consequences

- Semantic tokens and inlay hints both render, with no per-token DOM cost.
- Editor and rendered-markdown code share one colour palette, so changing a theme
  colour moves both. The bespoke visual language in `LSP.md` §5.12 still does not
  apply (ADR 0017).
- The backend does *less* work than before: no decode, no absolute positions.
- **Removed as dead code:** `SemanticTokenItem` and `parseSemanticTokensResult`
  (Go), `SemanticTokenItem` (`models.ts`), `state.lspInlayHints`,
  `state.lspInlayRequestId`, the manual fetch/render plumbing in `notes.js`, and
  the `.notes-lsp-inlay-hint` CSS rules.
- Neither provider can be tested under jsdom, because Monaco is disabled there.
  The frontend test asserts the inverse — that the notes module does *not* fetch
  either payload itself.
- **Servers may need semantic tokens switched on explicitly.** gopls ships with
  them disabled and sends none until its settings enable them:

  ```yaml
  Notes:
    Languages:
      go:
        LSP:
          command: [ gopls ]
          initializationOptions:
            semanticTokens: true
  ```

  Without it every identifier falls back to the Monarch grammar, which emits only
  `identifier` for function names, parameters and variables — so code looks flat
  and no error is raised anywhere. This is a silent config dependency; it is not
  defaulted, because "no hardcoded language-server defaults" is a stated
  constraint in `ai-docs/LSP-steps.md`.
- **Lesson:** a "fetched but discarded" feature looks implemented from every
  angle except the screen. When auditing, verify something *consumes* the
  response, not merely that the request is issued.

## Reference

- `utils/lsp/semantic_tokens.go` — raw passthrough and UTF-16 re-encoding
- `utils/lsp/process.go` — `parseInitializeSemanticTokensLegend`,
  `SemanticTokensLegend()`
- `frontend/src/monaco_adapter.js` — `registerDocumentSemanticTokensProvider`,
  `registerInlayHintsProvider`, `lspGeneration`, `refreshInlayHints`
- `frontend/src/notes.js` — `semanticTokens` / `inlayHints` callbacks
