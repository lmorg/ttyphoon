package mcp_client

import (
	"testing"

	"github.com/lmorg/ttyphoon/ai/mcp_config"
)

func TestBuildOAuthConfig_NoDefaultAtlassianScopesWhenUnset(t *testing.T) {
	cfg := BuildOAuthConfig("", "atlassian", "https://mcp.atlassian.com/v1/mcp/authv2", nil)
	if len(cfg.Scopes) != 0 {
		t.Fatalf("BuildOAuthConfig() scopes = %v, want empty", cfg.Scopes)
	}
}

func TestBuildOAuthConfig_ExplicitScopesOverrideDefaults(t *testing.T) {
	cfg := BuildOAuthConfig("", "atlassian", "https://mcp.atlassian.com/v1/mcp/authv2", &mcp_config.OAuthT{Scopes: []string{"custom:scope"}})
	if len(cfg.Scopes) != 1 || cfg.Scopes[0] != "custom:scope" {
		t.Fatalf("BuildOAuthConfig() scopes = %v, want %v", cfg.Scopes, []string{"custom:scope"})
	}
}

func TestBuildOAuthConfig_NoDefaultScopesForNonAtlassian(t *testing.T) {
	cfg := BuildOAuthConfig("", "demo", "https://example.com/mcp", nil)
	if len(cfg.Scopes) != 0 {
		t.Fatalf("BuildOAuthConfig() scopes = %v, want empty", cfg.Scopes)
	}
}
