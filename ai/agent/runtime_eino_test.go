package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	einoModel "github.com/cloudwego/eino/components/model"
	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
	"github.com/lmorg/ttyphoon/config"
)

type fakeAgentTool struct {
	enabled bool
	input   string
}

type structuredToolTestInput struct {
	Query   string `json:"query"`
	JSON    any    `json:"json"`
	Options struct {
		FileFilter string `json:"fileFilter,omitempty"`
	} `json:"options,omitempty"`
}

type fakeStructuredAgentTool struct {
	*fakeAgentTool
	input         *structuredToolTestInput
	callError     error
	partialOutput string
	onCall        func()
}

type fakeDescribedStructuredAgentTool struct {
	*fakeStructuredAgentTool
	describedInput any
}

func (f *fakeDescribedStructuredAgentTool) CallDescription(_ context.Context, input any) string {
	f.describedInput = input
	return "### Invocation details\n\nThis describes the requested operation."
}

func (f *fakeStructuredAgentTool) InputType() reflect.Type {
	return reflect.TypeOf(structuredToolTestInput{})
}

func (f *fakeStructuredAgentTool) CallStructured(_ context.Context, input any) (string, error) {
	structured, ok := input.(*structuredToolTestInput)
	if !ok {
		return "", fmt.Errorf("unexpected structured input type %T", input)
	}
	f.input = structured
	if f.onCall != nil {
		f.onCall()
	}
	if f.callError != nil {
		return f.partialOutput, f.callError
	}
	return "structured result", nil
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

type retryAfterTestError struct {
	Response *http.Response
	err      error
}

func (e *retryAfterTestError) Error() string { return e.err.Error() }
func (e *retryAfterTestError) Unwrap() error { return e.err }

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
		&fakeStructuredAgentTool{fakeAgentTool: &fakeAgentTool{enabled: true}},
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

func TestStructuredToolUsesFieldSchemaAndDecodedInput(t *testing.T) {
	delegate := &fakeStructuredAgentTool{fakeAgentTool: &fakeAgentTool{enabled: true}}
	tool, err := newEinoAgentTool(nil, delegate)
	if err != nil {
		t.Fatalf("newEinoAgentTool: %v", err)
	}
	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	generatedSchema, err := info.ParamsOneOf.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema: %v", err)
	}
	schemaJSON, err := json.Marshal(generatedSchema)
	if err != nil {
		t.Fatalf("marshal tool schema: %v", err)
	}
	for _, field := range []string{`"query"`, `"options"`, `"fileFilter"`} {
		if !strings.Contains(string(schemaJSON), field) {
			t.Fatalf("tool schema %s does not contain field %s", schemaJSON, field)
		}
	}
	if strings.Contains(string(schemaJSON), `"input"`) {
		t.Fatalf("structured tool schema contains legacy input wrapper: %s", schemaJSON)
	}

	if _, err := tool.InvokableRun(context.Background(), `{"query":"needle","json":{"n":9007199254740993},"options":{"fileFilter":"*.go"}}`); err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if delegate.input == nil || delegate.input.Query != "needle" || delegate.input.Options.FileFilter != "*.go" {
		t.Fatalf("structured input = %+v, want decoded query and file filter", delegate.input)
	}
	jsonValue, ok := delegate.input.JSON.(map[string]any)
	if !ok || jsonValue["n"] != json.Number("9007199254740993") {
		t.Fatalf("structured arbitrary JSON = %#v, want exact integer preserved as json.Number", delegate.input.JSON)
	}
}

func TestStructuredToolCallDescriptionUsesDecodedCallInputForApproval(t *testing.T) {
	delegate := &fakeDescribedStructuredAgentTool{
		fakeStructuredAgentTool: &fakeStructuredAgentTool{fakeAgentTool: &fakeAgentTool{enabled: true}},
	}
	permissionAgent := &Agent{toolStates: map[string]string{delegate.Name(): ToolStateApproval}}
	tool, err := newEinoAgentTool(&einoRuntime{agent: permissionAgent}, delegate)
	if err != nil {
		t.Fatalf("newEinoAgentTool: %v", err)
	}

	resultCh := make(chan error, 1)
	go func() {
		_, callErr := tool.InvokableRun(context.Background(), `{"query":"needle","options":{"fileFilter":"*.go"}}`)
		resultCh <- callErr
	}()

	var requestID string
	for range 200 {
		writePermissionRequests.mu.Lock()
		for id := range writePermissionRequests.m {
			requestID = id
		}
		writePermissionRequests.mu.Unlock()
		if requestID != "" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if requestID == "" {
		t.Fatal("no pending permission request was raised")
	}

	if err := ResolveWritePermissionRequest(requestID, "allow-once", ""); err != nil {
		t.Fatalf("ResolveWritePermissionRequest: %v", err)
	}
	if err := <-resultCh; err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}

	input, ok := delegate.describedInput.(*structuredToolTestInput)
	if !ok {
		t.Fatalf("CallDescription input type = %T, want *structuredToolTestInput", delegate.describedInput)
	}
	if input != delegate.input {
		t.Fatalf("CallDescription input %p differs from CallStructured input %p", input, delegate.input)
	}
	if input.Query != "needle" || input.Options.FileFilter != "*.go" {
		t.Fatalf("CallDescription input = %+v, want decoded invocation arguments", input)
	}
}

func TestFormatToolPermissionRequestMarkdownIncludesDescriptionBeforeOptions(t *testing.T) {
	description := "### Invocation details\n\nThe requested operation."
	got := formatToolPermissionRequestMarkdown("fake.tool", "wp-test", description)
	descriptionIndex := strings.Index(got, description)
	allowIndex := strings.Index(got, "[Allow this invocation]")
	if descriptionIndex < 0 || allowIndex < 0 || descriptionIndex > allowIndex {
		t.Fatalf("permission Markdown = %q, want description before the approval options", got)
	}

	legacy := formatToolPermissionRequestMarkdown("fake.tool", "wp-test", "")
	wantLegacy := "\n\n**Access requested for tool fake.tool**\n\n" +
		"- [Allow this invocation](ttyphoon://ai-tool-permission?request=wp-test&decision=allow-once)\n" +
		"- [Allow for this prompt](ttyphoon://ai-tool-permission?request=wp-test&decision=allow-prompt)\n" +
		"- [Deny this invocation](ttyphoon://ai-tool-permission?request=wp-test&decision=deny-once)\n" +
		"- [Deny for this prompt](ttyphoon://ai-tool-permission?request=wp-test&decision=deny-prompt)\n\n"
	if legacy != wantLegacy {
		t.Fatalf("permission Markdown without a description = %q, want unchanged format %q", legacy, wantLegacy)
	}
}

func TestListToolsReportsGeneratedStructuredSchema(t *testing.T) {
	tool := &fakeStructuredAgentTool{fakeAgentTool: &fakeAgentTool{enabled: true}}
	owner := &Agent{
		_tools:     []aitypes.Tool{tool},
		toolStates: map[string]string{tool.Name(): ToolStateAlways},
	}
	listed := owner.ListTools()
	if len(listed) != 1 {
		t.Fatalf("ListTools() returned %d tools, want 1", len(listed))
	}
	schemaJSON, ok := listed[0]["schema"].(string)
	if !ok {
		t.Fatalf("schema type = %T, want string", listed[0]["schema"])
	}
	for _, field := range []string{`"query"`, `"options"`, `"fileFilter"`} {
		if !strings.Contains(schemaJSON, field) {
			t.Errorf("listed schema %s does not contain %s", schemaJSON, field)
		}
	}
	if strings.Contains(schemaJSON, `"input"`) {
		t.Fatalf("listed schema still contains legacy input wrapper: %s", schemaJSON)
	}
}

func TestStructuredToolRejectsLegacyInputWrapper(t *testing.T) {
	delegate := &fakeStructuredAgentTool{fakeAgentTool: &fakeAgentTool{enabled: true}}
	tool, err := newEinoAgentTool(nil, delegate)
	if err != nil {
		t.Fatalf("newEinoAgentTool: %v", err)
	}
	output, err := tool.InvokableRun(context.Background(), `{"input":"legacy"}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if !strings.Contains(output, `unknown field "input"`) {
		t.Fatalf("InvokableRun output = %q, want legacy wrapper rejected", output)
	}
	if delegate.input != nil {
		t.Fatalf("structured tool was invoked with %+v after invalid arguments", delegate.input)
	}
}

func TestStructuredToolInvocationErrorIsModelVisibleAndRetryable(t *testing.T) {
	delegate := &fakeStructuredAgentTool{
		fakeAgentTool: &fakeAgentTool{enabled: true},
		callError:     errors.New("invalid query: choose an existing file"),
		partialOutput: "one file was already processed",
	}
	tool, err := newEinoAgentTool(nil, delegate)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint := &continuationCheckpoint{}
	ctx := withContinuationCheckpoint(context.Background(), checkpoint)
	output, err := tool.InvokableRun(ctx, `{"query":"missing"}`)
	if err != nil {
		t.Fatalf("recoverable tool error aborted invocation: %v", err)
	}
	for _, text := range []string{"fake.tool", "invalid query", "corrected input", "another tool", "partially completed", delegate.partialOutput} {
		if !strings.Contains(output, text) {
			t.Fatalf("tool result %q does not include %q", output, text)
		}
	}
	observations := checkpoint.snapshot().toolObservations
	if len(observations) != 1 || observations[0].Status != "error" || observations[0].Error != delegate.callError.Error() {
		t.Fatalf("failed tool observation = %+v", observations)
	}
	delegate.callError = nil
	output, err = tool.InvokableRun(context.Background(), `{"query":"existing"}`)
	if err != nil || output != "structured result" || delegate.input.Query != "existing" {
		t.Fatalf("corrected retry = %q, %v, input %+v", output, err, delegate.input)
	}
}

func TestStructuredToolLocalTimeoutDoesNotCancelAgent(t *testing.T) {
	delegate := &fakeStructuredAgentTool{fakeAgentTool: &fakeAgentTool{enabled: true}, callError: context.DeadlineExceeded}
	tool, err := newEinoAgentTool(nil, delegate)
	if err != nil {
		t.Fatal(err)
	}
	output, err := tool.InvokableRun(context.Background(), `{"query":"slow"}`)
	if err != nil || !strings.Contains(output, context.DeadlineExceeded.Error()) {
		t.Fatalf("local timeout = %q, %v; want model-visible failure", output, err)
	}
}

func TestStructuredToolCancellationDuringInvocationStaysTerminal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	delegate := &fakeStructuredAgentTool{
		fakeAgentTool: &fakeAgentTool{enabled: true},
		onCall:        cancel,
	}
	tool, err := newEinoAgentTool(nil, delegate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tool.InvokableRun(ctx, `{"query":"file"}`); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled tool error = %v, want context.Canceled", err)
	}
}

func TestStructuredToolExpiredAgentDeadlineStaysTerminal(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	delegate := &fakeStructuredAgentTool{fakeAgentTool: &fakeAgentTool{enabled: true}}
	tool, err := newEinoAgentTool(nil, delegate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tool.InvokableRun(ctx, `{"query":"file"}`); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired agent deadline = %v, want context.DeadlineExceeded", err)
	}
	if delegate.input != nil {
		t.Fatal("tool was called after the agent deadline expired")
	}
}

type failingStringInputTool struct{ *fakeAgentTool }

func (tool *failingStringInputTool) Call(context.Context, string) (string, error) {
	return "partial response", errors.New("server rejected arguments")
}

func TestStringInputToolInvocationErrorDoesNotAbortRun(t *testing.T) {
	delegate := &failingStringInputTool{fakeAgentTool: &fakeAgentTool{enabled: true}}
	tool := &einoAgentTool{delegate: delegate, stringInputTool: delegate}
	output, err := tool.InvokableRun(context.Background(), `{}`)
	if err != nil || !strings.Contains(output, "server rejected arguments") || !strings.Contains(output, "partial response") {
		t.Fatalf("string adapter failure = %q, %v; want model-visible error", output, err)
	}
}

type toolRecoveryChatModel struct {
	generate func([]*schema.Message) (*schema.Message, error)
}

func (model *toolRecoveryChatModel) WithTools([]*schema.ToolInfo) (einoModel.ToolCallingChatModel, error) {
	return &toolRecoveryChatModel{generate: model.generate}, nil
}

func (model *toolRecoveryChatModel) Generate(_ context.Context, messages []*schema.Message, _ ...einoModel.Option) (*schema.Message, error) {
	return model.generate(messages)
}

func (model *toolRecoveryChatModel) Stream(ctx context.Context, messages []*schema.Message, options ...einoModel.Option) (*schema.StreamReader[*schema.Message], error) {
	message, err := model.Generate(ctx, messages, options...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.Message{message}), nil
}

func TestEinoReActModelReceivesToolFailureAndRetries(t *testing.T) {
	delegate := &fakeStructuredAgentTool{fakeAgentTool: &fakeAgentTool{enabled: true}, callError: errors.New("file does not exist")}
	tool, err := newEinoAgentTool(nil, delegate)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	model := &toolRecoveryChatModel{generate: func(messages []*schema.Message) (*schema.Message, error) {
		last := messages[len(messages)-1]
		arguments := `{"query":"missing"}`
		switch calls {
		case 0:
		case 1:
			if last.Role != schema.Tool || !strings.Contains(last.Content, "file does not exist") {
				return nil, fmt.Errorf("model did not receive tool failure: %+v", last)
			}
			delegate.callError = nil
			arguments = `{"query":"existing"}`
		case 2:
			if last.Role != schema.Tool || last.Content != "structured result" {
				return nil, fmt.Errorf("model did not receive retry result: %+v", last)
			}
			calls++
			return &schema.Message{Role: schema.Assistant, Content: "Recovered and finished"}, nil
		default:
			return nil, errors.New("unexpected model call")
		}
		calls++
		return &schema.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{
			ID: fmt.Sprintf("retry-%d", calls), Type: "function",
			Function: schema.FunctionCall{Name: tool.info.Name, Arguments: arguments},
		}}}, nil
	}}
	runner, err := react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: model, ToolsConfig: compose.ToolsNodeConfig{Tools: []einoTool.BaseTool{tool}}, MaxStep: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner.Generate(context.Background(), []*schema.Message{schema.UserMessage("Read the file")})
	if err != nil {
		t.Fatalf("tool failure ended the ReAct run: %v", err)
	}
	if result.Content != "Recovered and finished" || calls != 3 || delegate.input.Query != "existing" {
		t.Fatalf("result = %+v, model calls %d, tool input %+v", result, calls, delegate.input)
	}
}

func TestStructuredToolSchemaIsSelfContainedObject(t *testing.T) {
	tool, err := newEinoAgentTool(nil, &fakeStructuredAgentTool{fakeAgentTool: &fakeAgentTool{enabled: true}})
	if err != nil {
		t.Fatalf("newEinoAgentTool: %v", err)
	}
	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	generated, err := info.ParamsOneOf.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema: %v", err)
	}
	schemaJSON, err := json.Marshal(generated)
	if err != nil {
		t.Fatalf("marshal tool schema: %v", err)
	}
	if !strings.Contains(string(schemaJSON), `"type":"object"`) {
		t.Fatalf("tool schema is not a root object: %s", schemaJSON)
	}
	for _, unwanted := range []string{`"$ref"`, `"$defs"`, `"$schema"`, `"$id"`} {
		if strings.Contains(string(schemaJSON), unwanted) {
			t.Fatalf("tool schema contains %s indirection the provider cannot resolve: %s", unwanted, schemaJSON)
		}
	}
}

type stringifiedOptionsTestT struct {
	FileFilter string `json:"fileFilter,omitempty"`
}

type stringifiedRequestTestT struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

type stringifiedInputTestT struct {
	Files    []string                   `json:"files,omitempty"`
	Options  stringifiedOptionsTestT    `json:"options,omitempty"`
	Requests []*stringifiedRequestTestT `json:"requests,omitempty"`
	Note     string                     `json:"note,omitempty"`
	Payload  any                        `json:"payload,omitempty"`
}

func TestDecodeStructuredToolInputRepairsStringifiedJSON(t *testing.T) {
	tests := []struct {
		name   string
		args   string
		verify func(*testing.T, *stringifiedInputTestT)
	}{
		{
			name: "array encoded as string",
			args: `{"files":"[\"a.go\"]"}`,
			verify: func(t *testing.T, got *stringifiedInputTestT) {
				if len(got.Files) != 1 || got.Files[0] != "a.go" {
					t.Fatalf("files = %#v, want [a.go]", got.Files)
				}
			},
		},
		{
			name: "object encoded as string",
			args: `{"options":"{\"fileFilter\":\"*.go\"}"}`,
			verify: func(t *testing.T, got *stringifiedInputTestT) {
				if got.Options.FileFilter != "*.go" {
					t.Fatalf("options.fileFilter = %q, want *.go", got.Options.FileFilter)
				}
			},
		},
		{
			name: "array of objects encoded as string",
			args: `{"requests":"[{\"name\":\"d1\",\"prompt\":\"p\"}]"}`,
			verify: func(t *testing.T, got *stringifiedInputTestT) {
				if len(got.Requests) != 1 || got.Requests[0].Name != "d1" || got.Requests[0].Prompt != "p" {
					t.Fatalf("requests = %#v, want one decoded request", got.Requests)
				}
			},
		},
		{
			name: "string field keeps its literal JSON text",
			args: `{"note":"{\"not\":\"parsed\"}"}`,
			verify: func(t *testing.T, got *stringifiedInputTestT) {
				if got.Note != `{"not":"parsed"}` {
					t.Fatalf("note = %q, want the literal string preserved", got.Note)
				}
			},
		},
		{
			name: "arbitrary JSON field keeps a string value",
			args: `{"payload":"{\"a\":1}"}`,
			verify: func(t *testing.T, got *stringifiedInputTestT) {
				if got.Payload != `{"a":1}` {
					t.Fatalf("payload = %#v, want the string preserved", got.Payload)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got stringifiedInputTestT
			if err := decodeStructuredToolInput(test.args, &got); err != nil {
				t.Fatalf("decodeStructuredToolInput(%s): %v", test.args, err)
			}
			test.verify(t, &got)
		})
	}
}

func TestDecodeStructuredToolInputErrorsUseJSONVocabulary(t *testing.T) {
	tests := []struct {
		name string
		args string
		want []string
	}{
		{
			name: "type mismatch names the field and JSON type",
			args: `{"files":123}`,
			want: []string{`field "files"`, "array of strings", "number"},
		},
		{
			name: "unknown field lists the allowed fields",
			args: `{"input":"legacy"}`,
			want: []string{`unknown field "input"`, "allowed fields", "files"},
		},
		{
			name: "malformed JSON is reported as invalid JSON",
			args: `{"files":`,
			want: []string{"not valid JSON"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got stringifiedInputTestT
			err := decodeStructuredToolInput(test.args, &got)
			if err == nil {
				t.Fatalf("decodeStructuredToolInput(%s) error = nil, want error", test.args)
			}
			message := err.Error()
			for _, want := range test.want {
				if !strings.Contains(message, want) {
					t.Errorf("error %q does not contain %q", message, want)
				}
			}
			for _, leak := range []string{"Go struct", "stringifiedInputTestT", "[]string", "json:"} {
				if strings.Contains(message, leak) {
					t.Errorf("error %q leaks the internal Go type detail %q", message, leak)
				}
			}
		})
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

func TestNewEinoAgentToolRejectsUnstructuredNativeTool(t *testing.T) {
	if _, err := newEinoAgentTool(nil, &fakeAgentTool{enabled: true}); err == nil || !strings.Contains(err.Error(), "must implement StructuredTool or JSONSchemaTool") {
		t.Fatalf("newEinoAgentTool() error = %v, want StructuredTool requirement", err)
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

			const denialReason = "This request is outside the approved scope."
			if err := ResolveWritePermissionRequest(reqID, test.decision, denialReason); err != nil {
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
				if !strings.Contains(err.Error(), denialReason) {
					t.Fatalf("RequestToolPermission() error = %v, want denial reason", err)
				}
			}

			agent.toolPermissionMu.Lock()
			stored := agent.toolPermissions["fake.tool"].decision != toolPermissionUndecided
			agent.toolPermissionMu.Unlock()
			if stored != test.wantStored {
				t.Fatalf("decision stored = %v, want %v", stored, test.wantStored)
			}
			if test.decision == "deny-prompt" {
				err := agent.RequestToolPermission(context.Background(), "fake.tool")
				if !errors.Is(err, ErrToolPermissionRefused) || !strings.Contains(err.Error(), denialReason) {
					t.Fatalf("subsequent prompt-scoped refusal = %v, want stored denial reason", err)
				}
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
	if !strings.Contains(streamed.String(), "Continuing after max steps (2/4)") {
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

func TestIsRateLimitError(t *testing.T) {
	for _, err := range []error{
		&openai.APIError{HTTPStatusCode: http.StatusTooManyRequests, HTTPStatus: "429 Too Many Requests"},
		errors.New("[NodeRunError] error, status code: 429, status: 429 Too Many Requests, message: OpenRouter could not verify available credits for this request in time. Retry shortly."),
	} {
		if !isRateLimitError(err) {
			t.Fatalf("isRateLimitError(%q) = false, want true", err)
		}
	}
	if isRateLimitError(errors.New("status code: 503, service unavailable")) {
		t.Fatal("isRateLimitError() = true for non-rate-limit error")
	}
}

func TestRateLimitRetryDelayHonorsRetryAfter(t *testing.T) {
	now := time.Date(2026, time.September, 28, 22, 0, 0, 0, time.UTC)
	for name, value := range map[string]string{
		"seconds": "2.5",
		"date":    now.Add(3 * time.Second).Format(http.TimeFormat),
	} {
		t.Run(name, func(t *testing.T) {
			err := &retryAfterTestError{
				Response: &http.Response{Header: http.Header{"Retry-After": []string{value}}},
				err:      errors.New("429 Too Many Requests"),
			}
			want := 2500 * time.Millisecond
			if name == "date" {
				want = 3 * time.Second
			}
			if got := rateLimitRetryDelay(err, 1, now); got != want {
				t.Fatalf("rateLimitRetryDelay() = %s, want %s", got, want)
			}
		})
	}
	if got := rateLimitRetryDelay(errors.New("429 Too Many Requests"), 3, now); got != 4*time.Second {
		t.Fatalf("fallback delay = %s, want 4s", got)
	}
	longRetry := &retryAfterTestError{
		Response: &http.Response{Header: http.Header{"Retry-After": []string{"45"}}},
		err:      errors.New("429 Too Many Requests"),
	}
	if got := rateLimitRetryDelay(longRetry, 3, now); got != 45*time.Second {
		t.Fatalf("long Retry-After delay = %s, want 45s", got)
	}
	millisecondRetry := &retryAfterTestError{
		Response: &http.Response{Header: http.Header{"Retry-After-Ms": []string{"125"}}},
		err:      errors.New("429 Too Many Requests"),
	}
	if got := rateLimitRetryDelay(millisecondRetry, 1, now); got != 125*time.Millisecond {
		t.Fatalf("Retry-After-Ms delay = %s, want 125ms", got)
	}
}

func TestEinoRuntimeRetriesRateLimitWithCheckpoint(t *testing.T) {
	var calls int
	var retryMessages []*schema.Message
	runtime := &einoRuntime{agent: &Agent{}}
	runtime.boundedWindowRunner = func(_ context.Context, messages []*schema.Message, _ func(string)) (string, error) {
		calls++
		if calls == 1 {
			return "partial", &retryAfterTestError{
				Response: &http.Response{Header: http.Header{"Retry-After": []string{"0.001"}}},
				err:      errors.New("429 Too Many Requests"),
			}
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
	if result != "partialrecovered" || calls != 2 {
		t.Fatalf("result = %q calls = %d, want partialrecovered and 2", result, calls)
	}
	if len(retryMessages) != 2 || !strings.Contains(retryMessages[1].Content, "Continuation checkpoint") {
		t.Fatalf("retry messages = %+v, want checkpoint continuation", retryMessages)
	}
	if !strings.Contains(streamed.String(), "Rate limited; retrying in 1ms (1/3)") {
		t.Fatalf("streamed output = %q, want rate-limit progress marker", streamed.String())
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
	if !strings.Contains(streamed.String(), "Continuing after max steps (2/2)") {
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
