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
- On macOS, browser windows construct `Version/<Safari version> Safari/<token>`
  as the WebKit user-agent application suffix instead of `wails.io`. Read
  `CFBundleShortVersionString` from `/Applications/Safari.app/Contents/Info.plist`
  using `plutil` once at application startup and cache the result. Propagate
  command failures rather than using a hard-coded application version.
- Keep the WebKit-generated prefix. The frozen Safari token is stored in
  `window/frontend/browser/safari.ver` (`605.1.15`), separate from the installed
  Safari application version. Main-window user agents and other platforms
  remain unchanged.

## Consequences

- Web pages opened from Notes remain in the application.
- New windows use a normal 1024-by-768 starting size and the URL host as title.
- New windows use `Window.AlwaysOnTop`, matching the main window's configuration.
- The current refactor declares `safariVersion`, but its `// embed safari.ver`
  comment is not a Go embed directive. Until wired with `//go:embed`, the Safari
  token is empty; the existing user-agent test still expects `605.1.15`.
- The new Wails binding is exposed through `WApp` and added to the generated
  bindings; frontend calls go through `wails-bindings.js`.

## Reference

- `frontend/src/markdown-utils.js` - shared Markdown link handlers
- `types/renderer.go` - desktop runtime boundary
- `window/frontend/browser/window.go` - URL validation and Wails window creation
- `window/frontend/browser/user_agent.go` - cached macOS Safari version lookup
- `window/frontend/browser/safari.ver` - frozen Safari user-agent token
- `window/frontend/main/runtime.go` - desktop runtime adapter
- `window/frontend/main/hyperlinks.go` - exported WApp binding
