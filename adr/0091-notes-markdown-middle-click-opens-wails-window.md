# 0091 - Open Notes Markdown web links in a Wails window on middle-click

Date: 2026-11-05

## Status

Accepted and implemented

## Context

Notes Markdown links already use a shared handler for regular and
auto-detected links. Middle-click previously sent links to the configured
default opener, outside the application.

## Decision

- Middle-clicking or Command-clicking an HTTP(S) link rendered in Notes Markdown
  opens it in a new Wails webview window and focuses that window.
- Validate the URL at the desktop runtime boundary and allow only absolute
  HTTP(S) URLs.
- Preserve ordinary left-click behavior and the configured default opener for other
  schemes, including `mailto:` and file links.
- Do not change terminal hyperlink behavior.

## Consequences

- Web pages opened from Notes remain in the application.
- New windows use a normal 1024-by-768 starting size and the URL host as title.
- New windows use `Window.AlwaysOnTop`, matching the main window's configuration.
- The new Wails binding is exposed through `WApp` and added to the generated
  bindings; frontend calls go through `wails-bindings.js`.

## Reference

- `frontend/src/markdown-utils.js` - shared Markdown link handlers
- `types/renderer.go` - desktop runtime boundary
- `window/frontend/browser/window.go` - URL validation and Wails window creation
- `window/frontend/main/runtime.go` - desktop runtime adapter
- `window/frontend/main/hyperlinks.go` - exported WApp binding
