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

// authFailureRx matches how the MCP transport signals that authorization is
// required. The go-sdk (v1.6.1) does not expose a typed HTTP-status error for
// connect/operation failures: a 401/403 without an OAuth handler surfaces as a
// plain fmt.Errorf carrying http.StatusText (e.g. "...: Unauthorized"). We
// therefore match the canonical 401/403 status texts and the WWW-Authenticate
// challenge header as whole words — as precise as the SDK allows. The bare
// status numbers and free-form phrases were intentionally dropped to reduce the
// chance that a downstream tool error (rather than the MCP connection itself) is
// misread as an authorization failure.
var authFailureRx = regexp.MustCompile(`(?i)\b(unauthorized|forbidden|www-authenticate)\b`)

func IsAuthorizationFailure(err error) bool {
	if err == nil {
		return false
	}
	return authFailureRx.MatchString(err.Error())
}

func DefaultRedirectURI() string {
	return "http://127.0.0.1:7700/"
}

func DefaultClientURI() string {
	// Disabled by default. Some providers can constrain granted capabilities
	// when a fixed metadata document is always supplied.
	return ""
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
