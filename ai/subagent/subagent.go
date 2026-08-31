// Package subagent runs stateless, tool-free LLM calls for an AI agent.
package subagent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/claude"
	einoOllama "github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const EmitInterval = 250 * time.Millisecond

const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderOllama    = "ollama"
)

type modelFactory func(context.Context) (model.BaseChatModel, error)

type Client struct {
	newModel modelFactory
}

func New(provider, modelName string, environmentValue func(string) string) *Client {
	return &Client{newModel: func(ctx context.Context) (model.BaseChatModel, error) {
		switch provider {
		case ProviderOpenAI:
			return openai.NewChatModel(ctx, &openai.ChatModelConfig{
				APIKey:  environmentValue("OPENAI_API_KEY"),
				Model:   modelName,
				BaseURL: environmentValue("OPENAI_BASE_URL"),
				ByAzure: strings.EqualFold(strings.TrimSpace(environmentValue("OPENAI_BY_AZURE")), "true"),
			})
		case ProviderAnthropic:
			var baseURL *string
			if value := strings.TrimSpace(environmentValue("CLAUDE_BASE_URL")); value != "" {
				baseURL = &value
			}
			return claude.NewChatModel(ctx, &claude.Config{
				APIKey: environmentValue("ANTHROPIC_API_KEY"), BaseURL: baseURL,
				Model: modelName, MaxTokens: 8192,
			})
		case ProviderOllama:
			baseURL := strings.TrimSpace(environmentValue("OLLAMA_HOST"))
			if baseURL == "" {
				baseURL = "http://localhost:11434"
			} else if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
				baseURL = "http://" + baseURL
			}
			return einoOllama.NewChatModel(ctx, &einoOllama.ChatModelConfig{BaseURL: baseURL, Model: modelName})
		default:
			return nil, fmt.Errorf("sub-agent service %q is not supported", provider)
		}
	}}
}

type Request struct {
	Name              string
	SystemPrompt      string
	Prompt            string
	EmitStream        func(string)
	StreamPrefix      string
	StreamSuffix      string
	FormatStreamChunk func(string) string
	RunWithTools      func(context.Context, string, func(string)) (string, error)
}

func (c *Client) Run(ctx context.Context, request Request) (string, error) {
	if c == nil {
		return "", errors.New("sub-agent client is required")
	}

	var (
		pending    strings.Builder
		lastEmit   = time.Now()
		flushTimer *time.Timer
		emitMu     sync.Mutex
	)
	var flush func()
	flushLocked := func() {
		if flushTimer != nil {
			flushTimer.Stop()
			flushTimer = nil
		}
		if pending.Len() == 0 || request.EmitStream == nil {
			return
		}
		request.EmitStream(pending.String())
		pending.Reset()
		lastEmit = time.Now()
	}
	flush = func() {
		emitMu.Lock()
		defer emitMu.Unlock()
		flushLocked()
	}
	if request.EmitStream != nil {
		prefix := request.StreamPrefix
		if prefix == "" {
			prefix = fmt.Sprintf("\n> **Sub-agent %s:** ", request.Name)
		}
		suffix := request.StreamSuffix
		if suffix == "" {
			suffix = "\n\n"
		}
		request.EmitStream(prefix)
		defer request.EmitStream(suffix)
		defer flush()
	}
	emitChunk := func(text string) {
		if text == "" {
			return
		}
		emitMu.Lock()
		formatChunk := request.FormatStreamChunk
		if formatChunk == nil {
			formatChunk = Quote
		}
		pending.WriteString(formatChunk(text))
		if time.Since(lastEmit) >= EmitInterval {
			flushLocked()
		} else if flushTimer == nil && request.EmitStream != nil {
			flushTimer = time.AfterFunc(EmitInterval-time.Since(lastEmit), flush)
		}
		emitMu.Unlock()
	}

	if request.RunWithTools != nil {
		content, err := request.RunWithTools(ctx, request.Prompt, emitChunk)
		if err != nil {
			return content, err
		}
		if strings.TrimSpace(content) == "" {
			return "", errors.New("empty sub-agent response")
		}
		return content, nil
	}
	if c.newModel == nil {
		return "", errors.New("sub-agent model factory is required")
	}

	chatModel, err := c.newModel(ctx)
	if err != nil {
		return "", fmt.Errorf("build sub-agent chat model: %w", err)
	}
	messages := []*schema.Message{schema.UserMessage(request.Prompt)}
	if request.SystemPrompt != "" {
		messages = append([]*schema.Message{schema.SystemMessage(request.SystemPrompt)}, messages...)
	}
	stream, err := chatModel.Stream(ctx, messages)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var content strings.Builder
	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return "", recvErr
		}
		if chunk == nil || chunk.Content == "" {
			continue
		}
		content.WriteString(chunk.Content)
		emitChunk(chunk.Content)
	}
	if strings.TrimSpace(content.String()) == "" {
		return "", errors.New("empty sub-agent response")
	}
	return content.String(), nil
}

func Quote(text string) string {
	return strings.ReplaceAll(text, "\n", "\n> ")
}
