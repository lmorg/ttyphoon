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

func ConnectAndUseHttp(overrides *mcp_config.OverrideT, server, serverURL string, oauth *mcp_config.OAuthT, hooks OAuthUIHooks, onOAuthRequired func(), useClient func(*Client) error) error {
	log.Printf("MCP OAuth: start ConnectAndUseHttp server=%q mcp_url=%q oauth_configured=%t", server, sanitizeURLForLog(serverURL), oauth != nil)
	if oauth != nil {
		log.Printf("MCP OAuth: provided config redirect_uri=%q metadata_url=%q scopes=%d client_id_set=%t client_secret_set=%t pkce=%t",
			oauth.RedirectURI,
			sanitizeURLForLog(oauth.AuthServerMetadataURL),
			len(oauth.Scopes),
			oauth.ClientID != "",
			oauth.ClientSecret != "",
			oauth.PKCEEnabled,
		)
	}

	c, err := ConnectHttp(overrides, serverURL)
	if err == nil {
		log.Printf("MCP OAuth: initial HTTP connect succeeded without OAuth")
		if useClient == nil {
			return nil
		}

		err = useClient(c)
		if err == nil {
			log.Printf("MCP OAuth: initial operation succeeded without OAuth")
			return nil
		}
		if !IsAuthorizationFailure(err) {
			log.Printf("MCP OAuth: initial operation failed with non-auth error: %v", err)
			return err
		}
		log.Printf("MCP OAuth: initial operation indicates OAuth required: %v", err)
	} else if !IsAuthorizationFailure(err) {
		log.Printf("MCP OAuth: initial connect failed with non-auth error: %v", err)
		return err
	} else {
		log.Printf("MCP OAuth: initial connect indicates OAuth required: %v", err)
	}

	if onOAuthRequired != nil {
		onOAuthRequired()
	}

	oauthCfg := BuildOAuthConfig(server, serverURL, oauth)
	log.Printf("MCP OAuth: interactive config redirect_uri=%q metadata_url=%q scopes=%d client_id_set=%t client_secret_set=%t pkce=%t",
		oauthCfg.RedirectURI,
		sanitizeURLForLog(oauthCfg.AuthServerMetadataURL),
		len(oauthCfg.Scopes),
		oauthCfg.ClientID != "",
		oauthCfg.ClientSecret != "",
		oauthCfg.PKCEEnabled,
	)

	c, err = ConnectHttpOAuthInteractive(overrides, serverURL, oauthCfg, hooks)
	if err != nil {
		log.Printf("MCP OAuth: interactive connect failed: %v", err)
		return err
	}
	log.Printf("MCP OAuth: interactive connect succeeded")

	if useClient == nil {
		return nil
	}

	err = useClient(c)
	if err != nil && IsAuthorizationFailure(err) {
		log.Printf("MCP OAuth: operation still unauthorized after interactive auth, retrying OAuth: %v", err)
		c, err = ConnectHttpOAuthInteractive(overrides, serverURL, oauthCfg, hooks)
		if err != nil {
			log.Printf("MCP OAuth: retry interactive connect failed: %v", err)
			return err
		}

		log.Printf("MCP OAuth: retry interactive connect succeeded; retrying operation")
		return useClient(c)
	}

	return err
}

func BuildOAuthConfig(server, serverURL string, oauth *mcp_config.OAuthT) OAuthConfig {
	redirectURI := DefaultRedirectURI()
	clientURI := DefaultClientURI()
	pkceEnabled := true
	var clientID, clientSecret, authServerMetadataURL string
	var scopes []string

	if oauth != nil {
		if oauth.RedirectURI != "" {
			redirectURI = oauth.RedirectURI
		}
		if oauth.Enabled || oauth.PKCEEnabled {
			pkceEnabled = oauth.PKCEEnabled
		}
		clientID = oauth.ClientID
		if oauth.ClientURI != "" {
			clientURI = oauth.ClientURI
		}
		clientSecret = oauth.ClientSecret
		scopes = oauth.Scopes
		authServerMetadataURL = oauth.AuthServerMetadataURL
	}

	tokenFile := DefaultTokenFile(server, serverURL)
	if oauth != nil && oauth.TokenFile != "" {
		tokenFile = oauth.TokenFile
	}

	cfg := OAuthConfig{
		RedirectURI:           redirectURI,
		PKCEEnabled:           pkceEnabled,
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
	log.Printf("MCP OAuth: BuildOAuthConfig server=%q mcp_host=%q auth_metadata_host=%q redirect_host=%q", server, mcpHost, authHost, redirectHost)
	if cfg.ClientURI != "" {
		log.Printf("MCP OAuth: BuildOAuthConfig client_uri=%q", sanitizeURLForLog(cfg.ClientURI))
	}
	if mcpHost != "" && authHost != "" && mcpHost != authHost {
		log.Printf("MCP OAuth: info MCP host differs from Auth host (often valid): mcp=%q auth=%q", mcpHost, authHost)
	}

	return cfg
}

// AuthenticateOAuthInteractive is deprecated and kept for reference only.
// OAuth authentication now uses go-sdk AuthorizationCodeHandler via buildGoSDKAuthHandler.
func AuthenticateOAuthInteractive(oauthErr error, serverURL, redirectURI string, overrides *mcp_config.OverrideT, openBrowser func(string), promptCallbackURL func() (string, error), onAutoCallbackUnavailable func(error)) error {
	return fmt.Errorf("OAuth authentication requires go-sdk handler (legacy mark3labs path removed)")
}

func ConnectHttpOAuthInteractive(overrides *mcp_config.OverrideT, serverURL string, oauthCfg OAuthConfig, hooks OAuthUIHooks) (*Client, error) {
	log.Printf("MCP OAuth: ConnectHttpOAuthInteractive mcp_url=%q redirect_uri=%q metadata_url=%q", sanitizeURLForLog(serverURL), oauthCfg.RedirectURI, sanitizeURLForLog(oauthCfg.AuthServerMetadataURL))
	log.Printf("MCP OAuth: configured registration methods=%s", oauthRegistrationMethodsSummary(oauthCfg))

	// Try Go-SDK AuthorizationCodeHandler-backed flow using UI hooks
	if handler, hErr := buildGoSDKAuthHandler(oauthCfg, overrides, hooks); hErr == nil && handler != nil {
		log.Printf("MCP OAuth: authorization handler created")
		if g, connErr := ConnectHttpGoSDK(overrides, serverURL, handler); connErr == nil {
			log.Printf("MCP OAuth: OAuth HTTP connect succeeded")
			return initClientFromGoSDK(g, overrides)
		} else {
			enriched := enrichOAuthConnectError(connErr, oauthCfg)
			log.Printf("MCP OAuth: OAuth HTTP connect failed: %v", enriched)
			return nil, fmt.Errorf("failed to establish OAuth connection to %s: %w", serverURL, enriched)
		}
	} else {
		log.Printf("MCP OAuth: failed creating authorization handler: %v", hErr)
		return nil, fmt.Errorf("failed to create OAuth authorization handler: %w", hErr)
	}

	// If go-sdk flow fails, return error (no fallback to legacy)
	//return nil, fmt.Errorf("failed to establish OAuth connection to %s", serverURL)
}

// buildGoSDKAuthHandler constructs an auth.OAuthHandler backed by the
// go-sdk AuthorizationCodeHandler using provided OAuth config and UI hooks.
func buildGoSDKAuthHandler(oauthCfg OAuthConfig, overrides *mcp_config.OverrideT, hooks OAuthUIHooks) (authsdk.OAuthHandler, error) {
	if oauthCfg.RedirectURI == "" {
		return nil, fmt.Errorf("redirect URI required for interactive OAuth")
	}

	log.Printf("MCP OAuth: buildGoSDKAuthHandler redirect_host=%q auth_metadata_host=%q", hostFromURL(oauthCfg.RedirectURI), hostFromURL(oauthCfg.AuthServerMetadataURL))

	fetcher := func(ctx context.Context, args *authsdk.AuthorizationArgs) (*authsdk.AuthorizationResult, error) {
		logOAuthAuthorizeRequest(args.URL)

		// Try automatic callback server first
		callback, err := StartOAuthCallbackServer(oauthCfg.RedirectURI)
		if err == nil {
			log.Printf("MCP OAuth: automatic callback server started redirect_uri=%q", oauthCfg.RedirectURI)
			defer callback.Close()
			if hooks.OpenBrowser != nil {
				log.Printf("MCP OAuth: opening browser for authorization")
				hooks.OpenBrowser(args.URL)
			}
			log.Printf("MCP OAuth: waiting for automatic callback")
			res, waitErr := callback.Wait(2 * time.Minute)
			if waitErr != nil {
				log.Printf("MCP OAuth: automatic callback wait failed: %v", waitErr)
				return nil, waitErr
			}
			log.Printf("MCP OAuth: automatic callback received state_present=%t code_len=%d", res.State != "", len(res.Code))
			return &authsdk.AuthorizationResult{Code: res.Code, State: res.State}, nil
		}
		log.Printf("MCP OAuth: automatic callback unavailable: %v", err)

		if hooks.OnAutoCallbackUnavailable != nil {
			hooks.OnAutoCallbackUnavailable(err)
		}

		// Fallback to pasted callback URL
		if hooks.PromptCallbackURL == nil {
			log.Printf("MCP OAuth: no PromptCallbackURL hook available")
			return nil, fmt.Errorf("no method to obtain callback URL")
		}
		if hooks.OpenBrowser != nil {
			log.Printf("MCP OAuth: opening browser for authorization (manual callback mode)")
			hooks.OpenBrowser(args.URL)
		}
		raw, pErr := hooks.PromptCallbackURL()
		if pErr != nil {
			log.Printf("MCP OAuth: PromptCallbackURL failed: %v", pErr)
			return nil, pErr
		}
		log.Printf("MCP OAuth: manual callback URL received callback_host=%q callback_url=%q", hostFromURL(raw), sanitizeURLForLog(raw))
		code, state, parseErr := parseOAuthCallbackURL(raw)
		if parseErr != nil {
			log.Printf("MCP OAuth: manual callback parse failed: %v", parseErr)
			return nil, parseErr
		}
		log.Printf("MCP OAuth: manual callback parsed state_present=%t code_len=%d", state != "", len(code))
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
		log.Printf("MCP OAuth: enabling Client ID Metadata Document registration url=%q", sanitizeURLForLog(clientIDMetadataURL))
		cfg.ClientIDMetadataDocumentConfig = &authsdk.ClientIDMetadataDocumentConfig{URL: clientIDMetadataURL}
	}

	// client registration: prefer preregistered client if provided
	if oauthCfg.ClientID != "" && clientIDMetadataURL != oauthCfg.ClientID {
		log.Printf("MCP OAuth: using preregistered client client_secret_set=%t", oauthCfg.ClientSecret != "")
		cfg.PreregisteredClient = &oauthex.ClientCredentials{
			ClientID: oauthCfg.ClientID,
		}
		if oauthCfg.ClientSecret != "" {
			cfg.PreregisteredClient.ClientSecretAuth = &oauthex.ClientSecretAuth{ClientSecret: oauthCfg.ClientSecret}
		}
	}

	// Use dynamic registration whenever we are not using explicit pre-registered credentials.
	if cfg.PreregisteredClient == nil {
		log.Printf("MCP OAuth: using dynamic client registration")
		cfg.DynamicClientRegistrationConfig = &authsdk.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				RedirectURIs: []string{oauthCfg.RedirectURI},
				ClientName:   app.Name(),
			},
		}
	} else if cfg.DynamicClientRegistrationConfig == nil {
		log.Printf("MCP OAuth: dynamic client registration disabled due to explicit client credentials")
	}

	if cfg.ClientIDMetadataDocumentConfig == nil && cfg.PreregisteredClient == nil && cfg.DynamicClientRegistrationConfig == nil {
		log.Printf("MCP OAuth: warning no client registration method configured")
	}

	h, err := authsdk.NewAuthorizationCodeHandler(cfg)
	if err != nil {
		log.Printf("MCP OAuth: NewAuthorizationCodeHandler failed: %v", err)
		return nil, err
	}
	log.Printf("MCP OAuth: NewAuthorizationCodeHandler created")

	preferredIssuer := ""
	if oauthCfg.AuthServerMetadataURL != "" {
		issuer, issuerErr := resolvePreferredAuthIssuer(context.Background(), oauthCfg.AuthServerMetadataURL, cfg.Client)
		if issuerErr != nil {
			log.Printf("MCP OAuth: unable to resolve preferred issuer from metadata URL: %v", issuerErr)
		} else {
			preferredIssuer = issuer
		}
	}

	// Wrap the handler to persist tokens to file
	return &gosdkOAuthHandler{inner: h, preferredAuthIssuer: preferredIssuer, tokenFile: oauthCfg.TokenFile}, nil
}

// gosdkOAuthHandler wraps an auth.AuthorizationCodeHandler and persists tokens
// to a file when TokenSource is requested.
type gosdkOAuthHandler struct {
	inner                 *authsdk.AuthorizationCodeHandler
	tokenFile             string
	preferredAuthIssuer   string
	prmMu                 sync.Mutex
	prmMetadataBaseURL    string
	prmMetadataServer     *http.Server
	prmMetadataServerAddr string
}

func (g *gosdkOAuthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	log.Printf("MCP OAuth: TokenSource requested token_file_set=%t", g.tokenFile != "")
	base, err := g.inner.TokenSource(ctx)
	if err != nil {
		log.Printf("MCP OAuth: inner TokenSource failed: %v", err)
		return nil, err
	}
	if g.tokenFile == "" {
		log.Printf("MCP OAuth: TokenSource ready without persistence")
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
			log.Printf("MCP OAuth: reusing persisted token token_file=%q", g.tokenFile)
			return oauth2.StaticTokenSource(tok), nil
		}
		log.Printf("MCP OAuth: no usable persisted token; deferring to interactive authorization")
		return nil, nil
	}

	log.Printf("MCP OAuth: TokenSource wrapped with file persistence token_file=%q", g.tokenFile)
	return NewFilePersistingTokenSource(g.tokenFile, base), nil
}

func (g *gosdkOAuthHandler) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	if resp != nil && req != nil {
		g.maybeInjectProtectedResourceMetadata(req, resp)
	}

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

	log.Printf("MCP OAuth: Authorize called request_host=%q response_status=%q www_authenticate_present=%t", hostFromURL(reqURL), status, strings.TrimSpace(wwwAuthenticate) != "")
	err := g.inner.Authorize(ctx, req, resp)
	if err != nil {
		log.Printf("MCP OAuth: Authorize failed: %v", err)
	} else {
		log.Printf("MCP OAuth: Authorize completed")
	}
	return err
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
		log.Printf("MCP OAuth: cannot parse WWW-Authenticate for PRM injection: %v", err)
		return
	}

	rmURL := resourceMetadataURLFromChallenges(challenges)
	if rmURL == "" {
		if !isPRMShimEnabled() {
			log.Printf("MCP OAuth: PRM shim disabled by default (set TTYPHOON_MCP_OAUTH_ENABLE_PRM_SHIM=1 to enable)")
		}
		return
	}

	// Fetch PRM once to get the raw (possibly pathful) authorization server issuer
	rawIssuer, err := extractIssuerFromResourceMetadata(context.Background(), rmURL)
	if err != nil {
		log.Printf("MCP OAuth: failed to extract issuer from resource_metadata: %v", err)
		return
	}

	// Auto-enable if pathful issuer detected (e.g. Atlassian's tenant-scoped issuers)
	if issuerHasPathComponent(rawIssuer) {
		log.Printf("MCP OAuth: detected pathful issuer in resource_metadata, auto-enabling PRM shim for normalization issuer=%q", rawIssuer)
	} else if !isPRMShimEnabled() {
		log.Printf("MCP OAuth: PRM shim disabled by default (set TTYPHOON_MCP_OAUTH_ENABLE_PRM_SHIM=1 to enable)")
		return
	}

	// Fetch the real auth server metadata from the pathful issuer URL.
	// This is where providers like Atlassian advertise their registration_endpoint.
	realMeta, metaErr := fetchAuthServerMetadataForPathfulIssuer(context.Background(), rawIssuer)
	if metaErr != nil {
		log.Printf("MCP OAuth: failed to fetch auth server metadata for issuer=%q: %v (continuing without)", rawIssuer, metaErr)
	}

	resource := req.URL.String()
	prmURL, err := g.ensureLocalProtectedResourceMetadataEndpoints(resource, realMeta)
	if err != nil {
		log.Printf("MCP OAuth: failed to create local PRM endpoint: %v", err)
		return
	}

	canonical := http.CanonicalHeaderKey("WWW-Authenticate")
	existing := append([]string{}, resp.Header[canonical]...)
	resp.Header[canonical] = append([]string{fmt.Sprintf(`Bearer resource_metadata=%q`, prmURL)}, existing...)
	log.Printf("MCP OAuth: injected local PRM metadata URL prm_url=%q raw_issuer=%q", prmURL, rawIssuer)
}

// ensureLocalProtectedResourceMetadataEndpoints starts a local HTTP server (once) that serves:
//   - /.well-known/oauth-protected-resource  — PRM pointing to the local server as auth server
//   - /.well-known/oauth-authorization-server — synthetic auth server metadata with a local issuer
//     (matching the PRM entry) plus all real endpoints including registration_endpoint
//
// This lets go-sdk pass strict issuer validation while still finding the provider's DCR endpoint.
func (g *gosdkOAuthHandler) ensureLocalProtectedResourceMetadataEndpoints(resource string, realMeta *realAuthServerMeta) (string, error) {
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
				log.Printf("MCP OAuth: local PRM server error: %v", err)
			}
		}()

		regEndpoint := ""
		if realMeta != nil {
			regEndpoint = realMeta.registrationEndpoint
		}
		log.Printf("MCP OAuth: started local shim at %q local_issuer=%q registration_endpoint=%q",
			g.prmMetadataBaseURL, localBase, sanitizeURLForLog(regEndpoint))
	}

	return g.prmMetadataBaseURL + "?resource=" + url.QueryEscape(resource), nil
}

type realAuthServerMeta struct {
	raw                  map[string]interface{}
	registrationEndpoint string
}

// fetchAuthServerMetadataForPathfulIssuer fetches auth server metadata from a potentially
// pathful issuer URL (e.g. https://auth.atlassian.com/tenant-id/...).
// It tries {issuer}/.well-known/oauth-authorization-server, which is where providers like
// Atlassian advertise the tenant-scoped registration_endpoint.
func fetchAuthServerMetadataForPathfulIssuer(ctx context.Context, pathfulIssuer string) (*realAuthServerMeta, error) {
	metaURL := strings.TrimRight(pathfulIssuer, "/") + "/.well-known/oauth-authorization-server"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", metaURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s: status %d", metaURL, resp.StatusCode)
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode %s: %w", metaURL, err)
	}

	regEndpoint, _ := raw["registration_endpoint"].(string)
	log.Printf("MCP OAuth: fetched pathful issuer metadata from %q registration_endpoint=%q",
		sanitizeURLForLog(metaURL), sanitizeURLForLog(regEndpoint))

	return &realAuthServerMeta{raw: raw, registrationEndpoint: regEndpoint}, nil
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

func resourceMetadataURLFromChallenges(cs []oauthex.Challenge) string {
	for _, c := range cs {
		if u := strings.TrimSpace(c.Params["resource_metadata"]); u != "" {
			return u
		}
	}
	return ""
}

func normalizedIssuerFromResourceMetadata(ctx context.Context, metadataURL, resourceURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return "", fmt.Errorf("build PRM request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch PRM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("PRM status %d", resp.StatusCode)
	}

	var prm struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&prm); err != nil {
		return "", fmt.Errorf("decode PRM JSON: %w", err)
	}
	if prm.Resource != "" && resourceURL != "" && prm.Resource != resourceURL {
		log.Printf("MCP OAuth: PRM resource mismatch prm_resource=%q request_resource=%q", prm.Resource, resourceURL)
	}
	if len(prm.AuthorizationServers) == 0 {
		return "", fmt.Errorf("PRM missing authorization_servers")
	}

	issuerRaw := strings.TrimSpace(prm.AuthorizationServers[0])
	u, err := url.Parse(issuerRaw)
	if err != nil {
		return "", fmt.Errorf("parse authorization server URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid authorization server URL %q", issuerRaw)
	}

	normalized := u.Scheme + "://" + u.Host
	if normalized == issuerRaw {
		return "", nil
	}
	log.Printf("MCP OAuth: normalizing authorization server issuer from %q to %q", issuerRaw, normalized)
	return normalized, nil
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

func resolvePreferredAuthIssuer(ctx context.Context, metadataURL string, httpClient *http.Client) (string, error) {
	if strings.TrimSpace(metadataURL) == "" {
		return "", nil
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	log.Printf("MCP OAuth: resolving preferred auth issuer from metadata_url=%q", sanitizeURLForLog(metadataURL))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return "", fmt.Errorf("build metadata request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read metadata body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("metadata status %d", resp.StatusCode)
	}

	var parsed struct {
		Issuer string `json:"issuer"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decode metadata JSON: %w", err)
	}

	issuer := strings.TrimSpace(parsed.Issuer)
	if issuer == "" {
		return "", fmt.Errorf("metadata missing issuer")
	}

	log.Printf("MCP OAuth: resolved preferred auth issuer=%q from metadata_url=%q", issuer, sanitizeURLForLog(metadataURL))
	return issuer, nil
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
		log.Printf("MCP OAuth: authorization request authorize_url=%q (unparseable: %v)", sanitizeURLForLog(rawURL), err)
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

type oauthLoggingRoundTripper struct {
	base http.RoundTripper
}

func (rt *oauthLoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	sanitized := sanitizeURLForLog(req.URL.String())
	log.Printf("MCP OAuth: http request method=%s url=%q", req.Method, sanitized)

	resp, err := rt.base.RoundTrip(req)
	if err != nil {
		log.Printf("MCP OAuth: http request failed method=%s url=%q err=%v", req.Method, sanitized, err)
		return resp, err
	}

	log.Printf("MCP OAuth: http response status=%d url=%q duration=%s", resp.StatusCode, sanitized, time.Since(start).Round(time.Millisecond))
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
		log.Printf("MCP OAuth: response fields %s", strings.Join(fields, " "))
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

func issuerHasPathComponent(issuer string) bool {
	if issuer == "" {
		return false
	}
	u, err := url.Parse(issuer)
	if err != nil {
		return false
	}
	path := strings.TrimSpace(u.Path)
	// Has path component if non-empty and not just "/"
	return path != "" && path != "/"
}

func extractIssuerFromResourceMetadata(ctx context.Context, metadataURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var prm struct {
		AuthorizationServers []string `json:"authorization_servers"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&prm); err != nil {
		return "", fmt.Errorf("decode JSON: %w", err)
	}
	if len(prm.AuthorizationServers) == 0 {
		return "", fmt.Errorf("no authorization_servers")
	}

	return strings.TrimSpace(prm.AuthorizationServers[0]), nil
}
