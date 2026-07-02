package mcp_client

import (
	"net/url"
	"testing"
)

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
