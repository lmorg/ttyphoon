# MCP OAuth login window

## Status

Accepted and implemented.

## Context

MCP authorization previously always launched an external browser. The existing
Notes URL-window implementation can create multiple independent, focused
windows, so it can also host an OAuth authorization page.

## Decision

- Add `config.Config.Ai.OAuthUseBuiltinWebview`, exposed as
  `AI.OAuthUseBuiltinWebview` in YAML, with `true` in the embedded defaults.
- When enabled, open each authorization URL through the desktop runtime's
  `OpenURLInNewWindow`, reusing `window/frontend/browser`. Do not reuse or
  navigate an existing login window.
- When disabled, preserve external-browser authorization.
- Keep OAuth transport logic independent of window implementation via its
  opener hook. Return opener errors immediately rather than waiting for a
  callback that cannot arrive or silently falling back to another browser.
- Preserve loopback callback handling, state/issuer validation, PKCE, token
  persistence, window focus, and configured always-on-top behavior.
- Return a per-window close function for managed URL windows. After the OAuth
  handler completes authorization and token exchange successfully, close only
  built-in windows opened by that authorization flow. Do not close windows on
  callback arrival alone, failed exchange, denial, or cancellation. External
  browser tabs and unrelated Notes/login windows are never closed.

## Consequences

Several URL/login windows may coexist. Windows remain independently closeable;
successful authorization automatically closes the owning built-in login window.
Closing a login window does not itself cancel startup: the existing task
cancellation and callback timeout still apply.

Built-in webviews have separate browser sessions, and some identity providers
disallow embedded user agents. Users can select the external browser for
compatibility. Concurrent authorizations still need distinct available callback
ports; independent windows do not remove that existing constraint.

## Reference

- [Notes URL windows](0091-notes-markdown-middle-click-opens-wails-window.md)
- [Window packages](0092-wails-window-packages-and-feature-bindings.md)
- [MCP OAuth lifecycle](0078-mcp-oauth-failures-must-be-retryable.md)
- [MCP configuration](../ai/mcp_config/options.md)
