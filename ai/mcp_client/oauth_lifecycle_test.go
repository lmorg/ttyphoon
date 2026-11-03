package mcp_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/ai/mcp_config"
	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestOAuthCancelledSetupCanRetryAutomaticallyWithIssuerValidation(t *testing.T) {
	probe, err := StartOAuthCallbackServer("http://127.0.0.1:0/")
	if err != nil {
		t.Fatal(err)
	}
	redirect := "http://" + probe.listener.Addr().String() + "/"
	probe.Close()
	server := mcp_sdk.NewServer(&mcp_sdk.Implementation{Name: "OAuth fixture", Version: "1"}, nil)
	server.AddTool(&mcp_sdk.Tool{Name: "echo", InputSchema: map[string]any{"type": "object"}},
		func(context.Context, *mcp_sdk.CallToolRequest) (*mcp_sdk.CallToolResult, error) {
			return &mcp_sdk.CallToolResult{}, nil
		})
	handler := mcp_sdk.NewStreamableHTTPHandler(func(*http.Request) *mcp_sdk.Server { return server }, nil)
	var base string
	var rejectAuthorized atomic.Bool
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/oauth-protected-resource":
			json.NewEncoder(w).Encode(map[string]any{"resource": base + "/mcp", "authorization_servers": []string{base}})
		case "/.well-known/oauth-authorization-server":
			json.NewEncoder(w).Encode(map[string]any{
				"issuer": base, "authorization_endpoint": base + "/authorize", "token_endpoint": base + "/token",
				"registration_endpoint": base + "/register", "response_types_supported": []string{"code"},
				"code_challenge_methods_supported": []string{"S256"}, "token_endpoint_auth_methods_supported": []string{"none"},
				"authorization_response_iss_parameter_supported": true,
			})
		case "/register":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"client_id": "fixture-client", "token_endpoint_auth_method": "none", "redirect_uris": []string{redirect}})
		case "/token":
			json.NewEncoder(w).Encode(map[string]any{"access_token": "fixture-token", "token_type": "Bearer", "expires_in": 3600})
		case "/mcp":
			if r.Header.Get("Authorization") != "Bearer fixture-token" || rejectAuthorized.Load() {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s/.well-known/oauth-protected-resource"`, base))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			handler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	base = provider.URL
	config := &mcp_config.OAuthT{RedirectURI: redirect, TokenFile: filepath.Join(t.TempDir(), "token.json")}
	var manual atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	hooks := OAuthUIHooks{OpenBrowser: func(string) { cancel() }, PromptCallbackURL: func() (string, error) {
		manual.Add(1)
		return "", errors.New("manual callback must never be used")
	}}
	redirectURL, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	occupied, err := net.Listen("tcp", redirectURL.Host)
	if err != nil {
		t.Fatal(err)
	}
	busyCtx, stopBusy := context.WithTimeout(context.Background(), time.Second)
	err = ConnectAndUseHttpContext(busyCtx, &mcp_config.OverrideT{}, "fixture", "server", base+"/mcp", config, hooks, nil,
		func(client *Client) error { return client.ListToolsContext(busyCtx) })
	stopBusy()
	occupied.Close()
	if err == nil || !strings.Contains(err.Error(), "automatic OAuth callback") || manual.Load() != 0 {
		t.Fatalf("busy callback port attempted a manual fallback: %v, prompts=%d", err, manual.Load())
	}
	err = ConnectAndUseHttpContext(ctx, &mcp_config.OverrideT{}, "fixture", "server", base+"/mcp", config, hooks, nil,
		func(client *Client) error { return client.ListToolsContext(ctx) })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled setup returned %v", err)
	}
	var browser atomic.Int32
	hooks.OpenBrowser = func(raw string) {
		browser.Add(1)
		authorization, err := url.Parse(raw)
		if err != nil {
			t.Error(err)
			return
		}
		callback := redirect + "?" + url.Values{"code": {"fixture-code"}, "state": {authorization.Query().Get("state")}, "iss": {base}}.Encode()
		response, err := http.Get(callback)
		if err != nil {
			t.Error(err)
			return
		}
		response.Body.Close()
	}
	setupCtx, finishSetup := context.WithCancel(context.Background())
	var connected *Client
	err = ConnectAndUseHttpContext(setupCtx, &mcp_config.OverrideT{}, "fixture", "server", base+"/mcp", config, hooks, nil,
		func(client *Client) error { connected = client; return client.ListToolsContext(setupCtx) })
	finishSetup()
	if err != nil {
		t.Fatal(err)
	}
	defer connected.Close()
	if browser.Load() != 1 || manual.Load() != 0 {
		t.Fatalf("browser=%d manual=%d", browser.Load(), manual.Load())
	}
	if err := connected.ListTools(); err != nil {
		t.Fatalf("finishing setup killed the live MCP session: %v", err)
	}
	rejectAuthorized.Store(true)
	if err := connected.ListTools(); err == nil {
		t.Fatal("accepted a rejected fresh token")
	}
	if browser.Load() != 1 {
		t.Fatal("a rejected fresh token reopened browser authorization")
	}
	rejectAuthorized.Store(false)
	if err := connected.ListTools(); err != nil {
		t.Fatalf("authorization failure poisoned the session: %v", err)
	}
}

func TestOAuthCallbackIgnoresOldStateAndPreservesIssuer(t *testing.T) {
	callback, err := startOAuthCallbackServer("http://127.0.0.1:0/", "current")
	if err != nil {
		t.Fatal(err)
	}
	defer callback.Close()
	base := "http://" + callback.listener.Addr().String() + "/"
	for _, state := range []string{"old", "current"} {
		response, err := http.Get(base + "?" + url.Values{"code": {"code"}, "state": {state}, "iss": {"https://issuer.example"}}.Encode())
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if state == "old" && response.StatusCode != http.StatusBadRequest {
			t.Fatal("stale callback was accepted")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := callback.WaitContext(ctx)
	if err != nil || result.State != "current" || result.Iss != "https://issuer.example" {
		t.Fatalf("callback lost state/issuer: %+v, %v", result, err)
	}
}

func TestOAuthCallbackReportsProviderDenial(t *testing.T) {
	callback, err := startOAuthCallbackServer("http://127.0.0.1:0/", "current")
	if err != nil {
		t.Fatal(err)
	}
	defer callback.Close()
	response, err := http.Get("http://" + callback.listener.Addr().String() + "/?state=current&error=access_denied")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := callback.WaitContext(ctx); err == nil || !strings.Contains(err.Error(), "access_denied") {
		t.Fatalf("provider denial returned %v", err)
	}
}
