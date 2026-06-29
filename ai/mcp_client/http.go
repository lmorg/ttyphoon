package mcp_client

import (
	"fmt"
	"log"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/app"
)

type OAuthConfig struct {
	ClientID              string
	ClientURI             string
	ClientSecret          string
	RedirectURI           string
	Scopes                []string
	AuthServerMetadataURL string
	PKCEEnabled           bool
	TokenFile             string
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

func DefaultClientURI() string {
	return "https://ttyphoon.com/oauth/client-metadata.json"
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
