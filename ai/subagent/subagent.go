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
	"github.com/lmorg/ttyphoon/ai/modelenv"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/envvars"
)

const EmitInterval = types.AI_EMIT_INTERVAL * time.Millisecond

const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderOllama    = "ollama"
)

type modelFactory func(context.Context) (model.BaseChatModel, error)

type Client struct {
	newModel modelFactory
}

func New(provider, modelName string, environment *envvars.Environment) *Client {
	return &Client{newModel: func(ctx context.Context) (model.BaseChatModel, error) {
		switch provider {
		case ProviderOpenAI:
			settings, err := modelenv.OpenAIConfig(environment, modelName)
			if err != nil {
				return nil, err
			}
			return openai.NewChatModel(ctx, &settings)
		case ProviderAnthropic:
			settings, err := modelenv.ClaudeConfig(environment, modelName, 8192)
			if err != nil {
				return nil, err
			}
			return claude.NewChatModel(ctx, &settings)
		case ProviderOllama:
			settings, err := modelenv.OllamaConfig(environment, modelName)
			if err != nil {
				return nil, err
			}
			return einoOllama.NewChatModel(ctx, &settings)
		default:
			return nil, fmt.Errorf("sub-agent service %q is not supported", provider)
		}
	}}
}

type Request struct {
	Name                 string
	SystemPrompt         string
	Prompt               string
	EmitStream           func(string)
	StreamPrefix         string
	StreamSuffix         string
	FormatStreamChunk    func(string) string
	DisableStreamFraming bool
	RunWithTools         func(ctx context.Context, systemPrompt, prompt string, emit func(string)) (string, error)
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
		defer flush()
		if !request.DisableStreamFraming {
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
		}
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
		content, err := request.RunWithTools(ctx, request.SystemPrompt, request.Prompt, emitChunk)
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
