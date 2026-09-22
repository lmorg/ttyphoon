package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
	"github.com/lmorg/ttyphoon/config"
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
	got := sanitizeToolName("mcp.atlassian.atlassianUserInfo")
	if got != "mcp_atlassian_atlassianUserInfo" {
		t.Fatalf("sanitizeToolName() = %q, want %q", got, "mcp_atlassian_atlassianUserInfo")
	}
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

func TestEinoAgentToolRecordsContinuationObservation(t *testing.T) {
	delegate := &fakeAgentTool{enabled: true}
	tool := &einoAgentTool{
		runtime:  &einoRuntime{},
		delegate: delegate,
	}
	checkpoint := &continuationCheckpoint{}
	ctx := withContinuationCheckpoint(context.Background(), checkpoint)

	output, err := tool.InvokableRun(ctx, `{"path":"main.go"}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v", err)
	}
	if output != "" {
		t.Fatalf("InvokableRun() output = %q, want empty fake output", output)
	}

	snapshot := checkpoint.snapshot()
	if len(snapshot.toolObservations) != 1 {
		t.Fatalf("tool observations = %d, want 1", len(snapshot.toolObservations))
	}
	observation := snapshot.toolObservations[0]
	if observation.Tool != "fake.tool" || len(observation.Inputs) != 1 || observation.Inputs[0] != `{"path":"main.go"}` {
		t.Fatalf("observation = %+v, want fake.tool with original arguments", observation)
	}
}

func TestFormatContinuationCheckpointMessageIncludesSemanticSections(t *testing.T) {
	checkpoint := &continuationCheckpoint{}
	checkpoint.setVisibleOutput("visible tail")
	recordContinuationToolObservation(withContinuationCheckpoint(context.Background(), checkpoint), aitypes.ToolObservation{
		Tool:              "grep",
		Status:            "ok",
		FilesRead:         []string{"a.go"},
		FilesModified:     []string{"b.go"},
		DirectoriesListed: []string{"/tmp/project"},
		Counts:            map[string]int{"matches": 2},
		SearchesRun:       []aitypes.SearchObservation{{Query: "needle", FileFilter: "*.go", ResultCount: 2, TopPaths: []string{"a.go", "b.go"}}},
	})

	message := formatContinuationCheckpointMessage(checkpoint)
	for _, want := range []string{"Tool observations", "Files read: a.go", "Files modified: b.go", "Directories listed: /tmp/project", "Counts: matches=2", "Search: needle", "visible tail"} {
		if !strings.Contains(message, want) {
			t.Fatalf("checkpoint message missing %q:\n%s", want, message)
		}
	}
}

func TestFormatContinuationCheckpointMessageCapsTotalSize(t *testing.T) {
	checkpoint := &continuationCheckpoint{}
	ctx := withContinuationCheckpoint(context.Background(), checkpoint)
	for i := 0; i < 10; i++ {
		recordContinuationToolObservation(ctx, aitypes.ToolObservation{Tool: "readFiles", Outputs: []string{strings.Repeat("x", continuationCheckpointMaxFieldChars)}})
	}

	message := formatContinuationCheckpointMessage(checkpoint)
	if len(message) > continuationCheckpointMaxTotalChars+len("\n\n[continuation checkpoint truncated]") {
		t.Fatalf("message length = %d, want capped", len(message))
	}
	if !strings.Contains(message, "[continuation checkpoint truncated]") {
		t.Fatalf("checkpoint message missing truncation marker")
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

func TestUnwrapToolInput_PlainJSONObject(t *testing.T) {
	got := unwrapToolInput(`{"prompt":"hello","size":"1024x1024"}`)
	if got != `{"prompt":"hello","size":"1024x1024"}` {
		t.Fatalf("unwrapToolInput() = %q, want the plain object preserved as JSON", got)
	}
}

func TestUnwrapToolInput_WrappedJSONObject(t *testing.T) {
	got := unwrapToolInput(`{"input":{"prompt":"hello","size":"1024x1024"}}`)
	if got != `{"prompt":"hello","size":"1024x1024"}` {
		t.Fatalf("unwrapToolInput() = %q, want the wrapped object encoded as JSON", got)
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

func TestEinoAgentTool_InvokableRun_RejectsDisabledTool(t *testing.T) {
	tool := &fakeAgentTool{enabled: true}
	agent := &Agent{toolStates: map[string]string{tool.Name(): ToolStateDisabled}}
	einoTool := &einoAgentTool{runtime: &einoRuntime{agent: agent}, delegate: tool}

	out, err := einoTool.InvokableRun(context.Background(), `{"input":"hello"}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v, want refusal reported to the LLM", err)
	}
	if !strings.Contains(out, "disabled") {
		t.Fatalf("InvokableRun() output = %q, want disabled refusal text", out)
	}
	if tool.input != "" {
		t.Fatalf("disabled tool received input %q", tool.input)
	}
}

func TestEinoAgentTool_InvokableRun_DeniedToolDoesNotAbortRun(t *testing.T) {
	tool := &fakeAgentTool{enabled: true}
	agent := &Agent{toolStates: map[string]string{tool.Name(): ToolStateDenied}}
	einoTool := &einoAgentTool{runtime: &einoRuntime{agent: agent}, delegate: tool}

	out, err := einoTool.InvokableRun(context.Background(), `{"input":"hello"}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v, want nil so the agent run continues", err)
	}
	if !strings.Contains(out, "refused") {
		t.Fatalf("InvokableRun() output = %q, want refusal text", out)
	}
	if tool.input != "" {
		t.Fatalf("denied tool received input %q", tool.input)
	}
}

func TestEinoAgentTool_InvokableRun_CancellationStaysTerminal(t *testing.T) {
	tool := &fakeAgentTool{enabled: true}
	agent := &Agent{toolStates: map[string]string{tool.Name(): ToolStateApproval}}
	einoTool := &einoAgentTool{runtime: &einoRuntime{agent: agent}, delegate: tool}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := einoTool.InvokableRun(ctx, `{"input":"hello"}`); !errors.Is(err, context.Canceled) {
		t.Fatalf("InvokableRun() error = %v, want context.Canceled", err)
	}
}

func TestRequestToolPermission_DecisionScopes(t *testing.T) {
	tests := []struct {
		decision    string
		wantAllowed bool
		wantStored  bool
	}{
		{decision: "allow-once", wantAllowed: true, wantStored: false},
		{decision: "allow-prompt", wantAllowed: true, wantStored: true},
		{decision: "deny-once", wantAllowed: false, wantStored: false},
		{decision: "deny-prompt", wantAllowed: false, wantStored: true},
	}

	for _, test := range tests {
		t.Run(test.decision, func(t *testing.T) {
			agent := &Agent{toolStates: map[string]string{"fake.tool": ToolStateApproval}}

			resolved := make(chan error, 1)
			go func() {
				resolved <- agent.RequestToolPermission(context.Background(), "fake.tool")
			}()

			var reqID string
			for range 200 {
				writePermissionRequests.mu.Lock()
				for id := range writePermissionRequests.m {
					reqID = id
				}
				writePermissionRequests.mu.Unlock()
				if reqID != "" {
					break
				}
				time.Sleep(time.Millisecond)
			}
			if reqID == "" {
				t.Fatal("no pending permission request was raised")
			}

			if err := ResolveWritePermissionRequest(reqID, test.decision); err != nil {
				t.Fatalf("ResolveWritePermissionRequest() error = %v", err)
			}

			err := <-resolved
			if test.wantAllowed && err != nil {
				t.Fatalf("RequestToolPermission() error = %v, want nil", err)
			}
			if !test.wantAllowed {
				if !errors.Is(err, ErrToolPermissionRefused) {
					t.Fatalf("RequestToolPermission() error = %v, want ErrToolPermissionRefused", err)
				}
			}

			agent.toolPermissionMu.Lock()
			stored := agent.toolPermissions["fake.tool"].decision != toolPermissionUndecided
			agent.toolPermissionMu.Unlock()
			if stored != test.wantStored {
				t.Fatalf("decision stored = %v, want %v", stored, test.wantStored)
			}
		})
	}
}

func TestResetToolPermissions_ClearsPromptScopedDecisions(t *testing.T) {
	agent := &Agent{toolPermissions: map[string]*toolPermissionState{
		"a": {decision: toolPermissionAllowedPrompt},
		"b": {decision: toolPermissionDeniedPrompt},
	}}

	agent.ResetToolPermissions()

	if len(agent.toolPermissions) != 0 {
		t.Fatalf("toolPermissions = %v, want empty after reset", agent.toolPermissions)
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

func TestEinoRuntimeContinuationAppendsPreviousWindowOutput(t *testing.T) {
	restore := setMaxContinuationsForTest(t, 3)
	defer restore()

	var calls int
	var secondCallMessages []*schema.Message
	runtime := &einoRuntime{agent: &Agent{}}
	runtime.boundedWindowRunner = func(ctx context.Context, messages []*schema.Message, _ func(string)) (string, error) {
		calls++
		if calls == 1 {
			recordContinuationToolObservation(ctx, aitypes.ToolObservation{Tool: "readFiles", Inputs: []string{`{"files":["main.go"]}`}, Outputs: []string{"main.go contents"}})
			return "partial visible output", errors.New("[GraphRunError] exceeds max steps")
		}
		secondCallMessages = append([]*schema.Message(nil), messages...)
		return "done", nil
	}

	var streamed strings.Builder
	result, err := runtime.RunLLMWithMessageStream(context.Background(), []*schema.Message{schema.UserMessage("original task")}, func(chunk string) {
		streamed.WriteString(chunk)
	})
	if err != nil {
		t.Fatalf("RunLLMWithMessageStream() error = %v", err)
	}
	if result != "partial visible outputdone" {
		t.Fatalf("result = %q, want concatenated continuation output", result)
	}
	if calls != 2 {
		t.Fatalf("bounded window calls = %d, want 2", calls)
	}
	if len(secondCallMessages) != 2 {
		t.Fatalf("second call messages = %d, want original + continuation", len(secondCallMessages))
	}
	last := secondCallMessages[len(secondCallMessages)-1]
	if last.Role != schema.User {
		t.Fatalf("continuation message role = %s, want user", last.Role)
	}
	if !strings.Contains(last.Content, "partial visible output") || !strings.Contains(last.Content, "Continue the original task") {
		t.Fatalf("continuation message content = %q, want previous output and continuation instruction", last.Content)
	}
	if !strings.Contains(last.Content, "Continuation checkpoint") {
		t.Fatalf("continuation message content = %q, want structured checkpoint section", last.Content)
	}
	if !strings.Contains(last.Content, "Tool observations") || !strings.Contains(last.Content, "readFiles") || !strings.Contains(last.Content, "main.go contents") {
		t.Fatalf("continuation message content = %q, want captured tool observation", last.Content)
	}
	if !strings.Contains(streamed.String(), "Continuing after max steps (1/3)") {
		t.Fatalf("streamed output = %q, want continuation progress marker", streamed.String())
	}
}

func TestIsTransientStreamError(t *testing.T) {
	for _, err := range []error{
		errors.New("failed to receive stream chunk: read tcp: connection reset by peer"),
		errors.New("unexpected EOF while reading model stream"),
	} {
		if !isTransientStreamError(err) {
			t.Fatalf("isTransientStreamError(%q) = false, want true", err)
		}
	}
	if isTransientStreamError(errors.New("tool permission denied")) {
		t.Fatal("isTransientStreamError() = true for a non-transport error")
	}
}

func TestEinoRuntimeRetriesTransientStreamWithCheckpoint(t *testing.T) {
	var calls int
	var retryMessages []*schema.Message
	runtime := &einoRuntime{agent: &Agent{}}
	runtime.boundedWindowRunner = func(_ context.Context, messages []*schema.Message, _ func(string)) (string, error) {
		calls++
		if calls == 1 {
			return "partial", errors.New("failed to receive stream chunk: connection reset by peer")
		}
		retryMessages = append([]*schema.Message(nil), messages...)
		return "recovered", nil
	}

	var streamed strings.Builder
	result, err := runtime.RunLLMWithMessageStream(context.Background(), []*schema.Message{schema.UserMessage("original task")}, func(chunk string) {
		streamed.WriteString(chunk)
	})
	if err != nil {
		t.Fatalf("RunLLMWithMessageStream() error = %v", err)
	}
	if result != "partialrecovered" {
		t.Fatalf("result = %q, want partialrecovered", result)
	}
	if calls != 2 {
		t.Fatalf("bounded window calls = %d, want 2", calls)
	}
	if len(retryMessages) != 2 || !strings.Contains(retryMessages[1].Content, "Continuation checkpoint") {
		t.Fatalf("retry messages = %+v, want checkpoint continuation", retryMessages)
	}
	if !strings.Contains(streamed.String(), "Retrying interrupted model stream (1/2)") {
		t.Fatalf("streamed output = %q, want retry progress marker", streamed.String())
	}
}

func TestEinoRuntimeContinuationAsksUserAtConfiguredBoundary(t *testing.T) {
	restore := setMaxContinuationsForTest(t, 1)
	defer restore()

	var calls int
	runtime := &einoRuntime{agent: &Agent{}}
	runtime.boundedWindowRunner = func(_ context.Context, _ []*schema.Message, _ func(string)) (string, error) {
		calls++
		return "partial", errors.New("[GraphRunError] exceeds max steps")
	}

	var streamed strings.Builder
	result, err := runtime.RunLLMWithMessageStream(context.Background(), []*schema.Message{schema.UserMessage("original task")}, func(chunk string) {
		streamed.WriteString(chunk)
		if requestID := userQuestionRequestIDFromMarkdown(chunk); requestID != "" {
			if resolveErr := ResolveUserQuestionRequest(requestID, "finish up"); resolveErr != nil {
				t.Errorf("ResolveUserQuestionRequest() error = %v", resolveErr)
			}
		}
	})
	if err != nil {
		t.Fatalf("RunLLMWithMessageStream() error = %v", err)
	}
	if result != "partial" {
		t.Fatalf("result = %q, want partial output", result)
	}
	if calls != 1 {
		t.Fatalf("bounded window calls = %d, want 1", calls)
	}
	if !strings.Contains(streamed.String(), "maximum number of continuations (1)") {
		t.Fatalf("streamed output = %q, want user question", streamed.String())
	}
	if strings.Contains(streamed.String(), "Continuing after max steps") {
		t.Fatalf("streamed output = %q, should not continue after finish up", streamed.String())
	}
}

func TestEinoRuntimeContinuationContinueChoiceOpensAnotherWindow(t *testing.T) {
	restore := setMaxContinuationsForTest(t, 1)
	defer restore()

	var calls int
	var secondCallMessages []*schema.Message
	runtime := &einoRuntime{agent: &Agent{}}
	runtime.boundedWindowRunner = func(_ context.Context, messages []*schema.Message, _ func(string)) (string, error) {
		calls++
		if calls == 1 {
			return "first", errors.New("[GraphRunError] exceeds max steps")
		}
		secondCallMessages = append([]*schema.Message(nil), messages...)
		return "done", nil
	}

	var streamed strings.Builder
	result, err := runtime.RunLLMWithMessageStream(context.Background(), []*schema.Message{schema.UserMessage("original task")}, func(chunk string) {
		streamed.WriteString(chunk)
		if requestID := userQuestionRequestIDFromMarkdown(chunk); requestID != "" {
			if resolveErr := ResolveUserQuestionRequest(requestID, "continue"); resolveErr != nil {
				t.Errorf("ResolveUserQuestionRequest() error = %v", resolveErr)
			}
		}
	})
	if err != nil {
		t.Fatalf("RunLLMWithMessageStream() error = %v", err)
	}
	if result != "firstdone" {
		t.Fatalf("result = %q, want firstdone", result)
	}
	if calls != 2 {
		t.Fatalf("bounded window calls = %d, want 2", calls)
	}
	if len(secondCallMessages) != 2 {
		t.Fatalf("second call messages = %d, want original + continuation", len(secondCallMessages))
	}
	if !strings.Contains(streamed.String(), "Continuing after max steps (1/1)") {
		t.Fatalf("streamed output = %q, want reset continuation progress marker", streamed.String())
	}
}

func setMaxContinuationsForTest(t *testing.T, value int) func() {
	t.Helper()
	old := config.Config.Ai.MaxContinuations
	config.Config.Ai.MaxContinuations = value
	return func() { config.Config.Ai.MaxContinuations = old }
}

func userQuestionRequestIDFromMarkdown(markdown string) string {
	const marker = "ttyphoon://ai-user-question?request="
	idx := strings.Index(markdown, marker)
	if idx < 0 {
		return ""
	}
	rest := markdown[idx+len(marker):]
	if amp := strings.Index(rest, "&"); amp >= 0 {
		return rest[:amp]
	}
	return rest
}
