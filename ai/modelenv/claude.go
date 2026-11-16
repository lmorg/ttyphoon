// Package modelenv prepares provider configuration from scoped environments.
package modelenv

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/lmorg/ttyphoon/utils/envvars"
	"golang.org/x/net/http/httpguts"
)

// ClaudeConfig prevents unsupported scoped SDK settings from falling back to
// the process environment or profile configuration.
func ClaudeConfig(env *envvars.Environment, model string, maxTokens int) (claude.Config, error) {
	scoped := env.Overlay(nil)
	for _, name := range []string{
		"ANTHROPIC_PROFILE", "ANTHROPIC_CONFIG_DIR",
		"ANTHROPIC_FEDERATION_RULE_ID", "ANTHROPIC_ORGANIZATION_ID",
		"ANTHROPIC_IDENTITY_TOKEN_FILE", "ANTHROPIC_IDENTITY_TOKEN",
		"ANTHROPIC_WORKSPACE_ID", "ANTHROPIC_SERVICE_ACCOUNT_ID", "ANTHROPIC_SCOPE",
		"ANTHROPIC_WEBHOOK_SIGNING_KEY",
	} {
		if _, ok := scoped[name]; ok {
			return claude.Config{}, fmt.Errorf("scoped %s is not supported by the Anthropic adapter", name)
		}
	}

	apiKey, hasScopedKey := scoped["ANTHROPIC_API_KEY"]
	authToken, hasScopedToken := scoped["ANTHROPIC_AUTH_TOKEN"]
	if !hasScopedKey && !hasScopedToken {
		apiKey = env.Value("ANTHROPIC_API_KEY")
		authToken = env.Value("ANTHROPIC_AUTH_TOKEN")
	}
	if apiKey != "" {
		authToken = ""
	}
	if apiKey == "" && authToken == "" {
		for _, name := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_CUSTOM_HEADERS", "HOME", "XDG_CONFIG_HOME", "APPDATA"} {
			if _, ok := scoped[name]; ok {
				return claude.Config{}, fmt.Errorf("scoped %s requires a non-empty Anthropic API key or auth token", name)
			}
		}
	}

	if model == "" {
		model = env.Value("ANTHROPIC_MODEL")
	}
	if model == "" {
		return claude.Config{}, fmt.Errorf("no model specified")
	}

	var baseURL *string
	for _, name := range []string{"CLAUDE_BASE_URL", "ANTHROPIC_BASE_URL"} {
		if value, ok := scoped[name]; ok {
			value = strings.TrimSpace(value)
			if value == "" {
				return claude.Config{}, fmt.Errorf("scoped %s cannot be empty for the Anthropic adapter", name)
			}
			baseURL = &value
			break
		}
	}
	if baseURL == nil {
		if value := strings.TrimSpace(env.Value("CLAUDE_BASE_URL")); value != "" {
			baseURL = &value
		} else if value := strings.TrimSpace(env.Value("ANTHROPIC_BASE_URL")); value != "" {
			baseURL = &value
		}
	}

	var headers map[string]string
	if raw, ok := scoped["ANTHROPIC_CUSTOM_HEADERS"]; ok {
		headers = make(map[string]string)
		for _, line := range strings.Split(raw, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			name, value, ok := strings.Cut(line, ":")
			name, value = strings.TrimSpace(name), strings.TrimSpace(value)
			if !ok || !httpguts.ValidHeaderFieldName(name) || !httpguts.ValidHeaderFieldValue(value) {
				return claude.Config{}, fmt.Errorf("invalid scoped ANTHROPIC_CUSTOM_HEADERS")
			}
			name = http.CanonicalHeaderKey(name)
			if _, exists := headers[name]; exists {
				return claude.Config{}, fmt.Errorf("duplicate scoped ANTHROPIC_CUSTOM_HEADERS name")
			}
			headers[name] = value
		}
	}
	return claude.Config{
		APIKey: apiKey, AuthToken: authToken, BaseURL: baseURL, Model: model,
		MaxTokens: maxTokens, AdditionalHeaderFields: headers,
	}, nil
}
