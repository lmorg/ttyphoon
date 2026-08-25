package agent

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/claude"
	einoOllama "github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// summariserPrefix is prepended to every summarised tool output so the main
// agent knows the payload it received is a compression rather than raw data.
// The system prompt tells the model to treat this prefix as a signal that
// details (JSON schemas, exact numbers, IDs, etc.) may need to be re-fetched.
const summariserPrefix = "[output summarised by subagent]\n\n"

//go:embed summariser_prompt.md
var summariserSystemPrompt string

// summariseToolOutput runs a stateless one-shot chat completion using the same
// LLM service/model as the main agent to compress a single tool's output.
// Called by einoAgentTool.InvokableRun when the raw output would push the main
// agent's conversation over its context budget.
func (r *einoRuntime) summariseToolOutput(ctx context.Context, toolName, toolInput, rawOutput string) (string, error) {
	chatModel, err := r.newSummariserChatModel(ctx)
	if err != nil {
		return "", fmt.Errorf("build summariser chat model: %w", err)
	}

	messages := []*schema.Message{
		schema.SystemMessage(summariserSystemPrompt),
		schema.UserMessage(fmt.Sprintf(
			"Tool name: %s\nTool input arguments (JSON):\n%s\n\nTool raw output:\n%s",
			toolName, toolInput, rawOutput,
		)),
	}

	resp, err := chatModel.Generate(ctx, messages)
	if err != nil {
		return "", err
	}
	if resp == nil || strings.TrimSpace(resp.Content) == "" {
		return "", fmt.Errorf("empty summariser response")
	}
	return summariserPrefix + resp.Content, nil
}

// newSummariserChatModel constructs a fresh ChatModel matching the agent's
// configured LLM service. This is intentionally not the same instance as the
// react agent's ChatModel: the summariser must not share tool bindings or
// conversation history.
func (r *einoRuntime) newSummariserChatModel(ctx context.Context) (einoModel.BaseChatModel, error) {
	switch r.agent.ProviderName() {
	case LLM_OPENAI:
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:  r.agent.EnvironmentValue("OPENAI_API_KEY"),
			Model:   r.agent.SummariseModelName(),
			BaseURL: r.agent.EnvironmentValue("OPENAI_BASE_URL"),
			ByAzure: strings.EqualFold(strings.TrimSpace(r.agent.EnvironmentValue("OPENAI_BY_AZURE")), "true"),
		})

	case LLM_ANTHROPIC:
		var baseURL *string
		rawBaseURL := strings.TrimSpace(r.agent.EnvironmentValue("CLAUDE_BASE_URL"))
		if rawBaseURL != "" {
			baseURL = &rawBaseURL
		}
		// Thinking is deliberately disabled for the summariser: it's a mechanical
		// compression task, and the thinking budget would waste tokens we're
		// trying to save.
		return claude.NewChatModel(ctx, &claude.Config{
			APIKey:    r.agent.EnvironmentValue("ANTHROPIC_API_KEY"),
			BaseURL:   baseURL,
			Model:     r.agent.SummariseModelName(),
			MaxTokens: einoAnthropicMaxTokens,
		})

	case LLM_OLLAMA:
		baseURL := strings.TrimSpace(r.agent.EnvironmentValue("OLLAMA_HOST"))
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
			baseURL = "http://" + baseURL
		}
		return einoOllama.NewChatModel(ctx, &einoOllama.ChatModelConfig{
			BaseURL: baseURL,
			Model:   r.agent.SummariseModelName(),
		})

	default:
		return nil, fmt.Errorf("summariser: service %q is not supported", r.agent.ServiceName())
	}
}
