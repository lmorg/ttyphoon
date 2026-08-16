package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/cloudwego/eino-ext/components/model/claude"
	einoOllama "github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
)

type einoRuntime struct {
	agent      *Agent
	agentReact *react.Agent
}

const einoMaxHistoryTurns = 8

// Anthropic's output cap; too low truncates tool-call JSON args mid-stream (e.g. large HTML body fields), leaving invalid JSON.
const einoAnthropicMaxTokens = 8192

type aiStreamCallbackCtxKey struct{}

type einoAgentTool struct {
	delegate         aitypes.Tool
	info             *schema.ToolInfo
	unwrapInputField bool
}

func newEinoAgentTool(t aitypes.Tool) (*einoAgentTool, error) {
	if t == nil {
		return nil, fmt.Errorf("nil tool")
	}

	var (
		params           *schema.ParamsOneOf
		unwrapInputField bool
	)
	if mcp, ok := t.(*mcpTool); ok && len(mcp.schema) > 0 {
		var js jsonschema.Schema
		if err := json.Unmarshal(mcp.schema, &js); err != nil {
			return nil, fmt.Errorf("invalid schema for tool %q: %w", t.Name(), err)
		}
		params = schema.NewParamsOneOfByJSONSchema(&js)
	} else {
		// Anthropic requires an explicit input schema for custom tools.
		// Non-MCP tools in ttyphoon historically take a single raw string input,
		// so expose an object schema with one required `input` field.
		params = schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"input": {
				Type:     schema.String,
				Desc:     "Raw input string for the tool.",
				Required: true,
			},
		})
		unwrapInputField = true
	}

	return &einoAgentTool{
		delegate:         t,
		unwrapInputField: unwrapInputField,
		info: &schema.ToolInfo{
			Name:        sanitizeToolName(t.Name()),
			Desc:        t.Description(),
			ParamsOneOf: params,
		},
	}, nil
}

func (t *einoAgentTool) Info(context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *einoAgentTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einoTool.Option) (string, error) {
	emitAIStreamToolProgress(ctx, "Action: "+t.delegate.Name()+"\n")
	emitAIStreamToolProgress(ctx, "Action Input: "+argumentsInJSON+"\n")

	toolInput := argumentsInJSON
	if t.unwrapInputField {
		toolInput = unwrapToolInput(argumentsInJSON)
	}

	output, err := t.delegate.Call(ctx, toolInput)
	if err != nil {
		return output, err
	}

	if output != "" {
		emitAIStreamToolProgress(ctx, "Action Output: "+output+"\n")
	}

	return output, nil
}

func unwrapToolInput(argumentsInJSON string) string {
	type wrappedInput struct {
		Input string `json:"input"`
	}

	var wrapped wrappedInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &wrapped); err == nil {
		if wrapped.Input != "" {
			return wrapped.Input
		}
		return ""
	}

	var plain string
	if err := json.Unmarshal([]byte(argumentsInJSON), &plain); err == nil {
		return plain
	}

	return argumentsInJSON
}

func withAIStreamCallback(ctx context.Context, fn func(string)) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, aiStreamCallbackCtxKey{}, fn)
}

func emitAIStreamToolProgress(ctx context.Context, text string) {
	if text == "" || ctx == nil {
		return
	}
	v := ctx.Value(aiStreamCallbackCtxKey{})
	fn, ok := v.(func(string))
	if !ok || fn == nil {
		return
	}
	fn(text)
}

func (r *einoRuntime) toolsConfig() (compose.ToolsNodeConfig, error) {
	tools := make([]einoTool.BaseTool, 0)
	usedNames := make(map[string]struct{})

	for _, tool := range r.agent._tools {
		if !tool.Enabled() {
			continue
		}

		einoTool, err := newEinoAgentTool(tool)
		if err != nil {
			return compose.ToolsNodeConfig{}, err
		}

		info, err := einoTool.Info(context.Background())
		if err != nil {
			return compose.ToolsNodeConfig{}, err
		}

		uniqueName := uniqueToolName(info.Name, usedNames)
		info.Name = uniqueName
		usedNames[uniqueName] = struct{}{}

		tools = append(tools, einoTool)
	}

	return compose.ToolsNodeConfig{Tools: tools}, nil
}

func sanitizeToolName(name string) string {
	if name == "" {
		return "tool"
	}

	var b strings.Builder
	b.Grow(len(name))

	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('_')
	}

	safe := strings.Trim(b.String(), "_-")
	if safe == "" {
		safe = "tool"
	}

	// Anthropic currently enforces max length 128 for tool names.
	if len(safe) > 128 {
		safe = safe[:128]
	}

	return safe
}

func uniqueToolName(base string, used map[string]struct{}) string {
	if _, ok := used[base]; !ok {
		return base
	}

	const maxLen = 128
	for i := 2; ; i++ {
		suffix := fmt.Sprintf("_%d", i)
		name := base
		if len(name)+len(suffix) > maxLen {
			name = name[:maxLen-len(suffix)]
		}
		candidate := name + suffix
		if _, ok := used[candidate]; !ok {
			return candidate
		}
	}
}

func newEinoRuntime(agent *Agent) (*einoRuntime, error) {
	r := &einoRuntime{agent: agent}
	if err := r.init(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *einoRuntime) init() error {
	switch r.agent.ServiceName() {
	case LLM_OPENAI:
		return r.initOpenAI()
	case LLM_ANTHROPIC:
		return r.initAnthropic()
	case LLM_OLLAMA:
		return r.initOllama()
	default:
		return fmt.Errorf("service %q is not yet supported by the Eino runtime", r.agent.ServiceName())
	}
}

func (r *einoRuntime) initOpenAI() error {
	toolsConfig, err := r.toolsConfig()
	if err != nil {
		return err
	}

	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   r.agent.ModelName(),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		ByAzure: strings.EqualFold(strings.TrimSpace(os.Getenv("OPENAI_BY_AZURE")), "true"),
	})
	if err != nil {
		return err
	}

	r.agentReact, err = react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig:      toolsConfig,
		MaxStep:          r.agent.MaxIterations(),
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *einoRuntime) initAnthropic() error {
	toolsConfig, err := r.toolsConfig()
	if err != nil {
		return err
	}

	var baseURL *string
	rawBaseURL := strings.TrimSpace(os.Getenv("CLAUDE_BASE_URL"))
	if rawBaseURL != "" {
		baseURL = &rawBaseURL
	}

	chatModel, err := claude.NewChatModel(context.Background(), &claude.Config{
		APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
		BaseURL:   baseURL,
		Model:     r.agent.ModelName(),
		MaxTokens: einoAnthropicMaxTokens,
	})
	if err != nil {
		return err
	}

	r.agentReact, err = react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel:      chatModel,
		ToolsConfig:           toolsConfig,
		MaxStep:               r.agent.MaxIterations(),
		StreamToolCallChecker: streamToolCallCheckerAllChunks,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *einoRuntime) initOllama() error {
	toolsConfig, err := r.toolsConfig()
	if err != nil {
		return err
	}

	baseURL := strings.TrimSpace(os.Getenv("OLLAMA_HOST"))
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	chatModel, err := einoOllama.NewChatModel(context.Background(), &einoOllama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   r.agent.ModelName(),
	})
	if err != nil {
		return err
	}

	r.agentReact, err = react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig:      toolsConfig,
		MaxStep:          r.agent.MaxIterations(),
	})
	if err != nil {
		return err
	}

	return nil
}

// streamToolCallCheckerAllChunks scans the full stream for tool calls.
// This is safer for providers like Claude that may emit text before tool calls.
func streamToolCallCheckerAllChunks(_ context.Context, modelOutput *schema.StreamReader[*schema.Message]) (bool, error) {
	defer modelOutput.Close()

	for {
		msg, err := modelOutput.Recv()
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if msg == nil {
			continue
		}
		if len(msg.ToolCalls) > 0 {
			return true, nil
		}
	}
}

func (r *einoRuntime) Reset() {
	r.agentReact = nil
}

func buildEinoConversationMessages(history []sessiondb.Entry, currentMessages []*schema.Message) []*schema.Message {
	start := 0
	if len(history) > einoMaxHistoryTurns {
		start = len(history) - einoMaxHistoryTurns
	}

	messages := make([]*schema.Message, 0, ((len(history)-start)*2)+len(currentMessages))

	for i := start; i < len(history); i++ {
		query := strings.TrimSpace(history[i].Prompt)
		if query != "" {
			messages = append(messages, schema.UserMessage(query))
		}

		response := strings.TrimSpace(history[i].LLMResponse)
		if response != "" {
			messages = append(messages, schema.AssistantMessage(response, nil))
		}
	}

	messages = append(messages, currentMessages...)
	return messages
}

func (r *einoRuntime) RunLLMWithMessageStream(ctx context.Context, messages []*schema.Message, streamCallback func(string)) (string, error) {
	if r.agentReact == nil {
		if err := r.init(); err != nil {
			return "", err
		}
	}

	ctx = withAIStreamCallback(ctx, streamCallback)

	history, err := sessiondb.ActiveSessionEntries(r.agent.Workspace(), einoMaxHistoryTurns)
	if err != nil {
		return "", err
	}

	stream, err := r.agentReact.Stream(ctx, buildEinoConversationMessages(history, messages))
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var response strings.Builder
	for {
		msg, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			return response.String(), recvErr
		}

		if msg == nil || msg.Content == "" {
			continue
		}

		response.WriteString(msg.Content)
		if streamCallback != nil {
			streamCallback(msg.Content)
		}
	}

	return response.String(), nil
}

func (r *einoRuntime) RunLLMWithStream(ctx context.Context, prompt string, streamCallback func(string)) (string, error) {
	return r.RunLLMWithMessageStream(ctx, []*schema.Message{schema.UserMessage(prompt)}, streamCallback)
}
