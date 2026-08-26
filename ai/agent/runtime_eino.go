package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/cloudwego/eino-ext/components/model/claude"
	einoOllama "github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	einoModel "github.com/cloudwego/eino/components/model"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	einoAgent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	einoCBUtils "github.com/cloudwego/eino/utils/callbacks"
	"github.com/eino-contrib/jsonschema"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
	"github.com/lmorg/ttyphoon/config"
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
	runtime          *einoRuntime
	delegate         aitypes.Tool
	info             *schema.ToolInfo
	unwrapInputField bool
}

func newEinoAgentTool(r *einoRuntime, t aitypes.Tool) (*einoAgentTool, error) {
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
		runtime:          r,
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
	emitAIStreamToolProgress(ctx, formatToolCallMarkdown(t.delegate.Name(), argumentsInJSON))

	toolInput := argumentsInJSON
	if t.unwrapInputField {
		toolInput = unwrapToolInput(argumentsInJSON)
	}

	output, err := t.delegate.Call(ctx, toolInput)
	if err != nil {
		emitAIStreamToolProgress(ctx, formatToolErrorMarkdown(err))
		return output, err
	}

	// Interposer: compress oversized tool outputs so the main agent's context stays
	// under the LLM's window. The main agent only ever sees the summarised form.
	threshold := config.Config.Ai.ToolSummariseThresholdChars
	summarising := t.runtime != nil && threshold > 0 && len(output) > threshold

	// When summarising, the streamed summary replaces the raw output in the UI + log.
	if output != "" && !summarising {
		emitAIStreamToolProgress(ctx, formatToolOutputMarkdown(output))
	}

	if summarising {
		summary, sErr := t.runtime.summariseToolOutput(ctx, t.delegate.Name(), argumentsInJSON, output)
		if sErr != nil {
			emitAIStreamToolProgress(ctx, formatToolSummaryFailureMarkdown(len(output), sErr))
			return fmt.Sprintf("[tool output too large, summariser failed: %s]", sErr), nil
		}
		emitAIStreamToolProgress(ctx, formatToolSummaryNoticeMarkdown(len(output), len(summary)))
		return summary, nil
	}

	return output, nil
}

// Tool progress is emitted as real markdown with ~~~~ tilde fences (rather than
// ``` triple-backticks) so tool arguments or outputs that themselves contain
// ``` blocks don't prematurely close the fence and leak into the surrounding UI.

func formatToolCallMarkdown(name, argumentsInJSON string) string {
	return fmt.Sprintf("\n\n**Tool call:** `%s`\n\n~~~~json\n%s\n~~~~\n\n", name, argumentsInJSON)
}

func formatToolOutputMarkdown(output string) string {
	return fmt.Sprintf("**Tool output:**\n\n~~~~\n%s\n~~~~\n\n", output)
}

func formatToolErrorMarkdown(err error) string {
	return fmt.Sprintf("**Tool error:**\n\n~~~~\n%s\n~~~~\n\n", err.Error())
}

func summariserStreamOpenMarkdown() string {
	return "**Summaring tool output:**\n\n~~~~\n"
}

func summariserStreamCloseMarkdown() string {
	return "\n~~~~\n\n"
}

func formatToolSummaryNoticeMarkdown(rawLen, summaryLen int) string {
	return fmt.Sprintf("_Output summarised for main agent (%d \u2192 %d chars)._\n\n", rawLen, summaryLen)
}

func formatToolSummaryFailureMarkdown(rawLen int, err error) string {
	return fmt.Sprintf("_Output was %d chars; summariser failed and the main agent received a placeholder instead: %s_\n\n", rawLen, err.Error())
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

// aiStreamEmitter serialises text and reasoning chunks onto a single stream,
// wrapping reasoning in a markdown blockquote that opens on the first reasoning
// chunk and closes when non-reasoning content follows.
type aiStreamEmitter struct {
	mu         sync.Mutex
	fn         func(string)
	inThinking bool
}

func (e *aiStreamEmitter) emitText(text string) {
	if e == nil || e.fn == nil || text == "" {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.inThinking {
		e.fn("\n\n")
		e.inThinking = false
	}
	e.fn(text)
}

func (e *aiStreamEmitter) emitReasoning(text string) {
	if e == nil || e.fn == nil || text == "" {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.inThinking {
		e.fn("\n> **Thinking:** ")
		e.inThinking = true
	}
	e.fn(strings.ReplaceAll(text, "\n", "\n> "))
}

func withAIStreamCallback(ctx context.Context, emitter *aiStreamEmitter) context.Context {
	if emitter == nil {
		return ctx
	}
	return context.WithValue(ctx, aiStreamCallbackCtxKey{}, emitter)
}

func emitAIStreamToolProgress(ctx context.Context, text string) {
	if text == "" || ctx == nil {
		return
	}
	if emitter, ok := ctx.Value(aiStreamCallbackCtxKey{}).(*aiStreamEmitter); ok {
		emitter.emitText(text)
	}
}

func (r *einoRuntime) toolsConfig() (compose.ToolsNodeConfig, error) {
	tools := make([]einoTool.BaseTool, 0)
	usedNames := make(map[string]struct{})

	for _, tool := range r.agent._tools {
		if !tool.Enabled() {
			continue
		}

		einoTool, err := newEinoAgentTool(r, tool)
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
	switch r.agent.ProviderName() {
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
		APIKey:  r.agent.EnvironmentValue("OPENAI_API_KEY"),
		Model:   r.agent.ModelName(),
		BaseURL: r.agent.EnvironmentValue("OPENAI_BASE_URL"),
		ByAzure: strings.EqualFold(strings.TrimSpace(r.agent.EnvironmentValue("OPENAI_BY_AZURE")), "true"),
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
	rawBaseURL := strings.TrimSpace(r.agent.EnvironmentValue("CLAUDE_BASE_URL"))
	if rawBaseURL != "" {
		baseURL = &rawBaseURL
	}

	chatModel, err := claude.NewChatModel(context.Background(), &claude.Config{
		APIKey:    r.agent.EnvironmentValue("ANTHROPIC_API_KEY"),
		BaseURL:   baseURL,
		Model:     r.agent.ModelName(),
		MaxTokens: einoAnthropicMaxTokens,
		ThinkingConfig: &anthropic.ThinkingConfigParamUnion{
			OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{},
		},
		AdditionalRequestFields: map[string]any{
			"output_config": map[string]any{
				"effort": "high",
			},
		},
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

	baseURL := strings.TrimSpace(r.agent.EnvironmentValue("OLLAMA_HOST"))
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

	// Anthropic requires system messages to precede all user and assistant messages.
	// Prompt builders place the system message before the current user message,
	// but restored history means it would otherwise be inserted mid-conversation.
	systemMessages := make([]*schema.Message, 0, len(currentMessages))
	for _, message := range currentMessages {
		if message != nil && message.Role == schema.System {
			systemMessages = append(systemMessages, message)
		}
	}
	messages = append(systemMessages, messages...)
	for _, message := range currentMessages {
		if message == nil || message.Role == schema.System {
			continue
		}
		messages = append(messages, message)
	}
	return messages
}

func (r *einoRuntime) RunLLMWithMessageStream(ctx context.Context, messages []*schema.Message, streamCallback func(string)) (string, error) {
	if r.agentReact == nil {
		if err := r.init(); err != nil {
			return "", err
		}
	}

	emitter := &aiStreamEmitter{fn: streamCallback}
	ctx = withAIStreamCallback(ctx, emitter)

	history, err := sessiondb.ActiveSessionEntries(r.agent.Workspace(), einoMaxHistoryTurns)
	if err != nil {
		return "", err
	}

	// The react agent's stream branch consumes intermediate model turns to detect tool calls,
	// so reasoning content emitted before tool calls never reaches the outer stream. Hook a
	// per-model-node callback to observe every turn's stream and forward reasoning to the emitter.
	reasoningWait := &sync.WaitGroup{}
	modelHandler := &einoCBUtils.ModelCallbackHandler{
		OnEndWithStreamOutput: func(cbCtx context.Context, _ *callbacks.RunInfo, stream *schema.StreamReader[*einoModel.CallbackOutput]) context.Context {
			reasoningWait.Add(1)
			go func() {
				defer reasoningWait.Done()
				drainReasoningStream(stream, emitter)
			}()
			return cbCtx
		},
	}
	cbHandler := einoCBUtils.NewHandlerHelper().ChatModel(modelHandler).Handler()

	stream, err := r.agentReact.Stream(ctx,
		buildEinoConversationMessages(history, messages),
		einoAgent.WithComposeOptions(compose.WithCallbacks(cbHandler)),
	)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	// Reasoning is streamed to the frontend and log via the callback above; the outer loop only
	// accumulates Content into the returned string used for conversation history.
	var response strings.Builder
	var chunkCount, contentChunks, contentBytes int
	streamStart := time.Now()
	var firstContentAt, lastContentAt time.Time
	for {
		msg, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			reasoningWait.Wait()
			return response.String(), recvErr
		}

		if msg == nil {
			continue
		}
		chunkCount++

		if msg.Content != "" {
			contentChunks++
			contentBytes += len(msg.Content)
			if firstContentAt.IsZero() {
				firstContentAt = time.Now()
			}
			lastContentAt = time.Now()
			response.WriteString(msg.Content)
			emitter.emitText(msg.Content)
		}
	}
	reasoningWait.Wait()

	spread := time.Duration(0)
	if !firstContentAt.IsZero() && !lastContentAt.IsZero() {
		spread = lastContentAt.Sub(firstContentAt)
	}
	log.Printf(
		"[debug] AI outer stream drained: chunks=%d content_chunks=%d content_bytes=%d ttfb=%s content_spread=%s total=%s",
		chunkCount, contentChunks, contentBytes,
		firstContentAt.Sub(streamStart).Round(time.Millisecond),
		spread.Round(time.Millisecond),
		time.Since(streamStart).Round(time.Millisecond),
	)

	return response.String(), nil
}

func drainReasoningStream(stream *schema.StreamReader[*einoModel.CallbackOutput], emitter *aiStreamEmitter) {
	defer stream.Close()
	start := time.Now()
	var reasoningChunks, contentChunks int
	var reasoningBytes, contentBytes int
	for {
		out, err := stream.Recv()
		if err == io.EOF {
			log.Printf(
				"[debug] AI model-turn stream drained: reasoning_chunks=%d reasoning_bytes=%d content_chunks=%d content_bytes=%d elapsed=%s",
				reasoningChunks, reasoningBytes, contentChunks, contentBytes,
				time.Since(start).Round(time.Millisecond),
			)
			return
		}
		if err != nil {
			return
		}
		if out == nil || out.Message == nil {
			continue
		}
		if out.Message.ReasoningContent != "" {
			reasoningChunks++
			reasoningBytes += len(out.Message.ReasoningContent)
			emitter.emitReasoning(out.Message.ReasoningContent)
		}
		if out.Message.Content != "" {
			contentChunks++
			contentBytes += len(out.Message.Content)
		}
	}
}

func (r *einoRuntime) RunLLMWithStream(ctx context.Context, prompt string, streamCallback func(string)) (string, error) {
	return r.RunLLMWithMessageStream(ctx, []*schema.Message{schema.UserMessage(prompt)}, streamCallback)
}
