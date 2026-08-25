package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
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

	called   bool
	ctx      context.Context
	prompt   string
	messages []*schema.Message

	resetCalled bool
}

func (f *fakeRuntime) RunLLMWithMessageStream(ctx context.Context, messages []*schema.Message, streamCallback func(string)) (string, error) {
	f.called = true
	f.ctx = ctx
	f.messages = messages
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

func (f *fakeRuntime) RunLLMWithStream(ctx context.Context, prompt string, streamCallback func(string)) (string, error) {
	f.prompt = prompt
	return f.RunLLMWithMessageStream(ctx, []*schema.Message{schema.UserMessage(prompt)}, streamCallback)
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
	emitter := &aiStreamEmitter{fn: func(s string) {
		chunks = append(chunks, s)
	}}
	ctx := withAIStreamCallback(context.Background(), emitter)

	emitAIStreamToolProgress(ctx, "\n## Action\n\nmcp_atlassian_search\n\n")
	emitAIStreamToolProgress(ctx, "\n## Action Input\n\n```\n{\"query\":\"abc\"}\n```\n\n")

	if len(chunks) != 2 {
		t.Fatalf("emitted chunks = %d, want 2", len(chunks))
	}
	wantChunk0 := "\n## Action\n\nmcp_atlassian_search\n\n"
	if chunks[0] != wantChunk0 {
		t.Fatalf("chunk[0] = %q, want %q", chunks[0], wantChunk0)
	}
	wantChunk1 := "\n## Action Input\n\n```\n{\"query\":\"abc\"}\n```\n\n"
	if chunks[1] != wantChunk1 {
		t.Fatalf("chunk[1] = %q, want %q", chunks[1], wantChunk1)
	}
}

func TestAIStreamEmitter_WrapsReasoningAsBlockquote(t *testing.T) {
	var chunks []string
	emitter := &aiStreamEmitter{fn: func(s string) { chunks = append(chunks, s) }}

	emitter.emitReasoning("first thought")
	emitter.emitReasoning("\nsecond thought")
	emitter.emitText("Final answer.")

	got := strings.Join(chunks, "")
	want := "\n> **Thinking:** first thought\n> second thought\n\nFinal answer."
	if got != want {
		t.Fatalf("emitted stream = %q, want %q", got, want)
	}
}

func TestAIStreamEmitter_ReopensBlockquoteAfterText(t *testing.T) {
	var chunks []string
	emitter := &aiStreamEmitter{fn: func(s string) { chunks = append(chunks, s) }}

	emitter.emitReasoning("thinking A")
	emitter.emitText("Action: foo\n")
	emitter.emitReasoning("thinking B")
	emitter.emitText("Final.")

	got := strings.Join(chunks, "")
	want := "\n> **Thinking:** thinking A\n\nAction: foo\n\n> **Thinking:** thinking B\n\nFinal."
	if got != want {
		t.Fatalf("emitted stream = %q, want %q", got, want)
	}
}

func TestEmitAIStreamToolProgress_NoCallbackNoPanic(t *testing.T) {
	emitAIStreamToolProgress(context.Background(), "\n## Action\n\nx\n\n")
	emitAIStreamToolProgress(nil, "\n## Action\n\nx\n\n")
	emitAIStreamToolProgress(context.Background(), "")
}

func TestFormatToolCallMarkdown_UsesTildeFences(t *testing.T) {
	got := formatToolCallMarkdown("mcp_atlassian_search", `{"query":"abc"}`)
	want := "\n\n**Tool call:** `mcp_atlassian_search`\n\n~~~~json\n{\"query\":\"abc\"}\n~~~~\n\n"
	if got != want {
		t.Fatalf("formatToolCallMarkdown = %q, want %q", got, want)
	}
}

func TestFormatToolOutputMarkdown_UsesTildeFences(t *testing.T) {
	got := formatToolOutputMarkdown("some text with ``` inside")
	want := "**Tool output:**\n\n~~~~\nsome text with ``` inside\n~~~~\n\n"
	if got != want {
		t.Fatalf("formatToolOutputMarkdown = %q, want %q", got, want)
	}
}

func TestBuildEinoConversationMessages_AppendsHistoryThenPrompt(t *testing.T) {
	history := []sessiondb.Entry{
		{Prompt: "first question", LLMResponse: "first answer"},
		{Prompt: "second question", LLMResponse: "second answer"},
	}

	msgs := buildEinoConversationMessages(history, []*schema.Message{schema.UserMessage("current prompt")})
	if len(msgs) != 5 {
		t.Fatalf("message count = %d, want 5", len(msgs))
	}

	if msgs[0].Role != schema.User || msgs[0].Content != "first question" {
		t.Fatalf("msg[0] = (%s, %q), want (user, %q)", msgs[0].Role, msgs[0].Content, "first question")
	}
	if msgs[1].Role != schema.Assistant || msgs[1].Content != "first answer" {
		t.Fatalf("msg[1] = (%s, %q), want (assistant, %q)", msgs[1].Role, msgs[1].Content, "first answer")
	}
	if msgs[2].Role != schema.User || msgs[2].Content != "second question" {
		t.Fatalf("msg[2] = (%s, %q), want (user, %q)", msgs[2].Role, msgs[2].Content, "second question")
	}
	if msgs[3].Role != schema.Assistant || msgs[3].Content != "second answer" {
		t.Fatalf("msg[3] = (%s, %q), want (assistant, %q)", msgs[3].Role, msgs[3].Content, "second answer")
	}
	if msgs[4].Role != schema.User || msgs[4].Content != "current prompt" {
		t.Fatalf("msg[4] = (%s, %q), want (user, %q)", msgs[4].Role, msgs[4].Content, "current prompt")
	}
}

func TestBuildEinoConversationMessagesMovesSystemPromptBeforeHistory(t *testing.T) {
	history := []sessiondb.Entry{
		{Prompt: "previous question", LLMResponse: "previous answer"},
	}

	msgs := buildEinoConversationMessages(history, []*schema.Message{
		schema.SystemMessage("current system prompt"),
		schema.UserMessage("current prompt"),
	})
	if len(msgs) != 4 {
		t.Fatalf("message count = %d, want 4", len(msgs))
	}
	wantRoles := []string{string(schema.System), string(schema.User), string(schema.Assistant), string(schema.User)}
	for i, wantRole := range wantRoles {
		if string(msgs[i].Role) != wantRole {
			t.Fatalf("msg[%d].Role = %s, want %s", i, msgs[i].Role, wantRole)
		}
	}
	if msgs[0].Content != "current system prompt" || msgs[3].Content != "current prompt" {
		t.Fatalf("messages = %#v, want system prompt first and current prompt last", msgs)
	}
}

func TestBuildEinoConversationMessages_TrimsToMaxHistoryTurns(t *testing.T) {
	history := make([]sessiondb.Entry, 0, einoMaxHistoryTurns+3)
	for i := 0; i < einoMaxHistoryTurns+3; i++ {
		history = append(history, sessiondb.Entry{
			Prompt:      fmt.Sprintf("question %d", i),
			LLMResponse: fmt.Sprintf("answer %d", i),
		})
	}

	msgs := buildEinoConversationMessages(history, []*schema.Message{schema.UserMessage("now")})
	expected := (einoMaxHistoryTurns * 2) + 1
	if len(msgs) != expected {
		t.Fatalf("message count = %d, want %d", len(msgs), expected)
	}

	if msgs[0].Content != "question 3" {
		t.Fatalf("oldest retained question = %q, want %q", msgs[0].Content, "question 3")
	}
	if msgs[1].Content != "answer 3" {
		t.Fatalf("oldest retained answer = %q, want %q", msgs[1].Content, "answer 3")
	}
	if msgs[len(msgs)-1].Content != "now" {
		t.Fatalf("latest prompt = %q, want %q", msgs[len(msgs)-1].Content, "now")
	}
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
	et, err := newEinoAgentTool(nil, f)
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
	if len(rt.messages) != 1 || rt.messages[0].Content != "prompt" {
		t.Fatalf("runtime messages = %#v, want single user prompt", rt.messages)
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

func TestAgentRunLLMWithMessageStream_UsesStructuredMessages(t *testing.T) {
	rt := &fakeRuntime{result: "done"}
	agent := &Agent{runtime: rt}
	messages := []*schema.Message{
		schema.SystemMessage("system"),
		schema.UserMessage("question"),
	}

	result, err := agent.RunLLMWithMessageStream(context.Background(), messages, nil)
	if err != nil {
		t.Fatalf("RunLLMWithMessageStream() error = %v", err)
	}
	if result != "done" {
		t.Fatalf("RunLLMWithMessageStream() result = %q, want %q", result, "done")
	}
	if len(rt.messages) != 2 {
		t.Fatalf("runtime messages len = %d, want 2", len(rt.messages))
	}
	if rt.messages[0].Role != schema.System || rt.messages[1].Role != schema.User {
		t.Fatalf("runtime messages roles = (%s, %s), want (system, user)", rt.messages[0].Role, rt.messages[1].Role)
	}
}
