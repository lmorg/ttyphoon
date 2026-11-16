package modelenv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/lmorg/ttyphoon/utils/envvars"
)

func TestOpenAIUsesScopedCredentialsAndEndpoint(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "host-key")
	t.Setenv("OPENAI_BASE_URL", "http://127.0.0.1:1")
	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-test","object":"chat.completion","model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()
	env, err := envvars.Resolve(nil, map[string]string{"OPENAI_API_KEY": "service-key", "OPENAI_BASE_URL": server.URL})
	if err != nil {
		t.Fatal(err)
	}
	settings, err := OpenAIConfig(env, "test-model")
	if err != nil {
		t.Fatal(err)
	}
	client, err := openai.NewChatModel(context.Background(), &settings)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Generate(context.Background(), []*schema.Message{schema.UserMessage("hello")}); err != nil {
		t.Fatal(err)
	}
	if <-requests != "Bearer service-key" || os.Getenv("OPENAI_API_KEY") != "host-key" {
		t.Fatal("OpenAI request did not retain the scoped credential")
	}
}

func TestClaudeUsesScopedSettingsInsteadOfHostSDKDefaults(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "host-key")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "host-token")
	t.Setenv("ANTHROPIC_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("ANTHROPIC_CUSTOM_HEADERS", "X-Host: host-header")
	requests := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_test","type":"message","role":"assistant","model":"test","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "claude.vars")
	content := "ANTHROPIC_API_KEY=scoped-key\nANTHROPIC_BASE_URL=" + server.URL + "\nANTHROPIC_CUSTOM_HEADERS=X-Scoped: scoped-header\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	env, err := envvars.Resolve([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	settings, err := ClaudeConfig(env, "test-model", 64)
	if err != nil {
		t.Fatal(err)
	}
	client, err := claude.NewChatModel(context.Background(), &settings)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Generate(context.Background(), []*schema.Message{schema.UserMessage("hello")}); err != nil {
		t.Fatalf("Claude request failed: %v", err)
	}
	header := <-requests
	if header.Get("X-Api-Key") != "scoped-key" || header.Get("X-Scoped") != "scoped-header" || header.Get("X-Host") != "" || header.Get("Authorization") != "" {
		t.Fatal("Claude request did not use only the scoped credentials and headers")
	}
	if got := os.Getenv("ANTHROPIC_API_KEY"); got != "host-key" {
		t.Fatal("host credentials were changed")
	}
}

func TestClaudeConcurrentServicesUseSeparateCredentials(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "host-key")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Get("X-Api-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_test","type":"message","role":"assistant","model":"test","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()
	errors := make(chan error, 2)
	for _, key := range []string{"service-one", "service-two"} {
		go func() {
			env, err := envvars.Resolve(nil, map[string]string{"ANTHROPIC_API_KEY": key, "ANTHROPIC_BASE_URL": server.URL})
			if err == nil {
				var settings claude.Config
				settings, err = ClaudeConfig(env, "test-model", 64)
				if err == nil {
					var client *claude.ChatModel
					client, err = claude.NewChatModel(context.Background(), &settings)
					if err == nil {
						_, err = client.Generate(context.Background(), []*schema.Message{schema.UserMessage("hello")})
					}
				}
			}
			errors <- err
		}()
	}
	for range 2 {
		if err := <-errors; err != nil {
			t.Fatalf("Claude request failed: %v", err)
		}
	}
	keys := map[string]bool{<-requests: true, <-requests: true}
	if !keys["service-one"] || !keys["service-two"] || os.Getenv("ANTHROPIC_API_KEY") != "host-key" {
		t.Fatal("concurrent Claude services did not retain separate credentials")
	}
}

func TestClaudeScopedSettingsFailClosed(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "host-key")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "host-token")
	t.Setenv("ANTHROPIC_MODEL", "host-model")
	for _, test := range []struct {
		name   string
		values map[string]string
		model  string
		want   string
	}{
		{"empty key", map[string]string{"ANTHROPIC_API_KEY": ""}, "model", "ANTHROPIC_API_KEY"},
		{"profile", map[string]string{"ANTHROPIC_PROFILE": "secret-profile"}, "model", "ANTHROPIC_PROFILE"},
		{"empty base", map[string]string{"ANTHROPIC_API_KEY": "key", "ANTHROPIC_BASE_URL": ""}, "model", "ANTHROPIC_BASE_URL"},
		{"empty model", map[string]string{"ANTHROPIC_MODEL": ""}, "", "no model specified"},
		{"invalid header", map[string]string{"ANTHROPIC_API_KEY": "key", "ANTHROPIC_CUSTOM_HEADERS": "secret-header"}, "model", "ANTHROPIC_CUSTOM_HEADERS"},
	} {
		t.Run(test.name, func(t *testing.T) {
			env, err := envvars.Resolve(nil, test.values)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ClaudeConfig(env, test.model, 64)
			if err == nil || !strings.Contains(err.Error(), test.want) || strings.Contains(err.Error(), "secret-") {
				t.Fatalf("ClaudeConfig() error = %v, want safe %s error", err, test.want)
			}
		})
	}
	env, err := envvars.Resolve(nil, map[string]string{"ANTHROPIC_AUTH_TOKEN": "scoped-token"})
	if err != nil {
		t.Fatal(err)
	}
	settings, err := ClaudeConfig(env, "model", 64)
	if err != nil || settings.APIKey != "" || settings.AuthToken != "scoped-token" {
		t.Fatalf("scoped token did not override host credential: %v", err)
	}
}

func TestProviderScopedEmptyEndpointsDoNotUseDefaults(t *testing.T) {
	for _, test := range []struct {
		name string
		call func(*envvars.Environment) error
	}{
		{"OPENAI_BASE_URL", func(env *envvars.Environment) error { _, err := OpenAIConfig(env, "model"); return err }},
		{"OLLAMA_HOST", func(env *envvars.Environment) error { _, err := OllamaConfig(env, "model"); return err }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.name, "host-value")
			env, err := envvars.Resolve(nil, map[string]string{test.name: ""})
			if err != nil {
				t.Fatal(err)
			}
			if err := test.call(env); err == nil {
				t.Fatalf("empty scoped %s unexpectedly fell back", test.name)
			}
		})
	}
}
