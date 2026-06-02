package mcp_client

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/utils/or"
	"github.com/mark3labs/mcp-go/client/transport"
)

func logOAuthDebugf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "[mcp oauth] "+format+"\n", args...)
}

func logOAuthHandlerDetails(ctx context.Context, serverURL, redirectURI string, h *transport.OAuthHandler) {
	if h == nil {
		logOAuthDebugf("handler unavailable for server=%q redirect=%q", serverURL, redirectURI)
		return
	}

	logOAuthDebugf("server=%q redirect_uri=%q client_id=%q client_secret_set=%t", serverURL, redirectURI, h.GetClientID(), h.GetClientSecret() != "")

	metadata, err := h.GetServerMetadata(ctx)
	if err != nil {
		logOAuthDebugf("metadata discovery failed for server=%q: %v", serverURL, err)
		return
	}
	if metadata == nil {
		logOAuthDebugf("metadata discovery returned nil for server=%q", serverURL)
		return
	}

	logOAuthDebugf(
		"metadata for server=%q issuer=%q authorization_endpoint=%q token_endpoint=%q registration_endpoint=%q",
		serverURL,
		metadata.Issuer,
		metadata.AuthorizationEndpoint,
		metadata.TokenEndpoint,
		metadata.RegistrationEndpoint,
	)
}

type OAuthUIHooks struct {
	OpenBrowser               func(string)
	PromptCallbackURL         func() (string, error)
	OnAutoCallbackUnavailable func(error)
}

func ConnectAndUseHttp(overrides *mcp_config.OverrideT, server, serverURL string, oauth *mcp_config.OAuthT, hooks OAuthUIHooks, onOAuthRequired func(), useClient func(*Client) error) error {
	c, err := ConnectHttp(overrides, serverURL)
	if err == nil {
		if useClient == nil {
			return nil
		}

		err = useClient(c)
		if err == nil {
			return nil
		}
		if !IsAuthorizationFailure(err) {
			return err
		}
	} else if !IsAuthorizationFailure(err) {
		return err
	}

	if onOAuthRequired != nil {
		onOAuthRequired()
	}

	oauthCfg := BuildOAuthConfig(server, serverURL, oauth)

	c, err = ConnectHttpOAuthInteractive(overrides, serverURL, oauthCfg, hooks)
	if err != nil {
		return err
	}

	if useClient == nil {
		return nil
	}

	err = useClient(c)
	if err != nil && IsAuthorizationFailure(err) {
		c, err = ConnectHttpOAuthInteractive(overrides, serverURL, oauthCfg, hooks)
		if err != nil {
			return err
		}

		return useClient(c)
	}

	return err
}

func BuildOAuthConfig(server, serverURL string, oauth *mcp_config.OAuthT) OAuthConfig {
	redirectURI := DefaultRedirectURI()
	tokenFile := DefaultTokenFile(server, serverURL)
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

	var tokStore TokenStore = NewFileTokenStore(tokenFile)

	cfg := OAuthConfig{
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

func AuthenticateOAuthInteractive(oauthErr error, serverURL, redirectURI string, overrides *mcp_config.OverrideT, openBrowser func(string), promptCallbackURL func() (string, error), onAutoCallbackUnavailable func(error)) error {
	h := GetOAuthHandler(oauthErr)
	if h == nil {
		logOAuthDebugf("no OAuth handler available for server=%q redirect_uri=%q error=%v", serverURL, redirectURI, oauthErr)
		return oauthErr
	}

	ctx := context.Background()
	logOAuthHandlerDetails(ctx, serverURL, redirectURI, h)

	codeVerifier, err := GenerateCodeVerifier()
	if err != nil {
		return fmt.Errorf("cannot generate OAuth code verifier: %w", err)
	}

	codeChallenge := GenerateCodeChallenge(codeVerifier)
	state, err := GenerateState()
	if err != nil {
		return fmt.Errorf("cannot generate OAuth state: %w", err)
	}

	appName := app.Name()
	if overrides != nil {
		appName = or.NotEmpty(overrides.AppName, app.Name())
	}

	if h.GetClientID() == "" {
		logOAuthDebugf("dynamic client registration starting for server=%q app_name=%q", serverURL, appName)
		err = h.RegisterClient(ctx, appName)
		if err != nil {
			logOAuthDebugf("dynamic client registration failed for server=%q: %v", serverURL, err)
			logOAuthHandlerDetails(ctx, serverURL, redirectURI, h)
			return fmt.Errorf("cannot register OAuth client dynamically: %w", err)
		}
		logOAuthDebugf("dynamic client registration succeeded for server=%q client_id=%q", serverURL, h.GetClientID())
	}

	callback, err := StartOAuthCallbackServer(redirectURI)
	if err == nil {
		defer callback.Close()

		authURL, authErr := h.GetAuthorizationURL(ctx, state, codeChallenge)
		if authErr != nil {
			logOAuthDebugf("authorization URL build failed for server=%q: %v", serverURL, authErr)
			logOAuthHandlerDetails(ctx, serverURL, redirectURI, h)
			return fmt.Errorf("cannot build OAuth authorization URL: %w", authErr)
		}
		logOAuthDebugf("authorization URL for server=%q: %s", serverURL, authURL)
		if openBrowser != nil {
			openBrowser(authURL)
		}

		params, waitErr := callback.Wait(2 * time.Minute)
		if waitErr != nil {
			logOAuthDebugf("callback wait failed for server=%q: %v", serverURL, waitErr)
			return waitErr
		}

		err = h.ProcessAuthorizationResponse(ctx, params.Code, params.State, codeVerifier)
		if err != nil {
			logOAuthDebugf("token exchange failed for server=%q: %v", serverURL, err)
			logOAuthHandlerDetails(ctx, serverURL, redirectURI, h)
			return fmt.Errorf("OAuth token exchange failed: %w", err)
		}
		logOAuthDebugf("token exchange succeeded for server=%q", serverURL)

		return nil
	}

	logOAuthDebugf("automatic callback server unavailable for server=%q redirect_uri=%q: %v", serverURL, redirectURI, err)

	if onAutoCallbackUnavailable != nil {
		onAutoCallbackUnavailable(err)
	}

	if promptCallbackURL == nil {
		return err
	}

	oauthHandler, ok := any(h).(interface {
		GetAuthorizationURL(context.Context, string, string) (string, error)
		ProcessAuthorizationResponse(context.Context, string, string, string) error
	})
	if !ok {
		return fmt.Errorf("OAuth handler does not support interactive authorization")
	}

	authURL, err := oauthHandler.GetAuthorizationURL(ctx, state, codeChallenge)
	if err != nil {
		logOAuthDebugf("manual authorization URL build failed for server=%q: %v", serverURL, err)
		logOAuthHandlerDetails(ctx, serverURL, redirectURI, h)
		return fmt.Errorf("cannot build OAuth authorization URL: %w", err)
	}
	logOAuthDebugf("manual authorization URL for server=%q: %s", serverURL, authURL)
	if openBrowser != nil {
		openBrowser(authURL)
	}

	raw, err := promptCallbackURL()
	if err != nil {
		return err
	}

	code, returnedState, err := parseOAuthCallbackURL(raw)
	if err != nil {
		return err
	}

	err = oauthHandler.ProcessAuthorizationResponse(ctx, code, returnedState, codeVerifier)
	if err != nil {
		logOAuthDebugf("manual token exchange failed for server=%q: %v", serverURL, err)
		logOAuthHandlerDetails(ctx, serverURL, redirectURI, h)
		return fmt.Errorf("OAuth token exchange failed: %w", err)
	}
	logOAuthDebugf("manual token exchange succeeded for server=%q", serverURL)

	return nil
}

func ConnectHttpOAuthInteractive(overrides *mcp_config.OverrideT, serverURL string, oauthCfg OAuthConfig, hooks OAuthUIHooks) (*Client, error) {
	c, err := ConnectHttpOAuth(overrides, serverURL, oauthCfg)
	if err != nil && IsAuthorizationFailure(err) {
		err = AuthenticateOAuthInteractive(
			err,
			serverURL,
			oauthCfg.RedirectURI,
			overrides,
			hooks.OpenBrowser,
			hooks.PromptCallbackURL,
			hooks.OnAutoCallbackUnavailable,
		)
		if err != nil {
			return nil, err
		}

		return ConnectHttpOAuth(overrides, serverURL, oauthCfg)
	}

	return c, err
}

func parseOAuthCallbackURL(raw string) (code string, state string, err error) {
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
