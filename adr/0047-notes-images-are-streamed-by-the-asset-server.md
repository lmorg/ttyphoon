# 0047 - Notes images are streamed by the asset server

## Status

Accepted.

## Context

Every image rendered in Notes View, Run, the Markdown modal and the AI panel was
loaded by calling the bound `GetImage` method. That method read the file, base64
encoded it and returned it as a Go string. The string crossed the JS bridge as a
JSON-marshalled IPC reply, which on macOS is delivered through
`evaluateJavaScript` on the **main thread** (ADR 0020, theory 6).

The cost was paid three times over:

- Go allocated roughly 3.7x the file size (bytes, base64 string, JSON escaping).
- The encoded payload was marshalled through the fixed 100-slot `processMessage`
  channel that the main thread also drains.
- The webview decoded a multi-megabyte data URL before it could paint.

`processWailsImages` also awaited each image **sequentially**, so a note with ten
screenshots serialised ten full round trips before anything rendered. A single
large image was enough to stall the UI, and this was a recurring contributor to
the hangs tracked in ADR 0020.

The old resolution path was also unsound. `rxExtension` is
`regexp.MustCompile(`.[a-zA-Z0-9]+$`)` — the dot is unescaped, so it matches any
character and `/etc/passwd` passes as a valid "extension". The fallback in
`resolveMarkdownAssetPath` strips leading separators but does nothing about
`../`, so `GetImage` would read arbitrary files given a crafted Markdown link.

## Decision

Serve images over the Wails asset server instead of the JS bridge.

- Register a `Handler` on `assetserver.Options`. Wails invokes it only for paths
  absent from the embedded `Assets`, so the bundle still wins for real assets.
- Images resolve to `/__asset?path=<encoded>`, which the webview loads from its
  own origin. `http.ServeFile` supplies Content-Type, Range, `ETag`/
  `Last-Modified` and `304` handling.
- Send `Cache-Control: no-cache`. The URL is stable across edits, so the webview
  must revalidate rather than serve a stale image after a file changes.
- Restrict the handler to `GET`/`HEAD`.
- `resolveNotesAsset` replaces the broken extension check with an explicit
  allowlist of image extensions, then requires the fully resolved path — cleaned
  and with symlinks evaluated — to sit inside the Markdown base directory, the
  project root, the user notes directory, the global notes directory or
  `~/Documents/ttyphoon`. Anything else is a `404`.
- `mdBaseDir` is now guarded by a `sync.RWMutex`. The handler runs on asset
  server goroutines while `GetFile` writes the field.
- `processWailsImages` becomes synchronous and performs no IPC. It marks each
  rewritten node with `data-asset-resolved` because a rewritten `src` still
  matches the `wails://` pattern on Windows, and a second pass would otherwise
  double-encode the path.
- The image context menu no longer fetches bytes up front. It resolves a data URL
  lazily inside `onSelect`, so opening the menu costs nothing and a failed read
  no longer suppresses the menu entirely.

## Consequences

- Images stream on the webview's own network path. No file contents cross the
  bridge, and the main-thread IPC channel is no longer a bottleneck for Notes.
- Images load in parallel and progressively, because the browser owns the
  requests rather than a sequential `await` loop.
- No size cap is needed. Streaming makes large files a bandwidth question rather
  than an allocation spike.
- **Behavioural change:** images resolving outside the five permitted roots now
  return `404` where `GetImage` previously read them from anywhere on disk.
- Path traversal and non-image reads are rejected on the new endpoint. The legacy
  `GetImage` binding is retained but is no longer on any hot path; its unsound
  `rxExtension` check remains and should be removed with the method.
- The context menu costs one fetch per invoked action rather than one per
  right-click. The bytes come from the local asset server, not the bridge.

## Reference

- `frontend.go` - `notesAssetHandler`, `resolveNotesAsset`, `pathWithinRoot`
- `frontend_assets_test.go` - containment, extension and method coverage
- `frontend/src/markdown-utils.js` - `notesAssetURL`, `processWailsImages`
- `frontend/src/notes.js` - image viewer and lazy context-menu byte loading
- ADR 0020 - sleep/wake hang hardening, theory 6 (main-thread IPC delivery)
- ADR 0046 - Markdown images are container-capped
