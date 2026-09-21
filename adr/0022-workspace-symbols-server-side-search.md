# 0022 — Workspace symbols must be searched server-side

## Status

Accepted.

## Context

`Go to workspace symbol` always reported `No workspace symbols found` against
gopls.

The frontend fetched the symbol list **once with an empty query** and then
filtered it locally, using the popup menu's `showSearch` / `hideItemsUntilQuery`
support:

```js
symbols = await NotesLspWorkspaceSymbols(state.currentFile, '');
```

This assumes `workspace/symbol` with an empty query means "give me everything".
It does not. The LSP specification defines the query as the string the client is
matching on, and leaves the matching strategy to the server.

Probing gopls v0.23.0 directly over stdio against this repository:

```
query=''                   -> 0 results
query='Term'               -> 100 results
query='SetCommandCallback' -> 2 results
query='a'                  -> 100 results
```

gopls scores every candidate through its `symbolMatcher` (default `fastfuzzy`).
An empty pattern scores zero everywhere, so the result set is empty. The failure
is entirely client-side; `workspaceSymbolProvider: true` is advertised and the
server is working correctly.

The 100-result cap is the more important discovery. Even if an empty query had
returned data, fetch-once-then-filter-locally would only ever have filtered an
arbitrary truncated subset — the local search box would silently hide symbols
that exist. The server must do the searching.

No gopls setting was adopted to work around this. `symbolMatcher:
"caseInsensitive"` would make an empty query match everything, but that changes
matching semantics for every query and still hits the result cap, so it treats
the symptom.

## Decision

Search on every keystroke, server-side.

`showLocalMenu` gains an optional `onQuery(query)` provider returning
`{ options, icons }`. When present:

- The menu opens with no items and issues no request; `hideItemsUntilQuery`
  suppresses the pointless empty-query call.
- Each query re-sources the item list from the provider. The existing
  `_goFilterSeq` guard already discards responses from superseded keystrokes.
- Local `FilterStrings` matching is **skipped** — the provider owns matching.
  Filtering server results a second time would drop legitimate fuzzy matches
  that don't contain the query as a substring.
- `selectMenuItem` passes the index into the current provider result set, so the
  caller's backing array must be replaced in lockstep with each query.

Item construction was extracted into `setListItems` so the initial open and the
re-query path build rows identically.

## Consequences

- Workspace symbol search now works, and is not limited by the server result cap
  in a misleading way — narrowing the query narrows the server's own search.
- Any menu needing server-side search can now use `onQuery` instead of
  pre-loading a full list.
- Providers are called per keystroke. gopls answers from an in-memory index so
  this is cheap; a slower provider would want debouncing.

## Lessons

- **"Empty query" is not a wildcard in LSP.** The server decides what matches;
  an empty pattern legitimately matches nothing.
- **A result cap invalidates client-side filtering entirely.** Once a server
  truncates, filtering its response locally searches a subset while appearing to
  search everything — silently wrong rather than visibly broken.
- **Probe the server before blaming its configuration.** Twenty lines of stdio
  JSON-RPC distinguished "gopls needs a flag" from "we send the wrong request",
  and pointed at the caller. Contrast ADR 0018, where a near-identical symptom
  (a feature producing nothing) *was* a missing gopls setting — the symptom does
  not identify the culprit.

## Reference

- `frontend/src/popup_menu.js` — `onQuery`, `dynamicQuery`, `setListItems`,
  `menuQuery`
- `frontend/src/notes.js` — `goToWorkspaceLspSymbol`
- `utils/lsp/symbols.go` — `RequestWorkspaceSymbols`
- ADR 0018 — semantic tokens, where the cause *was* server configuration
