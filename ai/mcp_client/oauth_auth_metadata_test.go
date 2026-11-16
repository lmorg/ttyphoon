package mcp_client

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/config"
)

func TestOAuthHandlerUsesAIWideEnvironment(t *testing.T) {
	t.Setenv("TTYPHOON_MCP_OAUTH_ENABLE_PRM_SHIM", "true")
	path := filepath.Join(t.TempDir(), "oauth.vars")
	write := func(value string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("TTYPHOON_MCP_OAUTH_ENABLE_PRM_SHIM="+value+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	previous := config.Config.Ai.EnvFiles
	config.Config.Ai.EnvFiles = []string{path}
	t.Cleanup(func() { config.Config.Ai.EnvFiles = previous })
	build := func() *gosdkOAuthHandler {
		t.Helper()
		handler, err := buildGoSDKAuthHandler(OAuthConfig{RedirectURI: "http://127.0.0.1:7700/"}, &mcp_config.OverrideT{}, OAuthUIHooks{})
		if err != nil {
			t.Fatal(err)
		}
		return handler.(*gosdkOAuthHandler)
	}
	write("")
	first := build()
	if first.forcePRMShim {
		t.Fatal("empty AI-wide value did not override the host")
	}
	t.Setenv("TTYPHOON_MCP_OAUTH_ENABLE_PRM_SHIM", "false")
	write("yes")
	second := build()
	if !second.forcePRMShim || first.forcePRMShim {
		t.Fatal("new OAuth handler did not use the updated AI-wide value")
	}
	if os.Getenv("TTYPHOON_MCP_OAUTH_ENABLE_PRM_SHIM") != "false" {
		t.Fatal("OAuth environment resolution changed the host")
	}
	config.Config.Ai.EnvFiles = nil
	t.Setenv("TTYPHOON_MCP_OAUTH_ENABLE_PRM_SHIM", " ON ")
	if !build().forcePRMShim {
		t.Fatal("OAuth handler did not preserve the host fallback")
	}
	config.Config.Ai.EnvFiles = []string{filepath.Join(t.TempDir(), "missing.vars")}
	if _, err := buildGoSDKAuthHandler(OAuthConfig{RedirectURI: "http://127.0.0.1:7700/"}, &mcp_config.OverrideT{}, OAuthUIHooks{}); err == nil || !strings.Contains(err.Error(), "resolve AI environment") {
		t.Fatalf("missing AI environment file was not reported: %v", err)
	}
}

func TestBuildDynamicClientRegistrationMetadata_UsesScopes(t *testing.T) {
	cfg := OAuthConfig{
		RedirectURI: "http://127.0.0.1:7700/",
		Scopes: []string{
			"offline_access",
			"read:jira-user",
			"read:jira-work",
		},
	}

	md := buildDynamicClientRegistrationMetadata(cfg)
	if md == nil {
		t.Fatalf("buildDynamicClientRegistrationMetadata() returned nil")
	}
	if md.Scope != "offline_access read:jira-user read:jira-work" {
		t.Fatalf("metadata.Scope = %q", md.Scope)
	}
}

func TestBuildDynamicClientRegistrationMetadata_EmptyScopes(t *testing.T) {
	cfg := OAuthConfig{RedirectURI: "http://127.0.0.1:7700/"}

	md := buildDynamicClientRegistrationMetadata(cfg)
	if md == nil {
		t.Fatalf("buildDynamicClientRegistrationMetadata() returned nil")
	}
	if md.Scope != "" {
		t.Fatalf("metadata.Scope = %q, want empty", md.Scope)
	}
}

func TestEnsureAuthorizationRequestScopes_AddsScopeWhenMissing(t *testing.T) {
	raw := "https://auth.example.com/authorize?response_type=code&client_id=abc"
	got := ensureAuthorizationRequestScopes(raw, []string{"offline_access", "read:jira-work"})

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse result URL: %v", err)
	}
	if u.Query().Get("scope") != "offline_access read:jira-work" {
		t.Fatalf("scope = %q", u.Query().Get("scope"))
	}
}

func TestEnsureAuthorizationRequestScopes_KeepsExistingScope(t *testing.T) {
	raw := "https://auth.example.com/authorize?scope=existing%3Ascope&client_id=abc"
	got := ensureAuthorizationRequestScopes(raw, []string{"offline_access", "read:jira-work"})

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse result URL: %v", err)
	}
	if u.Query().Get("scope") != "existing:scope" {
		t.Fatalf("scope = %q, want %q", u.Query().Get("scope"), "existing:scope")
	}
}
