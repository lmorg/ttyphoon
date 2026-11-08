# MCP Server Configuration Options

## OAuth Login Window

MCP OAuth login opens in a new, focused built-in browser window by default.
It reuses the URL windows opened by middle-clicking Notes Markdown links;
each login opens a separate window, and multiple URL windows can coexist.
These windows also respect `Window.AlwaysOnTop`.
After successful authorization and token exchange, the login window closes
automatically. Failed or cancelled logins stay open; unrelated URL windows and
external-browser tabs are not closed.

To use your external browser instead, add this to your ttyphoon YAML config:

```yaml
AI:
  OAuthUseBuiltinWebview: false
```

The Go setting is `config.Config.Ai.OAuthUseBuiltinWebview`. The built-in
webview does not share your external browser's login session. Some identity
providers reject embedded login windows; select the external browser for those
providers. The selection is explicit: failures do not silently switch browsers.
OAuth callback validation and token storage are unchanged.

## Descriptions and On-Demand Startup

Each server accepts an optional `description` explaining what its tools are useful
for. The AI sees the configured server names and descriptions, not their commands,
environment variables, URLs, or OAuth settings.

The built-in `startMcpServer` tool is enabled by default and asks for permission
before starting a server. Its approval prompt includes the server name and the
AI's justification for needing it. It can only start servers already present in
ttyphoon's MCP configuration files; it cannot supply new commands or URLs.
After startup, the server's tools are available within the same AI task.

Startup permission is separate from permissions for the tools the server exposes.
The usual per-invocation/per-prompt choices apply, and AI settings can override the
startup tool's default approval policy. Startup is not available to sub-agents.

## Tool Permission Defaults

Each server can configure defaults for every tool it exposes:

```json
"DefaultPermissions": {
  "invocation": "alwaysAllow",
  "subagents": "allow"
}
```

`invocation` accepts `"alwaysAllow"` or `"askPermission"`; `subagents`
accepts `"allow"` or `"deny"`. Omitted values use ttyphoon's defaults:
always allow invocation and allow sub-agent access. Changes made in AI settings
override these defaults for the active agent.

The `oauth` block inside each server entry is fully optional. If omitted entirely, OAuth is auto-detected when the server returns 401/403. All fields within it are also optional — missing values use sensible defaults.

## Full Example

```json
{
  "mcp": {
    "servers": {
      "MyServer": {
        "description": "Search project tickets and retrieve their acceptance criteria",
        "type": "http",
        "url": "https://mcp.example.com/v1/mcp",
        "oauth": {
          "clientId":              "your-client-id",
          "clientSecret":          "your-client-secret",
          "clientUri":             "https://your-client-metadata-doc-url",
          "redirectUri":           "http://127.0.0.1:7700/",
          "scopes":                ["openid", "offline_access"],
          "authServerMetadataUrl": "https://auth.example.com/.well-known/openid-configuration",
          "pkceEnabled":           true,
          "tokenFile":             "/path/to/token.json"
        }
      }
    }
  }
}
```

## Field Reference

| JSON key | Default | Description |
|---|---|---|
| `clientId` | auto | OAuth client identifier for pre-registered apps |
| `clientSecret` | auto | Client secret for confidential clients |
| `clientUri` | — | Provider-issued client metadata document URL; enables Client ID Metadata Document registration flow |
| `redirectUri` | `http://127.0.0.1:7700/` | OAuth redirect URL — must match what you registered with the provider |
| `scopes` | server-derived | Fallback OAuth scopes. Used only when the server advertises none (via the WWW-Authenticate challenge or Protected Resource Metadata `scopes_supported`); they never override server-advertised scopes |
| `authServerMetadataUrl` | auto-discovered | Authorization server OIDC/OAuth metadata endpoint, e.g. `https://auth.example.com/.well-known/openid-configuration` |
| `pkceEnabled` | `true` (forced) | Deprecated and ignored. PKCE (S256) is always enforced per the MCP authorization spec and cannot be disabled |
| `tokenFile` | `$XDG_CACHE_HOME/ttyphoon/mcp-tokens/{server}.json` | Path to persist OAuth tokens across sessions |

## Client Registration Methods

The client will attempt registration in the following priority order:

1. **Client ID Metadata Document** — used when `clientUri` is set (or `clientId` is an HTTPS URL). Requires the provider to support this flow.
2. **Pre-registered client** — used when `clientId` (and optionally `clientSecret`) are set.
3. **Dynamic client registration** — used as fallback when no explicit credentials are provided. Requires the provider to support RFC 7591.

If the provider rejects all available methods, the error message will list which methods were attempted and suggest next steps.

## Provider-Specific Examples

### Atlassian Rovo MCP

```json
"oauth": {
  "clientId":              "your-atlassian-client-id",
  "clientSecret":          "your-atlassian-client-secret",
  "authServerMetadataUrl": "https://auth.atlassian.com/.well-known/openid-configuration"
}
```

### Datadog MCP (auto-detected, no config required)

```json
"oauth": {}
```

Or omit the `oauth` block entirely.
