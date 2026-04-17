package agent

import (
	"context"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/ai/mcp_client"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func startServerCmdLine(cfgPath string, agent *Agent, envvars []string, server string, svr mcp_config.ServerT) error {
	debug.Log(envvars)
	log.Printf("MCP server %s: %s %v", server, svr.Command, svr.Args)

	c, err := mcp_client.ConnectCmdLine(&svr.Override, envvars, svr.Command, svr.Args...)
	if err != nil {
		return err
	}

	return startServer(cfgPath, agent, server, c)
}

func startServerHttp(cfgPath string, agent *Agent, server string, svr mcp_config.ServerT) error {
	serverURL := svr.Url
	log.Printf("MCP server %s: %s", server, serverURL)

	// Try plain HTTP first. If the server is protected, a 401/auth challenge will
	// cause us to switch into OAuth automatically.
	c, err := mcp_client.ConnectHttp(&svr.Override, serverURL)
	if err == nil {
		err = startServer(cfgPath, agent, server, c)
		if err == nil {
			return nil
		}
		if !mcp_client.IsAuthorizationFailure(err) {
			return err
		}
	} else if !mcp_client.IsAuthorizationFailure(err) {
		return err
	}

	agent.Renderer().DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("MCP server %s requires OAuth. Starting browser authentication...", server))

	oauthCfg := buildOAuthConfig(server, serverURL, svr.OAuth)

	return connectOAuthAndStart(&svr.Override, cfgPath, agent, server, serverURL, oauthCfg)
}

func buildOAuthConfig(server, serverURL string, oauth *mcp_config.OAuthT) mcp_client.OAuthConfig {
	redirectURI := mcp_client.DefaultRedirectURI()
	tokenFile := mcp_client.DefaultTokenFile(server, serverURL)
	pkceEnabled := true

	if oauth != nil {
		if oauth.RedirectURI != "" {
			redirectURI = oauth.RedirectURI
		}
		if oauth.TokenFile != "" {
			tokenFile = oauth.TokenFile
		}
		if oauth.Enabled || oauth.PKCEEnabled {
			pkceEnabled = oauth.PKCEEnabled
		}
	}

	var tokStore mcp_client.TokenStore = mcp_client.NewFileTokenStore(tokenFile)

	cfg := mcp_client.OAuthConfig{
		RedirectURI: redirectURI,
		PKCEEnabled: pkceEnabled,
		TokenStore:  tokStore,
	}

	if oauth != nil {
		cfg.ClientID = oauth.ClientID
		cfg.ClientSecret = oauth.ClientSecret
		cfg.Scopes = oauth.Scopes
		cfg.AuthServerMetadataURL = oauth.AuthServerMetadataURL
	}

	return cfg
}

func connectOAuthAndStart(overrides *mcp_config.OverrideT, cfgPath string, agent *Agent, server, serverURL string, oauthCfg mcp_client.OAuthConfig) error {
	c, err := mcp_client.ConnectHttpOAuth(overrides, serverURL, oauthCfg)
	if err != nil && mcp_client.IsAuthorizationFailure(err) {
		err = runOAuthInteractive(agent, err, oauthCfg.RedirectURI)
		if err != nil {
			return err
		}

		c, err = mcp_client.ConnectHttpOAuth(overrides, serverURL, oauthCfg)
	}
	if err != nil {
		return err
	}

	err = startServer(cfgPath, agent, server, c)
	if err != nil && mcp_client.IsAuthorizationFailure(err) {
		err = runOAuthInteractive(agent, err, oauthCfg.RedirectURI)
		if err != nil {
			return err
		}

		c, err = mcp_client.ConnectHttpOAuth(overrides, serverURL, oauthCfg)
		if err != nil {
			return err
		}

		return startServer(cfgPath, agent, server, c)
	}

	return err
}

func runOAuthInteractive(agent *Agent, oauthErr error, redirectURI string) error {
	h := mcp_client.GetOAuthHandler(oauthErr)
	if h == nil {
		return oauthErr
	}

	ctx := context.Background()

	codeVerifier, err := mcp_client.GenerateCodeVerifier()
	if err != nil {
		return fmt.Errorf("cannot generate OAuth code verifier: %w", err)
	}

	codeChallenge := mcp_client.GenerateCodeChallenge(codeVerifier)
	state, err := mcp_client.GenerateState()
	if err != nil {
		return fmt.Errorf("cannot generate OAuth state: %w", err)
	}

	if h.GetClientID() == "" {
		err = h.RegisterClient(ctx, "ttyphoon")
		if err != nil {
			return fmt.Errorf("cannot register OAuth client dynamically: %w", err)
		}
	}

	callback, err := startOAuthCallbackServer(redirectURI)
	if err != nil {
		agent.Renderer().DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Automatic OAuth callback unavailable (%v). Falling back to pasted callback URL.", err))
		return runOAuthInteractivePromptFallback(agent, h, ctx)
	}
	defer callback.Close()

	authURL, err := h.GetAuthorizationURL(ctx, state, codeChallenge)
	if err != nil {
		return fmt.Errorf("cannot build OAuth authorization URL: %w", err)
	}

	rctx := agent.Renderer().GetContext()
	if rctx != nil {
		runtime.BrowserOpenURL(rctx, authURL)
	}

	agent.Renderer().DisplayNotification(types.NOTIFY_INFO, "Complete OAuth in the opened browser window. TTYphoon is listening for the callback automatically.")

	params, err := callback.Wait(2 * time.Minute)
	if err != nil {
		return err
	}

	err = h.ProcessAuthorizationResponse(ctx, params.code, params.state, codeVerifier)
	if err != nil {
		return fmt.Errorf("OAuth token exchange failed: %w", err)
	}

	return nil
}

func runOAuthInteractivePromptFallback(agent *Agent, h any, ctx context.Context) error {
	oauthHandler, ok := h.(interface {
		GetAuthorizationURL(context.Context, string, string) (string, error)
		ProcessAuthorizationResponse(context.Context, string, string, string) error
	})
	if !ok {
		return fmt.Errorf("OAuth handler does not support interactive authorization")
	}

	codeVerifier, err := mcp_client.GenerateCodeVerifier()
	if err != nil {
		return fmt.Errorf("cannot generate OAuth code verifier: %w", err)
	}
	codeChallenge := mcp_client.GenerateCodeChallenge(codeVerifier)
	state, err := mcp_client.GenerateState()
	if err != nil {
		return fmt.Errorf("cannot generate OAuth state: %w", err)
	}

	authURL, err := oauthHandler.GetAuthorizationURL(ctx, state, codeChallenge)
	if err != nil {
		return fmt.Errorf("cannot build OAuth authorization URL: %w", err)
	}

	rctx := agent.Renderer().GetContext()
	if rctx != nil {
		runtime.BrowserOpenURL(rctx, authURL)
	}

	callbackURL, err := promptString(agent, "Paste OAuth callback URL")
	if err != nil {
		return err
	}

	code, returnedState, err := parseOAuthCallbackURL(callbackURL)
	if err != nil {
		return err
	}

	err = oauthHandler.ProcessAuthorizationResponse(ctx, code, returnedState, codeVerifier)
	if err != nil {
		return fmt.Errorf("OAuth token exchange failed: %w", err)
	}

	return nil
}

type oauthCallbackServer struct {
	server *http.Server
	result chan oauthCallbackResult
	err    chan error
}

type oauthCallbackResult struct {
	code  string
	state string
}

func startOAuthCallbackServer(redirectURI string) (*oauthCallbackServer, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect URI: %w", err)
	}
	if u.Scheme != "http" {
		return nil, fmt.Errorf("redirect URI must use http loopback for automatic callback capture")
	}
	host := u.Hostname()
	if host != "127.0.0.1" && host != "localhost" {
		return nil, fmt.Errorf("redirect URI host must be localhost or 127.0.0.1")
	}
	if u.Port() == "" {
		return nil, fmt.Errorf("redirect URI must include a port")
	}

	path := u.Path
	if path == "" {
		path = "/"
	}

	result := make(chan oauthCallbackResult, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if code == "" || state == "" {
			http.Error(w, "Missing code or state", http.StatusBadRequest)
			return
		}

		_, _ = io.WriteString(w, "<html><body><h1>Authentication complete</h1><p>You can close this window and return to "+html.EscapeString("TTYphoon")+".</p><script>window.close();</script></body></html>")
		select {
		case result <- oauthCallbackResult{code: code, state: state}:
		default:
		}
	})

	listener, err := net.Listen("tcp", u.Host)
	if err != nil {
		return nil, fmt.Errorf("cannot listen on OAuth callback URI %s: %w", redirectURI, err)
	}

	server := &http.Server{Handler: mux}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- serveErr
		}
	}()

	return &oauthCallbackServer{server: server, result: result, err: errCh}, nil
}

func (s *oauthCallbackServer) Wait(timeout time.Duration) (*oauthCallbackResult, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-s.result:
		return &res, nil
	case err := <-s.err:
		return nil, fmt.Errorf("OAuth callback server error: %w", err)
	case <-timer.C:
		return nil, fmt.Errorf("timed out waiting for OAuth callback")
	}
}

func (s *oauthCallbackServer) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.server.Shutdown(ctx)
}

func parseOAuthCallbackURL(raw string) (code string, state string, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("invalid callback URL: %w", err)
	}

	code = u.Query().Get("code")
	state = u.Query().Get("state")

	if code == "" {
		return "", "", fmt.Errorf("callback URL missing code parameter")
	}
	if state == "" {
		return "", "", fmt.Errorf("callback URL missing state parameter")
	}

	return code, state, nil
}

func promptString(agent *Agent, prompt string) (string, error) {
	ch := make(chan string)
	errCh := make(chan error)

	agent.Renderer().DisplayInputBox(prompt, "",
		func(s string) { ch <- s },
		func(_ string) { errCh <- fmt.Errorf("OAuth authorization canceled") },
	)

	select {
	case v := <-ch:
		if strings.TrimSpace(v) == "" {
			return "", fmt.Errorf("OAuth authorization canceled")
		}
		return v, nil
	case err := <-errCh:
		return "", err
	}
}

func startServer(cfgPath string, agent *Agent, server string, c *mcp_client.Client) error {
	err := c.ListTools()
	if err != nil {
		return err
	}

	agent.McpServerAdd(server, c)

	for i := range c.Tools.Tools {
		jsonSchema, err := c.Tools.Tools[i].MarshalJSON()
		if err != nil {
			return err
		}

		err = agent.ToolsAdd(&mcpTool{
			client: c,
			server: server,
			path:   cfgPath,
			name:   c.Tools.Tools[i].GetName(),
			schema: jsonSchema,
			description: fmt.Sprintf("%s\nInput schema: %s",
				c.Tools.Tools[i].Description,
				string(jsonSchema),
			),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
