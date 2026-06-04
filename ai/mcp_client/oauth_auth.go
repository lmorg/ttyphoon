package mcp_client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
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
	pkceEnabled := true
	var clientID, clientURI, clientSecret, authServerMetadataURL string
	var scopes []string

	if oauth != nil {
		if oauth.RedirectURI != "" {
			redirectURI = oauth.RedirectURI
		}
		if oauth.Enabled || oauth.PKCEEnabled {
			pkceEnabled = oauth.PKCEEnabled
		}
		clientID = oauth.ClientID
		clientURI = oauth.ClientURI
		clientSecret = oauth.ClientSecret
		scopes = oauth.Scopes
		authServerMetadataURL = oauth.AuthServerMetadataURL
	}

	cfg := OAuthConfig{
		RedirectURI:           redirectURI,
		PKCEEnabled:           pkceEnabled,
		ClientID:              clientID,
		ClientURI:             clientURI,
		ClientSecret:          clientSecret,
		Scopes:                scopes,
		AuthServerMetadataURL: authServerMetadataURL,
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
	return nil, fmt.Errorf("failed to establish OAuth connection to %s", serverURL)
}

// buildGoSDKAuthHandler constructs an auth.OAuthHandler backed by the
// go-sdk AuthorizationCodeHandler using provided OAuth config and UI hooks.
func buildGoSDKAuthHandler(oauthCfg OAuthConfig, overrides *mcp_config.OverrideT, hooks OAuthUIHooks) (authsdk.OAuthHandler, error) {
	if oauthCfg.RedirectURI == "" {
		return nil, fmt.Errorf("redirect URI required for interactive OAuth")
	}

	log.Printf("MCP OAuth: buildGoSDKAuthHandler redirect_host=%q auth_metadata_host=%q", hostFromURL(oauthCfg.RedirectURI), hostFromURL(oauthCfg.AuthServerMetadataURL))

	fetcher := func(ctx context.Context, args *authsdk.AuthorizationArgs) (*authsdk.AuthorizationResult, error) {
		log.Printf("MCP OAuth: authorization request authorize_url=%q authorize_host=%q", sanitizeURLForLog(args.URL), hostFromURL(args.URL))

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
		Client:                   http.DefaultClient,
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
	return &gosdkOAuthHandler{inner: h, preferredAuthIssuer: preferredIssuer}, nil
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

	issuerToForce := strings.TrimSpace(g.preferredAuthIssuer)
	if issuerToForce == "" {
		if rmURL := resourceMetadataURLFromChallenges(challenges); rmURL != "" {
			normalizedIssuer, nErr := normalizedIssuerFromResourceMetadata(context.Background(), rmURL, req.URL.String())
			if nErr != nil {
				log.Printf("MCP OAuth: failed to derive normalized issuer from resource metadata: %v", nErr)
			} else if normalizedIssuer != "" {
				issuerToForce = normalizedIssuer
				log.Printf("MCP OAuth: derived normalized issuer from challenge resource metadata issuer=%q", issuerToForce)

				fallbackMetadataURL := defaultOpenIDMetadataURLForAuthDomain(issuerToForce)
				if fallbackMetadataURL != "" {
					log.Printf("MCP OAuth: trying fallback auth metadata URL=%q", sanitizeURLForLog(fallbackMetadataURL))
					if discoveredIssuer, dErr := resolvePreferredAuthIssuer(context.Background(), fallbackMetadataURL, http.DefaultClient); dErr != nil {
						log.Printf("MCP OAuth: fallback metadata lookup failed: %v", dErr)
					} else if discoveredIssuer != "" {
						issuerToForce = discoveredIssuer
						log.Printf("MCP OAuth: fallback metadata lookup succeeded issuer=%q", issuerToForce)
					}
				}
			}
		}
	}

	if issuerToForce == "" {
		return
	}

	resource := req.URL.String()
	prmURL, err := g.ensureLocalProtectedResourceMetadataURL(resource, issuerToForce)
	if err != nil {
		log.Printf("MCP OAuth: failed to create local PRM endpoint: %v", err)
		return
	}

	canonical := http.CanonicalHeaderKey("WWW-Authenticate")
	existing := append([]string{}, resp.Header[canonical]...)
	resp.Header[canonical] = append([]string{fmt.Sprintf(`Bearer resource_metadata=%q`, prmURL)}, existing...)
	log.Printf("MCP OAuth: injected local PRM metadata URL for issuer alignment prm_url=%q issuer=%q", prmURL, issuerToForce)
}

func (g *gosdkOAuthHandler) ensureLocalProtectedResourceMetadataURL(resource, issuer string) (string, error) {
	g.prmMu.Lock()
	defer g.prmMu.Unlock()

	if g.prmMetadataBaseURL == "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
			requestedResource := r.URL.Query().Get("resource")
			if requestedResource == "" {
				requestedResource = resource
			}

			payload := map[string]any{
				"resource":              requestedResource,
				"authorization_servers": []string{issuer},
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(payload); err != nil {
				log.Printf("MCP OAuth: failed writing local PRM response: %v", err)
			}
		})

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return "", fmt.Errorf("listen local PRM endpoint: %w", err)
		}

		g.prmMetadataServerAddr = ln.Addr().String()
		g.prmMetadataBaseURL = "http://" + g.prmMetadataServerAddr + "/.well-known/oauth-protected-resource"
		g.prmMetadataServer = &http.Server{Handler: mux}

		go func() {
			if err := g.prmMetadataServer.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Printf("MCP OAuth: local PRM server error: %v", err)
			}
		}()

		log.Printf("MCP OAuth: started local PRM endpoint at %q for issuer=%q", g.prmMetadataBaseURL, issuer)
	}

	return g.prmMetadataBaseURL + "?resource=" + url.QueryEscape(resource), nil
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

func defaultOpenIDMetadataURLForAuthDomain(authDomain string) string {
	u, err := url.Parse(strings.TrimSpace(authDomain))
	if err != nil || u == nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/.well-known/openid-configuration"
}

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
