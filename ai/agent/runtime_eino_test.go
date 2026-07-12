package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

type fakeAgentTool struct {
	enabled bool
	input   string
}

func (f *fakeAgentTool) New(_ aitypes.Agent) (aitypes.Tool, error) { return f, nil }
func (f *fakeAgentTool) Enabled() bool                             { return f.enabled }
func (f *fakeAgentTool) Toggle()                                   { f.enabled = !f.enabled }
func (f *fakeAgentTool) Name() string                              { return "fake.tool" }
func (f *fakeAgentTool) Path() string                              { return "internal" }
func (f *fakeAgentTool) Description() string                       { return "fake" }
func (f *fakeAgentTool) Call(_ context.Context, s string) (string, error) {
	f.input = s
	return "", nil
}

type fakeRuntime struct {
	result string
	err    error
	chunks []string

	called bool
	ctx    context.Context
	prompt string

	resetCalled bool
}

func (f *fakeRuntime) RunLLMWithStream(ctx context.Context, prompt string, streamCallback func(string)) (string, error) {
	f.called = true
	f.ctx = ctx
	f.prompt = prompt
	if streamCallback != nil {
		if len(f.chunks) == 0 {
			streamCallback("chunk")
		} else {
			for _, chunk := range f.chunks {
				streamCallback(chunk)
			}
		}
	}
	return f.result, f.err
}

func (f *fakeRuntime) Reset() {
	f.resetCalled = true
}

func TestNewPreferredRuntime_ReturnsInitErrorRuntimeWhenEinoInitFails(t *testing.T) {
	rt := newPreferredRuntime(&Agent{})
	if _, ok := rt.(*runtimeInitError); !ok {
		t.Fatalf("newPreferredRuntime() type = %T, want *runtimeInitError", rt)
	}

	_, err := rt.RunLLMWithStream(context.Background(), "prompt", nil)
	if err == nil {
		t.Fatalf("RunLLMWithStream() error = nil, want init error")
	}
}

func TestNewPreferredRuntime_ReturnsInitErrorRuntimeForUnsupportedService(t *testing.T) {
	rt := newPreferredRuntime(&Agent{serviceName: "unsupported-provider"})
	if _, ok := rt.(*runtimeInitError); !ok {
		t.Fatalf("newPreferredRuntime() type = %T, want *runtimeInitError", rt)
	}

	_, err := rt.RunLLMWithStream(context.Background(), "prompt", nil)
	if err == nil {
		t.Fatalf("RunLLMWithStream() error = nil, want init error")
	}
}

func TestNewEinoRuntime_UnsupportedServiceError(t *testing.T) {
	_, err := newEinoRuntime(&Agent{serviceName: "unsupported-provider"})
	if err == nil {
		t.Fatalf("newEinoRuntime() error = nil, want unsupported service error")
	}
	if !strings.Contains(err.Error(), "not yet supported") {
		t.Fatalf("newEinoRuntime() error = %q, want contains %q", err.Error(), "not yet supported")
	}
}

func TestNewEinoRuntime_AnthropicRequiresModel(t *testing.T) {
	t.Setenv("ANTHROPIC_MODEL", "")

	_, err := newEinoRuntime(&Agent{serviceName: LLM_ANTHROPIC, modelName: ""})
	if err == nil {
		t.Fatalf("newEinoRuntime() error = nil, want anthropic model requirement error")
	}
	if !strings.Contains(err.Error(), "no model specified") {
		t.Fatalf("newEinoRuntime() error = %q, want contains %q", err.Error(), "no model specified")
	}
}

func TestToolsConfig_OnlyEnabledToolsAreWired(t *testing.T) {
	rt := &einoRuntime{agent: &Agent{_tools: []aitypes.Tool{
		&mcpTool{
			server:      "srv",
			name:        "enabled",
			description: "enabled mcp tool",
			schema:      []byte(`{"type":"object","properties":{"q":{"type":"string"}},"required":["q"]}`),
			enabled:     true,
		},
		&mcpTool{
			server:      "srv",
			name:        "disabled",
			description: "disabled mcp tool",
			schema:      []byte(`{"type":"object"}`),
			enabled:     false,
		},
		&fakeAgentTool{enabled: true},
	}}}

	cfg, err := rt.toolsConfig()
	if err != nil {
		t.Fatalf("toolsConfig() error = %v", err)
	}
	if len(cfg.Tools) != 2 {
		t.Fatalf("toolsConfig() tool count = %d, want 2", len(cfg.Tools))
	}

	info, err := cfg.Tools[0].Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if info.Name != "mcp_srv_enabled" {
		t.Fatalf("tool name = %q, want %q", info.Name, "mcp_srv_enabled")
	}
	if info.ParamsOneOf == nil {
		t.Fatalf("tool ParamsOneOf is nil, want schema params")
	}

	info2, err := cfg.Tools[1].Info(context.Background())
	if err != nil {
		t.Fatalf("Info(1) error = %v", err)
	}
	if info2.Name != "fake_tool" {
		t.Fatalf("second tool name = %q, want %q", info2.Name, "fake_tool")
	}
	if info2.ParamsOneOf == nil {
		t.Fatalf("second tool ParamsOneOf is nil, want non-nil schema")
	}
}

func TestToolsConfig_DuplicateSanitizedNamesAreMadeUnique(t *testing.T) {
	rt := &einoRuntime{agent: &Agent{_tools: []aitypes.Tool{
		&mcpTool{server: "atlassian", name: "search", description: "one", schema: []byte(`{"type":"object"}`), enabled: true},
		&mcpTool{server: "atlassian", name: "search", description: "two", schema: []byte(`{"type":"object"}`), enabled: true},
	}}}

	cfg, err := rt.toolsConfig()
	if err != nil {
		t.Fatalf("toolsConfig() error = %v", err)
	}
	if len(cfg.Tools) != 2 {
		t.Fatalf("toolsConfig() tool count = %d, want 2", len(cfg.Tools))
	}

	info0, err := cfg.Tools[0].Info(context.Background())
	if err != nil {
		t.Fatalf("Info(0) error = %v", err)
	}
	info1, err := cfg.Tools[1].Info(context.Background())
	if err != nil {
		t.Fatalf("Info(1) error = %v", err)
	}

	if info0.Name != "mcp_atlassian_search" {
		t.Fatalf("first tool name = %q, want %q", info0.Name, "mcp_atlassian_search")
	}
	if info1.Name != "mcp_atlassian_search_2" {
		t.Fatalf("second tool name = %q, want %q", info1.Name, "mcp_atlassian_search_2")
	}
}

func TestToolsConfig_InvalidSchemaReturnsError(t *testing.T) {
	rt := &einoRuntime{agent: &Agent{_tools: []aitypes.Tool{
		&mcpTool{
			server:      "srv",
			name:        "broken",
			description: "broken mcp tool",
			schema:      []byte(`{"type":"object",`),
			enabled:     true,
		},
	}}}

	_, err := rt.toolsConfig()
	if err == nil {
		t.Fatalf("toolsConfig() error = nil, want invalid schema error")
	}
	if !strings.Contains(err.Error(), "invalid schema") {
		t.Fatalf("toolsConfig() error = %q, want contains %q", err.Error(), "invalid schema")
	}
}

func TestSanitizeToolName_AllowsOnlyProviderSafeCharacters(t *testing.T) {
	got := sanitizeToolName("mcp.atlassian.atlassianUserInfo")
	if got != "mcp_atlassian_atlassianUserInfo" {
		t.Fatalf("sanitizeToolName() = %q, want %q", got, "mcp_atlassian_atlassianUserInfo")
	}
}

func TestWithAIStreamCallback_EmitsToolProgress(t *testing.T) {
	var chunks []string
	ctx := withAIStreamCallback(context.Background(), func(s string) {
		chunks = append(chunks, s)
	})

	emitAIStreamToolProgress(ctx, "Action: mcp_atlassian_search\n")
	emitAIStreamToolProgress(ctx, "Action Input: {\"query\":\"abc\"}\n")

	if len(chunks) != 2 {
		t.Fatalf("emitted chunks = %d, want 2", len(chunks))
	}
	if chunks[0] != "Action: mcp_atlassian_search\n" {
		t.Fatalf("chunk[0] = %q", chunks[0])
	}
	if chunks[1] != "Action Input: {\"query\":\"abc\"}\n" {
		t.Fatalf("chunk[1] = %q", chunks[1])
	}
}

func TestEmitAIStreamToolProgress_NoCallbackNoPanic(t *testing.T) {
	emitAIStreamToolProgress(context.Background(), "Action: x\n")
	emitAIStreamToolProgress(nil, "Action: x\n")
	emitAIStreamToolProgress(context.Background(), "")
}

func TestUnwrapToolInput_WrappedObject(t *testing.T) {
	got := unwrapToolInput(`{"input":"hello"}`)
	if got != "hello" {
		t.Fatalf("unwrapToolInput() = %q, want %q", got, "hello")
	}
}

func TestUnwrapToolInput_JSONString(t *testing.T) {
	got := unwrapToolInput(`"hello"`)
	if got != "hello" {
		t.Fatalf("unwrapToolInput() = %q, want %q", got, "hello")
	}
}

func TestUnwrapToolInput_FallbackRaw(t *testing.T) {
	raw := `not-json`
	got := unwrapToolInput(raw)
	if got != raw {
		t.Fatalf("unwrapToolInput() = %q, want %q", got, raw)
	}
}

func TestEinoAgentTool_InvokableRun_UnwrapsInputField(t *testing.T) {
	f := &fakeAgentTool{enabled: true}
	et, err := newEinoAgentTool(f)
	if err != nil {
		t.Fatalf("newEinoAgentTool() error = %v", err)
	}

	_, err = et.InvokableRun(context.Background(), `{"input":"abc"}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v", err)
	}
	if f.input != "abc" {
		t.Fatalf("tool input = %q, want %q", f.input, "abc")
	}
}

func TestNewPreferredRuntime_OllamaUsesEinoRuntime(t *testing.T) {
	rt := newPreferredRuntime(&Agent{serviceName: LLM_OLLAMA, modelName: "llama3"})
	if _, ok := rt.(*einoRuntime); !ok {
		t.Fatalf("newPreferredRuntime() type = %T, want *einoRuntime", rt)
	}
}

func TestStreamToolCallCheckerAllChunks_DetectsToolCall(t *testing.T) {
	idx := 0
	msg := &schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			Index: &idx,
			ID:    "tool-1",
			Type:  "function",
			Function: schema.FunctionCall{
				Name:      "search",
				Arguments: `{"query":"foo"}`,
			},
		}},
	}

	stream := schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("thinking", nil), msg})
	got, err := streamToolCallCheckerAllChunks(context.Background(), stream)
	if err != nil {
		t.Fatalf("streamToolCallCheckerAllChunks() error = %v", err)
	}
	if !got {
		t.Fatalf("streamToolCallCheckerAllChunks() = %v, want true", got)
	}
}

func TestStreamToolCallCheckerAllChunks_ReturnsFalseOnEOF(t *testing.T) {
	stream := schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("no tools", nil)})
	got, err := streamToolCallCheckerAllChunks(context.Background(), stream)
	if err != nil {
		t.Fatalf("streamToolCallCheckerAllChunks() error = %v", err)
	}
	if got {
		t.Fatalf("streamToolCallCheckerAllChunks() = %v, want false", got)
	}
}

func TestStreamToolCallCheckerAllChunks_PropagatesRecvError(t *testing.T) {
	errExpected := errors.New("recv failed")
	stream, writer := schema.Pipe[*schema.Message](1)
	go func() {
		defer writer.Close()
		writer.Send(nil, errExpected)
	}()

	got, err := streamToolCallCheckerAllChunks(context.Background(), stream)
	if got {
		t.Fatalf("streamToolCallCheckerAllChunks() = %v, want false", got)
	}
	if !errors.Is(err, errExpected) {
		t.Fatalf("streamToolCallCheckerAllChunks() error = %v, want %v", err, errExpected)
	}
}

func TestAgentRunLLMWithStream_InvokesCancelThenRuntime(t *testing.T) {
	rt := &fakeRuntime{result: "done"}
	canceled := false
	chunks := ""

	agent := &Agent{
		runtime: rt,
		fnCancel: func() {
			canceled = true
		},
	}

	result, err := agent.RunLLMWithStream(context.Background(), "prompt", func(chunk string) {
		chunks += chunk
	})
	if err != nil {
		t.Fatalf("RunLLMWithStream() error = %v", err)
	}
	if !canceled {
		t.Fatalf("fnCancel was not called")
	}
	if agent.fnCancel != nil {
		t.Fatalf("agent.fnCancel was not cleared")
	}
	if !rt.called {
		t.Fatalf("runtime was not called")
	}
	if rt.prompt != "prompt" {
		t.Fatalf("runtime prompt = %q, want %q", rt.prompt, "prompt")
	}
	if chunks != "chunk" {
		t.Fatalf("streamed chunks = %q, want %q", chunks, "chunk")
	}
	if result != "done" {
		t.Fatalf("result = %q, want %q", result, "done")
	}
}

func TestAgentRunLLMWithStream_CallbackOrderAndAccumulation(t *testing.T) {
	canceled := false
	fnCancelSeen := false
	gotChunks := ""

	rt := &fakeRuntime{
		result: "AB",
		chunks: []string{"A", "B"},
	}

	agent := &Agent{
		runtime: rt,
		fnCancel: func() {
			canceled = true
		},
	}

	result, err := agent.RunLLMWithStream(context.Background(), "prompt", func(chunk string) {
		if !canceled || agent.fnCancel != nil {
			fnCancelSeen = true
		}
		gotChunks += chunk
	})
	if err != nil {
		t.Fatalf("RunLLMWithStream() error = %v", err)
	}
	if fnCancelSeen {
		t.Fatalf("stream callback observed cancellation state before fnCancel lifecycle completed")
	}
	if gotChunks != "AB" {
		t.Fatalf("stream callback chunks = %q, want %q", gotChunks, "AB")
	}
	if result != "AB" {
		t.Fatalf("RunLLMWithStream() result = %q, want %q", result, "AB")
	}
}

func TestAgentRunLLMWithStream_PropagatesRuntimeError(t *testing.T) {
	errExpected := errors.New("runtime failed")
	rt := &fakeRuntime{err: errExpected}

	agent := &Agent{runtime: rt}
	_, err := agent.RunLLMWithStream(context.Background(), "prompt", nil)
	if !errors.Is(err, errExpected) {
		t.Fatalf("RunLLMWithStream() error = %v, want %v", err, errExpected)
	}
}

func TestAgentRunLLMWithStream_PreservesPartialResultOnError(t *testing.T) {
	errExpected := errors.New("stream interrupted")
	rt := &fakeRuntime{
		result: "partial-response",
		err:    errExpected,
		chunks: []string{"partial-", "response"},
	}

	agent := &Agent{runtime: rt}
	result, err := agent.RunLLMWithStream(context.Background(), "prompt", nil)
	if !errors.Is(err, errExpected) {
		t.Fatalf("RunLLMWithStream() error = %v, want %v", err, errExpected)
	}
	if result != "partial-response" {
		t.Fatalf("RunLLMWithStream() result = %q, want %q", result, "partial-response")
	}
}
