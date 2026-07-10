package mcp_config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type ConfigT struct {
	Mcp struct {
		Servers ServersT `json:"servers"`
		Inputs  InputsT  `json:"inputs"`
	} `json:"mcp"`
	McpServers *ServersT `json:"mcp.servers"`
	Source     string
}

type ServersT map[string]ServerT

type ServerT struct {
	// Command is the executable name for stdio transport (type: "command")
	Command string `json:"command"`
	// Args are arguments to pass to the command
	Args []string `json:"args"`
	// Env are environment variables to set for the command
	Env EnvVarsT `json:"env"`
	// Type is "command" for stdio or "http" for HTTP/HTTPS
	Type string `json:"type"`
	// Url is the HTTP endpoint for HTTP servers
	Url string `json:"url"`
	// OAuth is optional. If omitted, the client will attempt non-OAuth connection first,
	// and if that fails with 401/403, will trigger interactive OAuth with defaults.
	// OAuth can be partially specified to override specific aspects (scopes, custom redirectUri, etc.)
	OAuth    *OAuthT   `json:"oauth,omitempty"`
	Override OverrideT `json:"override"`
}

// OAuthT defines OAuth 2.0 configuration for a server.
// All fields are optional:
//   - If omitted entirely, OAuth is auto-detected on 401/403 errors
//   - If partially specified, missing fields are auto-populated with sensible defaults
//
// Deprecated fields (enabled, clientId, clientSecret) are still accepted for backward compatibility
// but are no longer required since the client can dynamically register or auto-detect OAuth requirements.
type OAuthT struct {
	// Enabled is deprecated. OAuth is now auto-detected. Kept for backward compatibility.
	Enabled bool `json:"enabled"`
	// ClientID is the OAuth client identifier. If omitted, dynamic registration will be used.
	ClientID string `json:"clientId"`
	// ClientURI is reserved for future use.
	ClientURI string `json:"clientUri"`
	// ClientSecret is the OAuth client secret. If omitted, dynamic registration will be used.
	// For backward compatibility, public clients may omit this.
	ClientSecret string `json:"clientSecret"`
	// RedirectURI customizes the OAuth redirect URL (default: http://127.0.0.1:7700/)
	RedirectURI string `json:"redirectUri"`
	// Scopes are OAuth scopes used only as a FALLBACK. The MCP SDK derives scopes
	// from the server (the WWW-Authenticate challenge, or Protected Resource
	// Metadata scopes_supported). These configured scopes are injected into the
	// authorization request ONLY when the server advertises none; they never
	// override server-advertised scopes.
	Scopes []string `json:"scopes"`
	// AuthServerMetadataURL is the authorization server metadata endpoint for OIDC discovery
	// (default: auto-discovered from server's .well-known/openid-configuration or oauth-authorization-server)
	AuthServerMetadataURL string `json:"authServerMetadataUrl"`
	// PKCEEnabled is deprecated and ignored. PKCE (S256) is always enforced by the
	// underlying MCP SDK per the MCP authorization spec and cannot be disabled;
	// this field is retained only so existing configs continue to parse.
	PKCEEnabled bool `json:"pkceEnabled"`
	// TokenFile is the path to store and retrieve OAuth tokens
	// (default: $XDG_CACHE_HOME/ttyphoon/mcp-tokens/{server}.json)
	TokenFile string `json:"tokenFile"`
}

type OverrideT struct {
	AppName string `json:"appName"`
	WebSite string `json:"webSite"`
}

type EnvVarsT map[string]string

func (env EnvVarsT) Slice() []string {
	var envvars []string
	for k, v := range env {
		envvars = append(envvars, fmt.Sprintf("%s=%s", k, v))
	}
	return envvars
}

// IsOAuthConfigured returns true if the OAuth section is present and any field is set.
func (o *OAuthT) IsOAuthConfigured() bool {
	if o == nil {
		return false
	}
	return o.Enabled ||
		o.ClientID != "" ||
		o.ClientSecret != "" ||
		o.RedirectURI != "" ||
		len(o.Scopes) > 0 ||
		o.AuthServerMetadataURL != "" ||
		o.TokenFile != ""
}

func ReadJson(filename string) (*ConfigT, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	config := new(ConfigT)
	config.McpServers = &config.Mcp.Servers
	err = json.Unmarshal(b, config)
	return config, err
}
