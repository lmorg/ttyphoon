package mcp_client

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/app"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
)

type OAuthConfig struct {
	ClientID              string
	ClientSecret          string
	RedirectURI           string
	Scopes                []string
	AuthServerMetadataURL string
	PKCEEnabled           bool
	TokenStore            client.TokenStore
}

// Name:   Visual Studio Code
// Website:  https://code.visualstudio.com
/*
redirect urls:
https://insiders.vscode.dev/redirect
https://vscode.dev/redirect
http://127.0.0.1/
http://127.0.0.1:33418/
*/
type TokenStore = client.TokenStore
type Token = client.Token
type MemoryTokenStore = client.MemoryTokenStore

var rxUnsafeTokenPath = regexp.MustCompile(`[^-_a-zA-Z0-9.]+`)

func NewMemoryTokenStore() *MemoryTokenStore {
	return client.NewMemoryTokenStore()
}

func ConnectHttp(overrides *mcp_config.OverrideT, url string) (*Client, error) {
	c, err := client.NewStreamableHttpClient(url)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return initClient(c, overrides)
}

func ConnectHttpOAuth(overrides *mcp_config.OverrideT, url string, oauthCfg OAuthConfig) (*Client, error) {
	cfg := client.OAuthConfig{
		ClientID:              oauthCfg.ClientID,
		ClientSecret:          oauthCfg.ClientSecret,
		RedirectURI:           oauthCfg.RedirectURI,
		Scopes:                oauthCfg.Scopes,
		TokenStore:            oauthCfg.TokenStore,
		AuthServerMetadataURL: oauthCfg.AuthServerMetadataURL,
		PKCEEnabled:           oauthCfg.PKCEEnabled,
	}

	c, err := client.NewOAuthStreamableHttpClient(url, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth client: %w", err)
	}

	return initClient(c, overrides)
}

func IsOAuthAuthorizationRequiredError(err error) bool {
	return client.IsOAuthAuthorizationRequiredError(err)
}

func GetOAuthHandler(err error) *transport.OAuthHandler {
	return client.GetOAuthHandler(err)
}

func GenerateCodeVerifier() (string, error) {
	return client.GenerateCodeVerifier()
}

func GenerateCodeChallenge(verifier string) string {
	return client.GenerateCodeChallenge(verifier)
}

func GenerateState() (string, error) {
	return client.GenerateState()
}

func IsAuthorizationFailure(err error) bool {
	if err == nil {
		return false
	}
	if IsOAuthAuthorizationRequiredError(err) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "401") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "www-authenticate") ||
		strings.Contains(msg, "authorization required")
}

func DefaultRedirectURI() string {
	return "http://127.0.0.1:38085/oauth/callback"
}

func DefaultTokenFile(serverName, rawURL string) string {
	host := serverName
	if u, err := url.Parse(rawURL); err == nil {
		host = u.Hostname()
		if host == "" {
			host = serverName
		}
		if p := strings.Trim(strings.Trim(path.Clean(u.Path), "/"), "."); p != "" && p != "/" {
			host += "-" + strings.ReplaceAll(p, "/", "-")
		}
	}

	name := strings.Trim(rxUnsafeTokenPath.ReplaceAllString(host, "-"), "-")
	if name == "" {
		name = "default"
	}

	return path.Join(xdg.CacheHome, app.DirName, "mcp-tokens", name+".json")
}
