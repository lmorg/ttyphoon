package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOAuthBuiltinWebviewDefaultAndOverride(t *testing.T) {
	var settings configT
	if err := yaml.Unmarshal(defaults, &settings); err != nil {
		t.Fatal(err)
	}
	if !settings.Ai.OAuthUseBuiltinWebview {
		t.Fatal("MCP OAuth must use the built-in webview by default")
	}
	if err := yaml.Unmarshal([]byte("AI:\n  OAuthUseBuiltinWebview: false\n"), &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Ai.OAuthUseBuiltinWebview {
		t.Fatal("explicit false did not select the external browser")
	}
}
