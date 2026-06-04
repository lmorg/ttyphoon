# MCP Server OAuth Configuration Options

The `oauth` block inside each server entry is fully optional. If omitted entirely, OAuth is auto-detected when the server returns 401/403. All fields within it are also optional — missing values use sensible defaults.

## Full Example

```json
{
  "mcp": {
    "servers": {
      "MyServer": {
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
| `scopes` | server default | Array of OAuth scopes to request |
| `authServerMetadataUrl` | auto-discovered | Authorization server OIDC/OAuth metadata endpoint, e.g. `https://auth.example.com/.well-known/openid-configuration` |
| `pkceEnabled` | `true` | Enables PKCE (Proof Key for Code Exchange). Strongly recommended; disable only if the provider explicitly rejects it |
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
