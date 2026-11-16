package modelenv

import (
	"fmt"
	"strings"

	einoOllama "github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/lmorg/ttyphoon/utils/envvars"
)

func OpenAIConfig(env *envvars.Environment, model string) (openai.ChatModelConfig, error) {
	if value, ok := env.Overlay(nil)["OPENAI_BASE_URL"]; ok && strings.TrimSpace(value) == "" {
		return openai.ChatModelConfig{}, fmt.Errorf("scoped OPENAI_BASE_URL cannot be empty")
	}
	return openai.ChatModelConfig{
		APIKey: env.Value("OPENAI_API_KEY"), Model: model, BaseURL: env.Value("OPENAI_BASE_URL"),
		ByAzure: strings.EqualFold(strings.TrimSpace(env.Value("OPENAI_BY_AZURE")), "true"),
	}, nil
}

func OllamaConfig(env *envvars.Environment, model string) (einoOllama.ChatModelConfig, error) {
	baseURL := strings.TrimSpace(env.Value("OLLAMA_HOST"))
	if baseURL == "" {
		if _, ok := env.Overlay(nil)["OLLAMA_HOST"]; ok {
			return einoOllama.ChatModelConfig{}, fmt.Errorf("scoped OLLAMA_HOST cannot be empty")
		}
		baseURL = "http://localhost:11434"
	} else if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	return einoOllama.ChatModelConfig{BaseURL: baseURL, Model: model}, nil
}
