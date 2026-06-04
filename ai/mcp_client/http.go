package mcp_client

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/app"
	"golang.org/x/oauth2"
)

type OAuthConfig struct {
	ClientID              string
	ClientURI             string
	ClientSecret          string
	RedirectURI           string
	Scopes                []string
	AuthServerMetadataURL string
	PKCEEnabled           bool
}

var rxUnsafeTokenPath = regexp.MustCompile(`[^-_a-zA-Z0-9.]+`)

func ConnectHttp(overrides *mcp_config.OverrideT, url string) (*Client, error) {
	// Use the Go-SDK HTTP adapter for HTTP transport
	g, err := ConnectHttpGoSDK(overrides, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP MCP client: %w", err)
	}

	return initClientFromGoSDK(g, overrides)
}

func ConnectHttpOAuth(overrides *mcp_config.OverrideT, url string, oauthCfg OAuthConfig) (*Client, error) {
	// Build OAuth config for go-sdk
	var simple *simpleOAuthHandler
	if oauthCfg.RedirectURI != "" || len(oauthCfg.Scopes) > 0 {
		ts := NewFilePersistingTokenSource("", nil)
		simple = &simpleOAuthHandler{ts: ts}
	}

	g, err := ConnectHttpGoSDK(overrides, url, simple)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth MCP client: %w", err)
	}

	return initClientFromGoSDK(g, overrides)
}

// simpleOAuthHandler is a minimal OAuthHandler that exposes a TokenSource but
// does not perform interactive authorization itself.
type simpleOAuthHandler struct {
	ts oauth2.TokenSource
}

func (h *simpleOAuthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	return h.ts, nil
}

func (h *simpleOAuthHandler) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	return fmt.Errorf("authorization required")
}

func IsAuthorizationFailure(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "401") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "www-authenticate") ||
		strings.Contains(msg, "authorization required")
}

func DefaultRedirectURI() string {
	return "http://127.0.0.1:7700/"
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

	cacheFile := path.Join(xdg.CacheHome, app.DirName, "mcp-tokens", name+".json")
	log.Printf(`MCP OAuth: DefaultTokenFile="%s"`, cacheFile)
	return cacheFile
}
