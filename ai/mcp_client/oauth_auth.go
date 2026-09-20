package mcp_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/app"

	authsdk "github.com/modelcontextprotocol/go-sdk/auth"
	oauthex "github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
)

type OAuthUIHooks struct {
	OpenBrowser               func(string)
	PromptCallbackURL         func() (string, error)
	OnAutoCallbackUnavailable func(error)
}

func ConnectAndUseHttp(overrides *mcp_config.OverrideT, workspace, server, serverURL string, oauth *mcp_config.OAuthT, hooks OAuthUIHooks, onOAuthRequired func(), useClient func(*Client) error) error {
	log.Printf("[debug] MCP OAuth: start ConnectAndUseHttp server=%q mcp_url=%q oauth_configured=%t", server, sanitizeURLForLog(serverURL), oauth != nil)
	if oauth != nil {
		log.Printf("[debug] MCP OAuth: provided config redirect_uri=%q metadata_url=%q scopes=%d client_id_set=%t client_secret_set=%t",
			oauth.RedirectURI,
			sanitizeURLForLog(oauth.AuthServerMetadataURL),
			len(oauth.Scopes),
			oauth.ClientID != "",
			oauth.ClientSecret != "",
		)
	}

	c, err := ConnectHttp(overrides, serverURL)
	if err == nil {
		log.Printf("[debug] MCP OAuth: initial HTTP connect succeeded without OAuth")
		if useClient == nil {
			return nil
		}

		err = useClient(c)
		if err == nil {
			log.Printf("[debug] MCP OAuth: initial operation succeeded without OAuth")
			return nil
		}
		if !IsAuthorizationFailure(err) {
			log.Printf("[debug] MCP OAuth: initial operation failed with non-auth error: %v", err)
			return err
		}
		log.Printf("[debug] MCP OAuth: initial operation indicates OAuth required: %v", err)
	} else if !IsAuthorizationFailure(err) {
		log.Printf("[error] MCP OAuth: initial connect failed with non-auth error: %v", err)
		return err
	} else {
		log.Printf("[debug] MCP OAuth: initial connect indicates OAuth required: %v", err)
	}

	if onOAuthRequired != nil {
		onOAuthRequired()
	}

	oauthCfg := BuildOAuthConfig(workspace, server, serverURL, oauth)
	log.Printf("[debug] MCP OAuth: interactive config redirect_uri=%q metadata_url=%q scopes=%d client_id_set=%t client_secret_set=%t",
		oauthCfg.RedirectURI,
		sanitizeURLForLog(oauthCfg.AuthServerMetadataURL),
		len(oauthCfg.Scopes),
		oauthCfg.ClientID != "",
		oauthCfg.ClientSecret != "",
	)

	c, err = ConnectHttpOAuthInteractive(overrides, serverURL, oauthCfg, hooks)
	if err != nil {
		log.Printf("[error] MCP OAuth: interactive connect failed: %v", err)
		return err
	}
	log.Printf("[debug] MCP OAuth: interactive connect succeeded")

	if useClient == nil {
		return nil
	}

	// The interactive client carries an OAuthHandler, so the go-sdk transport
	// already performs the Authorize + single-retry dance internally on any
	// 401/403 seen during this operation. A further outer retry here would only
	// re-run the entire interactive flow (re-opening the browser) for a request
	// the SDK has already failed to authorize, so we surface the error instead.
	if err := useClient(c); err != nil {
		if IsAuthorizationFailure(err) {
			log.Printf("[error] MCP OAuth: operation still unauthorized after interactive auth (SDK already retried): %v", err)
		}
		return err
	}
	return nil
}

func BuildOAuthConfig(workspace, server, serverURL string, oauth *mcp_config.OAuthT) OAuthConfig {
	redirectURI := DefaultRedirectURI()
	clientURI := ""
	var clientID, clientSecret, authServerMetadataURL string
	var scopes []string

	if oauth != nil {
		if oauth.RedirectURI != "" {
			redirectURI = oauth.RedirectURI
		}
		clientID = oauth.ClientID
		if oauth.ClientURI != "" {
			clientURI = oauth.ClientURI
		}
		clientSecret = oauth.ClientSecret
		scopes = oauth.Scopes
		authServerMetadataURL = oauth.AuthServerMetadataURL
	}

	tokenFile := DefaultTokenFile(workspace, server, serverURL)
	if oauth != nil && oauth.TokenFile != "" {
		tokenFile = oauth.TokenFile
	}

	cfg := OAuthConfig{
		RedirectURI:           redirectURI,
		ClientID:              clientID,
		ClientURI:             clientURI,
		ClientSecret:          clientSecret,
		Scopes:                scopes,
		AuthServerMetadataURL: authServerMetadataURL,
		TokenFile:             tokenFile,
	}

	mcpHost := hostFromURL(serverURL)
	authHost := hostFromURL(authServerMetadataURL)
	redirectHost := hostFromURL(redirectURI)
	log.Printf("[debug] MCP OAuth: BuildOAuthConfig server=%q mcp_host=%q auth_metadata_host=%q redirect_host=%q", server, mcpHost, authHost, redirectHost)
	if cfg.ClientURI != "" {
		log.Printf("[debug] MCP OAuth: BuildOAuthConfig client_uri=%q", sanitizeURLForLog(cfg.ClientURI))
	}
	if len(cfg.Scopes) > 0 {
		log.Printf("[debug] MCP OAuth: BuildOAuthConfig default/requested scopes=%v", cfg.Scopes)
	}
	if mcpHost != "" && authHost != "" && mcpHost != authHost {
		log.Printf("[debug] MCP OAuth: info MCP host differs from Auth host (often valid): mcp=%q auth=%q", mcpHost, authHost)
	}

	return cfg
}

func ConnectHttpOAuthInteractive(overrides *mcp_config.OverrideT, serverURL string, oauthCfg OAuthConfig, hooks OAuthUIHooks) (*Client, error) {
	log.Printf("[debug] MCP OAuth: ConnectHttpOAuthInteractive mcp_url=%q redirect_uri=%q metadata_url=%q", sanitizeURLForLog(serverURL), oauthCfg.RedirectURI, sanitizeURLForLog(oauthCfg.AuthServerMetadataURL))
	log.Printf("[debug] MCP OAuth: configured registration methods=%s", oauthRegistrationMethodsSummary(oauthCfg))

	// Try Go-SDK AuthorizationCodeHandler-backed flow using UI hooks
	if handler, hErr := buildGoSDKAuthHandler(oauthCfg, overrides, hooks); hErr == nil && handler != nil {
		log.Printf("[debug] MCP OAuth: authorization handler created")
		if g, connErr := ConnectHttpGoSDK(overrides, serverURL, handler); connErr == nil {
			log.Printf("[debug] MCP OAuth: OAuth HTTP connect succeeded")
			return initClientFromGoSDK(g, overrides)
		} else {
			enriched := enrichOAuthConnectError(connErr, oauthCfg)
			log.Printf("[error] MCP OAuth: OAuth HTTP connect failed: %v", enriched)
			return nil, fmt.Errorf("failed to establish OAuth connection to %s: %w", serverURL, enriched)
		}
	} else {
		log.Printf("[error] MCP OAuth: failed creating authorization handler: %v", hErr)
		return nil, fmt.Errorf("failed to create OAuth authorization handler: %w", hErr)
	}
}

// buildGoSDKAuthHandler constructs an auth.OAuthHandler backed by the
// go-sdk AuthorizationCodeHandler using provided OAuth config and UI hooks.
func buildGoSDKAuthHandler(oauthCfg OAuthConfig, overrides *mcp_config.OverrideT, hooks OAuthUIHooks) (authsdk.OAuthHandler, error) {
	if oauthCfg.RedirectURI == "" {
		return nil, fmt.Errorf("redirect URI required for interactive OAuth")
	}

	log.Printf("[debug] MCP OAuth: buildGoSDKAuthHandler redirect_host=%q auth_metadata_host=%q", hostFromURL(oauthCfg.RedirectURI), hostFromURL(oauthCfg.AuthServerMetadataURL))

	fetcher := func(ctx context.Context, args *authsdk.AuthorizationArgs) (*authsdk.AuthorizationResult, error) {
		authURL := ensureAuthorizationRequestScopes(args.URL, oauthCfg.Scopes)
		logOAuthAuthorizeRequest(authURL)

		// Try automatic callback server first
		callback, err := StartOAuthCallbackServer(oauthCfg.RedirectURI)
		if err == nil {
			log.Printf("[debug] MCP OAuth: automatic callback server started redirect_uri=%q", oauthCfg.RedirectURI)
			defer callback.Close()
			if hooks.OpenBrowser != nil {
				log.Printf("[debug] MCP OAuth: opening browser for authorization")
				hooks.OpenBrowser(authURL)
			}
			log.Printf("[debug] MCP OAuth: waiting for automatic callback")
			res, waitErr := callback.Wait(2 * time.Minute)
			if waitErr != nil {
				log.Printf("[error] MCP OAuth: automatic callback wait failed: %v", waitErr)
				return nil, waitErr
			}
			log.Printf("[debug] MCP OAuth: automatic callback received state_present=%t code_len=%d", res.State != "", len(res.Code))
			return &authsdk.AuthorizationResult{Code: res.Code, State: res.State}, nil
		}
		log.Printf("[error] MCP OAuth: automatic callback unavailable: %v", err)

		if hooks.OnAutoCallbackUnavailable != nil {
			hooks.OnAutoCallbackUnavailable(err)
		}

		// Fallback to pasted callback URL
		if hooks.PromptCallbackURL == nil {
			log.Printf("[error] MCP OAuth: no PromptCallbackURL hook available")
			return nil, fmt.Errorf("no method to obtain callback URL")
		}
		if hooks.OpenBrowser != nil {
			log.Printf("[debug] MCP OAuth: opening browser for authorization (manual callback mode)")
			hooks.OpenBrowser(authURL)
		}
		raw, pErr := hooks.PromptCallbackURL()
		if pErr != nil {
			log.Printf("[debug] MCP OAuth: PromptCallbackURL failed: %v", pErr)
			return nil, pErr
		}
		log.Printf("[debug] MCP OAuth: manual callback URL received callback_host=%q callback_url=%q", hostFromURL(raw), sanitizeURLForLog(raw))
		code, state, parseErr := parseOAuthCallbackURL(raw)
		if parseErr != nil {
			log.Printf("[debug] MCP OAuth: manual callback parse failed: %v", parseErr)
			return nil, parseErr
		}
		log.Printf("[debug] MCP OAuth: manual callback parsed state_present=%t code_len=%d", state != "", len(code))
		return &authsdk.AuthorizationResult{Code: code, State: state}, nil
	}

	cfg := &authsdk.AuthorizationCodeHandlerConfig{
		RedirectURL:              oauthCfg.RedirectURI,
		AuthorizationCodeFetcher: fetcher,
		Client:                   newOAuthLoggingClient(),
	}

	clientIDMetadataURL := strings.TrimSpace(oauthCfg.ClientURI)
	if clientIDMetadataURL == "" && strings.HasPrefix(strings.TrimSpace(oauthCfg.ClientID), "https://") {
		// Some providers represent the client ID metadata document URL via client_id.
		clientIDMetadataURL = strings.TrimSpace(oauthCfg.ClientID)
	}
	if clientIDMetadataURL != "" {
		log.Printf("[debug] MCP OAuth: enabling Client ID Metadata Document registration url=%q", sanitizeURLForLog(clientIDMetadataURL))
		cfg.ClientIDMetadataDocumentConfig = &authsdk.ClientIDMetadataDocumentConfig{URL: clientIDMetadataURL}
	}

	// client registration: prefer preregistered client if provided
	if oauthCfg.ClientID != "" && clientIDMetadataURL != oauthCfg.ClientID {
		log.Printf("[debug] MCP OAuth: using preregistered client client_secret_set=%t", oauthCfg.ClientSecret != "")
		cfg.PreregisteredClient = &oauthex.ClientCredentials{
			ClientID: oauthCfg.ClientID,
		}
		if oauthCfg.ClientSecret != "" {
			cfg.PreregisteredClient.ClientSecretAuth = &oauthex.ClientSecretAuth{ClientSecret: oauthCfg.ClientSecret}
		}
	}

	// Use dynamic registration whenever we are not using explicit pre-registered credentials.
	if cfg.PreregisteredClient == nil {
		log.Printf("[debug] MCP OAuth: using dynamic client registration")
		cfg.DynamicClientRegistrationConfig = &authsdk.DynamicClientRegistrationConfig{
			Metadata: buildDynamicClientRegistrationMetadata(oauthCfg),
		}
	} else if cfg.DynamicClientRegistrationConfig == nil {
		log.Printf("[error] MCP OAuth: dynamic client registration disabled due to explicit client credentials")
	}

	if cfg.ClientIDMetadataDocumentConfig == nil && cfg.PreregisteredClient == nil && cfg.DynamicClientRegistrationConfig == nil {
		log.Printf("[warn] MCP OAuth: warning no client registration method configured")
	}

	h, err := authsdk.NewAuthorizationCodeHandler(cfg)
	if err != nil {
		log.Printf("[error] MCP OAuth: NewAuthorizationCodeHandler failed: %v", err)
		return nil, err
	}
	log.Printf("[debug] MCP OAuth: NewAuthorizationCodeHandler created")

	// Wrap the handler to persist tokens to file
	return &gosdkOAuthHandler{inner: h, tokenFile: oauthCfg.TokenFile}, nil
}

func ensureAuthorizationRequestScopes(rawURL string, scopes []string) string {
	if strings.TrimSpace(rawURL) == "" || len(scopes) == 0 {
		return rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil || u == nil {
		return rawURL
	}

	q := u.Query()
	if strings.TrimSpace(q.Get("scope")) != "" {
		return rawURL
	}

	q.Set("scope", strings.Join(scopes, " "))
	u.RawQuery = q.Encode()
	patched := u.String()
	log.Printf("[debug] MCP OAuth: injected scopes into authorization request scope=%q", strings.Join(scopes, " "))
	return patched
}

func buildDynamicClientRegistrationMetadata(oauthCfg OAuthConfig) *oauthex.ClientRegistrationMetadata {
	metadata := &oauthex.ClientRegistrationMetadata{
		RedirectURIs: []string{oauthCfg.RedirectURI},
		ClientName:   app.Name(),
	}
	if len(oauthCfg.Scopes) > 0 {
		metadata.Scope = strings.Join(oauthCfg.Scopes, " ")
	}
	return metadata
}

// gosdkOAuthHandler wraps an auth.AuthorizationCodeHandler and persists tokens
// to a file when TokenSource is requested.
type gosdkOAuthHandler struct {
	inner                 *authsdk.AuthorizationCodeHandler
	tokenFile             string
	prmMu                 sync.Mutex
	prmMetadataBaseURL    string
	prmMetadataServer     *http.Server
	prmMetadataServerAddr string
}

func (g *gosdkOAuthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	log.Printf("MCP OAuth: TokenSource requested token_file_set=%t", g.tokenFile != "")
	base, err := g.inner.TokenSource(ctx)
	if err != nil {
		log.Printf("[debug] MCP OAuth: inner TokenSource failed: %v", err)
		return nil, err
	}
	if g.tokenFile == "" {
		log.Printf("[debug] MCP OAuth: TokenSource ready without persistence")
		return base, nil
	}

	// Before the interactive flow completes the inner handler has no in-memory
	// token and returns a nil source. We must not substitute an erroring source
	// here: the transport relies on a nil TokenSource to send the unauthenticated
	// request, receive a 401, and trigger Authorize(). Returning an error from
	// Token() at this stage aborts the request before authorization can start.
	if base == nil {
		// Reuse a previously persisted, still-valid token so a restart can skip
		// re-authorization. If none exists (or it has expired and cannot be
		// refreshed here), fall through to nil and let the interactive flow run.
		if tok, rerr := readTokenFile(g.tokenFile); rerr == nil && tok.Valid() {
			log.Printf("[debug] MCP OAuth: reusing persisted token token_file=%q", g.tokenFile)
			return oauth2.StaticTokenSource(tok), nil
		}
		log.Printf("[debug] MCP OAuth: no usable persisted token; deferring to interactive authorization")
		return nil, nil
	}

	log.Printf("[debug] MCP OAuth: TokenSource wrapped with file persistence token_file=%q", g.tokenFile)
	return NewFilePersistingTokenSource(g.tokenFile, base), nil
}

func (g *gosdkOAuthHandler) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	if resp != nil && req != nil {
		g.maybeInjectProtectedResourceMetadata(req, resp)
	}
	// The local PRM shim (if started by maybeInjectProtectedResourceMetadata) is
	// only needed for the duration of this authorization flow; tear it down when
	// Authorize returns so we don't leak a listener for the process lifetime.
	defer g.closeProtectedResourceMetadataServer()

	reqURL := ""
	if req != nil && req.URL != nil {
		reqURL = req.URL.String()
	}
	status := ""
	wwwAuthenticate := ""
	if resp != nil {
		status = resp.Status
		wwwAuthenticate = resp.Header.Get("Www-Authenticate")
	}

	log.Printf("[debug] MCP OAuth: Authorize called request_host=%q response_status=%q www_authenticate_present=%t", hostFromURL(reqURL), status, strings.TrimSpace(wwwAuthenticate) != "")
	err := g.inner.Authorize(ctx, req, resp)
	if err != nil {
		log.Printf("[error] MCP OAuth: Authorize failed: %v", err)
	} else {
		log.Printf("[debug] MCP OAuth: Authorize completed")
	}
	return err
}

// closeProtectedResourceMetadataServer shuts down the local PRM shim server (if
// one was started) and resets its state so a later authorization attempt can
// start a fresh one. Safe to call when no server is running.
func (g *gosdkOAuthHandler) closeProtectedResourceMetadataServer() {
	g.prmMu.Lock()
	server := g.prmMetadataServer
	g.prmMetadataServer = nil
	g.prmMetadataBaseURL = ""
	g.prmMetadataServerAddr = ""
	g.prmMu.Unlock()

	if server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[error] MCP OAuth: error shutting down local PRM server: %v", err)
		return
	}
	log.Printf("[debug] MCP OAuth: local PRM server shut down")
}

func (g *gosdkOAuthHandler) maybeInjectProtectedResourceMetadata(req *http.Request, resp *http.Response) {
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		return
	}

	hdrVals := resp.Header[http.CanonicalHeaderKey("WWW-Authenticate")]
	if len(hdrVals) == 0 {
		return
	}

	challenges, err := oauthex.ParseWWWAuthenticate(hdrVals)
	if err != nil {
		log.Printf("[error] MCP OAuth: cannot parse WWW-Authenticate for PRM injection: %v", err)
		return
	}

	rmURL := resourceMetadataURLFromChallenges(challenges)
	if rmURL == "" {
		if isPRMShimEnabled() {
			log.Printf("[debug] MCP OAuth: PRM shim forced but the challenge carried no resource_metadata to normalize")
		}
		return
	}

	// Fetch the PRM once to get the (possibly pathful) authorization server issuer
	// and the resource's advertised scopes. The shim must forward scopes_supported
	// so the SDK requests the same scopes it would without the shim; otherwise the
	// authorize request is sent with no scope and the provider issues a token with
	// reduced/default scopes (Atlassian: only ...:agent-interface scopes, which the
	// Jira/Confluence data tools reject with "scope does not match").
	rm, err := fetchResourceMetadata(context.Background(), rmURL)
	if err != nil {
		log.Printf("[error] MCP OAuth: failed to read resource_metadata: %v", err)
		return
	}
	rawIssuer := rm.authorizationServer

	// The shim rewrites the authorization server's issuer so the SDK's strict
	// RFC 8414 issuer validation passes, which weakens the mix-up defence. Only
	// trust the issuer's metadata when it is served over HTTPS.
	if !isHTTPSURL(rawIssuer) {
		log.Printf("[error] MCP OAuth: not applying PRM shim, authorization server issuer is not HTTPS issuer=%q", rawIssuer)
		return
	}

	// Fetch the real authorization server metadata (the same document the SDK
	// fetches). Providers like Atlassian use a pathful issuer but declare the bare
	// origin as the metadata issuer, which trips the SDK's strict RFC 8414 check;
	// that is exactly the mismatch this shim normalizes.
	realMeta, metaErr := fetchAuthServerMetadataForPathfulIssuer(context.Background(), rawIssuer)
	if metaErr != nil {
		log.Printf("[error] MCP OAuth: could not fetch authorization server metadata for issuer=%q: %v", rawIssuer, metaErr)
	}

	metaIssuer := ""
	if realMeta != nil {
		metaIssuer, _ = realMeta.raw["issuer"].(string)
	}
	issuerMismatch := strings.TrimSpace(metaIssuer) != "" &&
		strings.TrimRight(strings.TrimSpace(metaIssuer), "/") != strings.TrimRight(strings.TrimSpace(rawIssuer), "/")

	// Only bypass the SDK's issuer validation when it would otherwise fail (the
	// advertised issuer disagrees with the metadata's declared issuer) or when
	// explicitly forced. Compliant providers keep the SDK's fully-validated flow.
	switch {
	case issuerMismatch:
		log.Printf("[info] MCP OAuth: authorization server metadata issuer %q does not match advertised issuer %q (RFC 8414 mismatch); applying PRM shim", metaIssuer, rawIssuer)
	case isPRMShimEnabled():
		log.Printf("[info] MCP OAuth: applying PRM shim (forced via TTYPHOON_MCP_OAUTH_ENABLE_PRM_SHIM) issuer=%q", rawIssuer)
	default:
		log.Printf("[debug] MCP OAuth: PRM shim not needed; authorization server issuer validates normally issuer=%q", rawIssuer)
		return
	}

	// Every credential-bearing endpoint the SDK will use must be HTTPS, or the
	// shim (which already defeats issuer validation) could let a tampered metadata
	// document downgrade the authorize/token/registration flow to plaintext and
	// leak the authorization code or tokens.
	if realMeta != nil {
		if err := validateAuthServerEndpointsHTTPS(realMeta.raw); err != nil {
			log.Printf("[error] MCP OAuth: refusing PRM shim, %v", err)
			return
		}
	}

	resource := req.URL.String()
	prmURL, err := g.ensureLocalProtectedResourceMetadataEndpoints(resource, realMeta, rm.scopesSupported)
	if err != nil {
		log.Printf("[error] MCP OAuth: failed to create local PRM endpoint: %v", err)
		return
	}

	canonical := http.CanonicalHeaderKey("WWW-Authenticate")
	existing := append([]string{}, resp.Header[canonical]...)
	resp.Header[canonical] = append([]string{fmt.Sprintf(`Bearer resource_metadata=%q`, prmURL)}, existing...)
	log.Printf("[debug] MCP OAuth: injected local PRM metadata URL prm_url=%q raw_issuer=%q", prmURL, rawIssuer)
}

// ensureLocalProtectedResourceMetadataEndpoints starts a local HTTP server (once) that serves:
//   - /.well-known/oauth-protected-resource  — PRM pointing to the local server as auth server
//   - /.well-known/oauth-authorization-server — synthetic auth server metadata with a local issuer
//     (matching the PRM entry) plus all real endpoints including registration_endpoint
//
// This lets go-sdk pass strict issuer validation while still finding the provider's DCR endpoint.
func (g *gosdkOAuthHandler) ensureLocalProtectedResourceMetadataEndpoints(resource string, realMeta *realAuthServerMeta, scopesSupported []string) (string, error) {
	g.prmMu.Lock()
	defer g.prmMu.Unlock()

	if g.prmMetadataBaseURL == "" {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return "", fmt.Errorf("listen local PRM endpoint: %w", err)
		}

		localBase := "http://" + ln.Addr().String()

		// Build auth server metadata JSON once: real endpoints + local issuer for strict validation
		authMetaJSON, _ := json.Marshal(buildSyntheticAuthServerMetadata(localBase, realMeta))

		mux := http.NewServeMux()

		// PRM: authorization_servers points to our local server (which has the correct issuer)
		mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
			requestedResource := r.URL.Query().Get("resource")
			if requestedResource == "" {
				requestedResource = resource
			}
			payload := map[string]any{
				"resource":              requestedResource,
				"authorization_servers": []string{localBase},
			}
			if len(scopesSupported) > 0 {
				payload["scopes_supported"] = scopesSupported
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(payload); err != nil {
				log.Printf("MCP OAuth: failed writing local PRM response: %v", err)
			}
		})

		// Auth server metadata: issuer == localBase (so go-sdk validation passes)
		// All real provider endpoints are preserved so auth/token/DCR flows reach the real server.
		serveAuthMeta := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(authMetaJSON)
		}
		mux.HandleFunc("/.well-known/oauth-authorization-server", serveAuthMeta)
		mux.HandleFunc("/.well-known/openid-configuration", serveAuthMeta)

		g.prmMetadataServerAddr = ln.Addr().String()
		g.prmMetadataBaseURL = localBase + "/.well-known/oauth-protected-resource"
		g.prmMetadataServer = &http.Server{Handler: mux}

		go func() {
			if err := g.prmMetadataServer.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Printf("[error] MCP OAuth: local PRM server error: %v", err)
			}
		}()

		regEndpoint := ""
		if realMeta != nil {
			regEndpoint = realMeta.registrationEndpoint
		}
		log.Printf("[debug] MCP OAuth: started local shim at %q local_issuer=%q registration_endpoint=%q",
			g.prmMetadataBaseURL, localBase, sanitizeURLForLog(regEndpoint))
	}

	return g.prmMetadataBaseURL + "?resource=" + url.QueryEscape(resource), nil
}

type realAuthServerMeta struct {
	raw                  map[string]interface{}
	registrationEndpoint string
}

// authServerMetadataCandidateURLs returns the well-known metadata URLs to try for
// an authorization-server issuer, in priority order. For a pathful issuer it
// tries the RFC 8414 layout first (well-known inserted between host and path,
// e.g. https://host/.well-known/oauth-authorization-server/<tenant>), which is
// what providers like Atlassian actually serve, then the OIDC path-appended
// layouts, then the root well-known endpoints.
func authServerMetadataCandidateURLs(issuer string) []string {
	u, err := url.Parse(strings.TrimSpace(issuer))
	if err != nil || u == nil || u.Scheme == "" || u.Host == "" {
		return nil
	}

	origin := u.Scheme + "://" + u.Host
	pathPart := strings.Trim(u.Path, "/")

	var urls []string
	if pathPart != "" {
		urls = append(urls,
			origin+"/.well-known/oauth-authorization-server/"+pathPart,
			origin+"/"+pathPart+"/.well-known/oauth-authorization-server",
			origin+"/"+pathPart+"/.well-known/openid-configuration",
		)
	}
	urls = append(urls,
		origin+"/.well-known/oauth-authorization-server",
		origin+"/.well-known/openid-configuration",
	)
	return urls
}

// fetchAuthServerMetadataForPathfulIssuer fetches authorization-server metadata for
// an issuer that may carry a path (e.g. https://auth.atlassian.com/<tenant>). It
// tries the RFC 8414 and OIDC well-known layouts in turn and returns the first
// document it can retrieve, preserving the tenant-scoped registration_endpoint.
func fetchAuthServerMetadataForPathfulIssuer(ctx context.Context, pathfulIssuer string) (*realAuthServerMeta, error) {
	candidates := authServerMetadataCandidateURLs(pathfulIssuer)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("invalid issuer URL %q", pathfulIssuer)
	}

	var lastErr error
	for _, metaURL := range candidates {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, metaURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("build request %s: %w", metaURL, err)
			continue
		}

		resp, err := ssrfSafeOAuthClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("fetch %s: %w", metaURL, err)
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			lastErr = fmt.Errorf("fetch %s: status %d", metaURL, resp.StatusCode)
			continue
		}

		var raw map[string]interface{}
		err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&raw)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("decode %s: %w", metaURL, err)
			continue
		}

		regEndpoint, _ := raw["registration_endpoint"].(string)
		metaIssuer, _ := raw["issuer"].(string)
		log.Printf("[debug] MCP OAuth: fetched authorization server metadata from %q issuer=%q registration_endpoint=%q",
			sanitizeURLForLog(metaURL), metaIssuer, sanitizeURLForLog(regEndpoint))

		return &realAuthServerMeta{raw: raw, registrationEndpoint: regEndpoint}, nil
	}

	return nil, lastErr
}

// buildSyntheticAuthServerMetadata builds an auth server metadata document for the local shim.
// It copies all real fields from realMeta (so real endpoints like authorize/token/registration
// are preserved), but patches the issuer to localBase so go-sdk's strict issuer validation passes.
func buildSyntheticAuthServerMetadata(localBase string, realMeta *realAuthServerMeta) map[string]any {
	m := map[string]any{
		"issuer": localBase,
	}
	if realMeta != nil && realMeta.raw != nil {
		for k, v := range realMeta.raw {
			if k != "issuer" {
				m[k] = v
			}
		}
	}
	return m
}

// isHTTPSURL reports whether raw is a well-formed https:// URL with a host.
func isHTTPSURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil {
		return false
	}
	return strings.EqualFold(u.Scheme, "https") && u.Host != ""
}

// validateAuthServerEndpointsHTTPS ensures the credential-bearing endpoints in a
// discovered authorization-server metadata document use HTTPS. It hardens the
// PRM shim, which bypasses the SDK's issuer validation, against a tampered
// metadata document downgrading the flow to plaintext.
func validateAuthServerEndpointsHTTPS(meta map[string]interface{}) error {
	for _, key := range []string{"authorization_endpoint", "token_endpoint", "registration_endpoint"} {
		v, ok := meta[key]
		if !ok {
			continue
		}
		s, _ := v.(string)
		if strings.TrimSpace(s) == "" {
			continue
		}
		if !isHTTPSURL(s) {
			return fmt.Errorf("auth server %s is not HTTPS: %q", key, s)
		}
	}
	return nil
}

func resourceMetadataURLFromChallenges(cs []oauthex.Challenge) string {
	for _, c := range cs {
		if u := strings.TrimSpace(c.Params["resource_metadata"]); u != "" {
			return u
		}
	}
	return ""
}

func parseOAuthCallbackURL(raw string) (code string, state string, err error) {
	log.Printf("MCP OAuth: parsing callback URL callback_host=%q callback_url=%q", hostFromURL(raw), sanitizeURLForLog(raw))
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("invalid callback URL: %w", err)
	}

	code = u.Query().Get("code")
	state = u.Query().Get("state")

	if strings.TrimSpace(code) == "" {
		return "", "", fmt.Errorf("callback URL missing code parameter")
	}
	if strings.TrimSpace(state) == "" {
		return "", "", fmt.Errorf("callback URL missing state parameter")
	}

	return code, state, nil
}

func hostFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return ""
	}
	return u.Hostname()
}

func sanitizeURLForLog(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return raw
	}
	if u.Path == "" {
		u.Path = "/"
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

/*func defaultOpenIDMetadataURLForAuthDomain(authDomain string) string {
	u, err := url.Parse(strings.TrimSpace(authDomain))
	if err != nil || u == nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/.well-known/openid-configuration"
}*/

func oauthRegistrationMethodsSummary(oauthCfg OAuthConfig) string {
	methods := make([]string, 0, 3)

	clientIDMetadataURL := strings.TrimSpace(oauthCfg.ClientURI)
	if clientIDMetadataURL == "" && strings.HasPrefix(strings.TrimSpace(oauthCfg.ClientID), "https://") {
		clientIDMetadataURL = strings.TrimSpace(oauthCfg.ClientID)
	}
	if clientIDMetadataURL != "" {
		methods = append(methods, "client_id_metadata_document")
	}

	if oauthCfg.ClientID != "" && clientIDMetadataURL != oauthCfg.ClientID {
		if oauthCfg.ClientSecret != "" {
			methods = append(methods, "pre_registered_client_with_secret")
		} else {
			methods = append(methods, "pre_registered_client_public")
		}
	}

	if oauthCfg.ClientID == "" || clientIDMetadataURL != "" {
		methods = append(methods, "dynamic_client_registration")
	}

	if len(methods) == 0 {
		return "none"
	}
	return strings.Join(methods, ",")
}

// logOAuthAuthorizeRequest parses the authorization request URL handed to the
// fetcher and logs the individual OAuth parameters (client_id, redirect_uri,
// scope, resource/audience, code_challenge_method, etc.). These are needed to
// diagnose scope/audience mismatches. None of these values are secrets: the
// client_id is public, the code_challenge is a one-way hash, and state is a
// CSRF nonce (only its presence is logged).
func logOAuthAuthorizeRequest(rawURL string) {
	u, err := url.Parse(rawURL)
	if err != nil || u == nil {
		log.Printf("[error] MCP OAuth: authorization request authorize_url=%q (unparseable: %v)", sanitizeURLForLog(rawURL), err)
		return
	}

	q := u.Query()
	endpoint := *u
	endpoint.RawQuery = ""
	endpoint.Fragment = ""

	resource := q.Get("resource")
	if resource == "" {
		resource = q.Get("audience")
	}

	log.Printf("MCP OAuth: authorization request authorization_endpoint=%q authorize_host=%q response_type=%q client_id=%q redirect_uri=%q scope=%q resource=%q code_challenge_method=%q code_challenge_present=%t state_present=%t",
		endpoint.String(),
		u.Hostname(),
		q.Get("response_type"),
		q.Get("client_id"),
		q.Get("redirect_uri"),
		q.Get("scope"),
		resource,
		q.Get("code_challenge_method"),
		q.Get("code_challenge") != "",
		q.Get("state") != "",
	)
}

// newOAuthLoggingClient returns the HTTP client the go-sdk uses for every OAuth
// side-channel request (authorization server metadata, protected resource
// metadata, dynamic client registration, and token exchange/refresh). Wrapping
// it lets us observe the token_endpoint, the registered client_id, and the
// granted scopes that the SDK otherwise handles internally.
func newOAuthLoggingClient() *http.Client {
	return &http.Client{Transport: &oauthLoggingRoundTripper{base: http.DefaultTransport}}
}

// ssrfSafeOAuthClient is used for OAuth metadata fetches whose target URL comes
// from server-controlled data (the resource_metadata and issuer URLs advertised
// in a WWW-Authenticate challenge). It refuses to connect to loopback, private,
// link-local, multicast or unspecified addresses to blunt SSRF attempts,
// including DNS-rebinding, because the Control hook runs after name resolution.
//
// It is deliberately NOT used for the SDK's own client (newOAuthLoggingClient),
// which must be able to reach the local 127.0.0.1 PRM shim.
var ssrfSafeOAuthClient = newSSRFSafeOAuthClient()

func newSSRFSafeOAuthClient() *http.Client {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	dialer.Control = func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("ssrf guard: cannot parse address %q: %w", address, err)
		}
		ip := net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("ssrf guard: unresolved address %q", host)
		}
		if !isPublicIP(ip) {
			return fmt.Errorf("ssrf guard: refusing to connect to non-public address %s", ip)
		}
		return nil
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: &oauthLoggingRoundTripper{base: transport},
	}
}

// isPublicIP reports whether ip is a globally routable unicast address, i.e. not
// loopback, private (RFC 1918 / RFC 4193), link-local, multicast or unspecified.
func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	return true
}

type oauthLoggingRoundTripper struct {
	base http.RoundTripper
}

func (rt *oauthLoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	sanitized := sanitizeURLForLog(req.URL.String())
	log.Printf("[debug] MCP OAuth: http request method=%s url=%q", req.Method, sanitized)

	resp, err := rt.base.RoundTrip(req)
	if err != nil {
		log.Printf("[error] MCP OAuth: http request failed method=%s url=%q err=%v", req.Method, sanitized, err)
		return resp, err
	}

	log.Printf("[debug] MCP OAuth: http response status=%d url=%q duration=%s", resp.StatusCode, sanitized, time.Since(start).Round(time.Millisecond))
	logSafeOAuthResponseFields(resp)
	return resp, nil
}

// logSafeOAuthResponseFields buffers a JSON OAuth response, restores it for the
// SDK to consume, and logs an allowlist of non-sensitive fields. It NEVER logs
// access_token, refresh_token, id_token or client_secret. cfg.Client is used
// only for small OAuth control-plane requests (not the MCP data stream), so
// fully buffering the body here is safe.
func logSafeOAuthResponseFields(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "json") {
		return
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return
	}
	// Restore the body so the SDK can still decode it.
	resp.Body = io.NopCloser(bytes.NewReader(body))

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return
	}

	// Allowlist of safe, diagnostically-useful fields across metadata,
	// registration and token responses. Secrets are deliberately excluded.
	safe := []string{
		"scope", "scopes_supported",
		"authorization_endpoint", "token_endpoint", "registration_endpoint",
		"code_challenge_methods_supported", "grant_types_supported",
		"response_types_supported", "client_id", "resource", "aud", "audience",
		"token_type", "expires_in", "issuer",
	}
	fields := make([]string, 0, len(safe))
	for _, k := range safe {
		if v, ok := parsed[k]; ok {
			fields = append(fields, fmt.Sprintf("%s=%v", k, v))
		}
	}
	if len(fields) > 0 {
		log.Printf("[debug] MCP OAuth: response fields %s", strings.Join(fields, " "))
	}
}

func enrichOAuthConnectError(err error, oauthCfg OAuthConfig) error {
	if err == nil {
		return nil
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "no configured client registration methods are supported by the authorization server") {
		return fmt.Errorf("%w; configured registration methods: %s; next steps: set oauth.clientId+oauth.clientSecret for a pre-registered app, or set oauth.clientUri to a provider-issued client ID metadata document URL", err, oauthRegistrationMethodsSummary(oauthCfg))
	}

	return err
}

func isPRMShimEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("TTYPHOON_MCP_OAUTH_ENABLE_PRM_SHIM")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

type resourceMetadata struct {
	authorizationServer string
	scopesSupported     []string
}

// fetchResourceMetadata reads an RFC 9728 Protected Resource Metadata document.
// It returns the first advertised authorization server and the resource's
// scopes_supported. The shim must forward scopes_supported so the SDK requests
// the same scopes it would without the shim.
func fetchResourceMetadata(ctx context.Context, metadataURL string) (*resourceMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := ssrfSafeOAuthClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var prm struct {
		AuthorizationServers []string `json:"authorization_servers"`
		ScopesSupported      []string `json:"scopes_supported"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&prm); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	if len(prm.AuthorizationServers) == 0 {
		return nil, fmt.Errorf("no authorization_servers")
	}

	return &resourceMetadata{
		authorizationServer: strings.TrimSpace(prm.AuthorizationServers[0]),
		scopesSupported:     prm.ScopesSupported,
	}, nil
}
