package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adrg/xdg"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/tools/mcp"
)

type startMcpServerTool = mcp.StartServer
type startMcpServerInput = mcp.StartServerInput

func mcpLaunchConfig(t *testing.T, url string) string {
	t.Helper()
	old := xdg.ConfigDirs
	xdg.ConfigDirs = []string{t.TempDir()}
	t.Cleanup(func() { xdg.ConfigDirs = old })
	directory := filepath.Join(xdg.ConfigDirs[0], "ttyphoon", "mcp")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "test.json")
	content := fmt.Sprintf(`{"mcp":{"servers":{"example":{"type":"http","url":%q,"description":"Inspect project tickets","env":{"SECRET":"private-token"}}}}}`, url)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return makeMcpServerKey(path, "example")
}

func mcpLaunchAgent(t *testing.T) (*Agent, *startMcpServerTool) {
	t.Helper()
	owner := &Agent{
		renderer:      &mcpLoadTestRenderer{started: make(chan func(), 4)},
		maxIterations: 12,
	}
	created, err := (&startMcpServerTool{}).New(owner)
	if err != nil {
		t.Fatal(err)
	}
	tool := created.(*startMcpServerTool)
	owner._tools = []aitypes.Tool{tool}
	t.Cleanup(func() {
		for _, connection := range owner._mcpServers {
			_ = connection.Close()
		}
	})
	return owner, tool
}

func TestMcpLaunchDefaultPermissionsAndCatalog(t *testing.T) {
	key := mcpLaunchConfig(t, "https://private.example/mcp?token=secret")
	owner, tool := mcpLaunchAgent(t)
	if !tool.Enabled() || owner.ToolState(tool.Name()) != ToolStateApproval {
		t.Fatal("MCP launch tool must be enabled and ask for approval by default")
	}
	if err := owner.SetToolAllowedInSubagent(tool.Name(), true); err == nil {
		t.Fatal("interactive MCP launch was allowed in a subagent")
	}
	description := tool.Description()
	for _, expected := range []string{"example", "Inspect project tickets", key} {
		if !strings.Contains(description, expected) {
			t.Errorf("catalog missing %q: %s", expected, description)
		}
	}
	for _, secret := range []string{"private-token", "private.example", "SECRET"} {
		if strings.Contains(description, secret) {
			t.Errorf("catalog exposes server configuration %q", secret)
		}
	}
	if owner.ListMcpServers()[0]["description"] != "Inspect project tickets" {
		t.Fatal("AI settings catalog lost the server description")
	}
	schema := string(toolInputSchemaJSON(tool))
	if !strings.Contains(schema, `"serverKey"`) || !strings.Contains(schema, `"justification"`) {
		t.Fatalf("missing structured input fields: %s", schema)
	}
}

func TestMcpLaunchValidatesBeforeStarting(t *testing.T) {
	key := mcpLaunchConfig(t, "https://unused.example/mcp")
	owner, tool := mcpLaunchAgent(t)
	for _, request := range []*startMcpServerInput{
		{ServerKey: key},
		{ServerKey: "/tmp/unconfigured.json::example", Justification: "Inspect tickets"},
	} {
		if _, err := tool.CallStructured(context.Background(), request); err == nil {
			t.Fatalf("accepted invalid request %+v", request)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := tool.CallStructured(ctx, &startMcpServerInput{ServerKey: key, Justification: "Inspect tickets"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled startup returned %v", err)
	}
	if len(owner._mcpServers) != 0 {
		t.Fatal("invalid request started a server")
	}
}

func TestMcpLaunchApprovalIncludesNameAndJustificationAndDenialDoesNotStart(t *testing.T) {
	key := mcpLaunchConfig(t, "https://unused.example/mcp")
	owner, tool := mcpLaunchAgent(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	prompt := make(chan string, 1)
	emitter := &aiStreamEmitter{fn: func(text string) { prompt <- text }}
	ctx = withAIStreamCallback(ctx, emitter)
	adapter, err := newEinoAgentTool(&einoRuntime{agent: owner}, tool)
	if err != nil {
		t.Fatal(err)
	}
	request := startMcpServerInput{ServerKey: key, Justification: "Find the acceptance criteria for this ticket"}
	input, _ := json.Marshal(request)
	result := make(chan error, 1)
	go func() {
		output, err := adapter.InvokableRun(ctx, string(input))
		if err == nil && !strings.Contains(output, "Do not retry") {
			err = fmt.Errorf("denial result did not reach the model: %s", output)
		}
		result <- err
	}()
	var markdown string
	for !strings.Contains(markdown, "Allow this invocation") {
		select {
		case part := <-prompt:
			markdown += part
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	for _, expected := range []string{`Enable MCP server "example"`, request.Justification} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("approval missing %q: %s", expected, markdown)
		}
	}
	writePermissionRequests.mu.Lock()
	var id string
	for candidate, request := range writePermissionRequests.m {
		if request.toolName == tool.Name() {
			id = candidate
		}
	}
	writePermissionRequests.mu.Unlock()
	if err := ResolveWritePermissionRequest(id, "deny-once", ""); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if len(owner._mcpServers) != 0 {
		t.Fatal("denied invocation started a server")
	}
}

func TestMcpLaunchStartsConfiguredServerAndRefreshesToolsWithinRun(t *testing.T) {
	server := mcpLifecycleFixture(t)
	key := mcpLaunchConfig(t, server.URL)
	owner, launcher := mcpLaunchAgent(t)
	owner.toolStates = map[string]string{launcher.Name(): ToolStateAlways}
	companionCalls := 0
	owner._tools = append(owner._tools, &fakeStructuredAgentTool{
		fakeAgentTool: &fakeAgentTool{enabled: true},
		onCall:        func() { companionCalls++ },
	})
	input, _ := json.Marshal(startMcpServerInput{ServerKey: key, Justification: "Inspect project tickets"})
	modelCalls := 0
	model := &toolRecoveryChatModel{generate: func(messages []*schema.Message) (*schema.Message, error) {
		modelCalls++
		switch modelCalls {
		case 1:
			return schema.AssistantMessage("", []schema.ToolCall{{
				ID: "launch", Type: "function",
				Function: schema.FunctionCall{Name: launcher.Name(), Arguments: string(input)},
			}, {
				ID: "parallel", Type: "function",
				Function: schema.FunctionCall{Name: "fake_tool", Arguments: `{"query":"parallel","json":{}}`},
			}}), nil
		case 2:
			results := make(map[string]string)
			for _, message := range messages {
				if message.Role == schema.Tool {
					results[message.ToolCallID] = message.Content
				}
			}
			if !strings.Contains(results["launch"], "is enabled") || results["parallel"] != "structured result" {
				return nil, fmt.Errorf("startup or parallel tool result missing from conversation: %+v", messages)
			}

			return schema.AssistantMessage("", []schema.ToolCall{{
				ID: "echo", Type: "function",
				Function: schema.FunctionCall{Name: "mcp_example_echo", Arguments: "{}"},
			}}), nil
		case 3:
			last := messages[len(messages)-1]
			if last.Role != schema.Tool || last.ToolCallID != "echo" || strings.Contains(last.Content, "not found") {
				return nil, fmt.Errorf("new MCP tool did not run: %+v", last)
			}
			return schema.AssistantMessage("Finished using the MCP server", nil), nil
		default:
			return nil, fmt.Errorf("unexpected model call %d", modelCalls)
		}
	}}
	runtime := &einoRuntime{agent: owner, chatModel: model}
	tools, err := runtime.toolsConfig()
	if err != nil {
		t.Fatal(err)
	}
	runtime.agentReact, err = react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: model, ToolsConfig: tools, MaxStep: 12,
		ToolReturnDirectly: map[string]struct{}{TOOL_START_MCP_SERVER: {}},
	})
	if err != nil {
		t.Fatal(err)
	}
	owner.runtime = runtime
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream := runtime.streamWithMcpLaunches(ctx, []*schema.Message{schema.UserMessage("Inspect the ticket")})
	defer stream.Close()
	var output strings.Builder
	for {
		message, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if message != nil {
			output.WriteString(message.Content)
		}
	}
	if output.String() != "Finished using the MCP server" || modelCalls != 3 || companionCalls != 1 || !owner.McpServerExists("example") {
		t.Fatalf("output=%q calls=%d companion calls=%d loaded=%v", output.String(), modelCalls, companionCalls, owner.McpServerExists("example"))
	}
}

func TestMcpLaunchFailureAndDenialContinueRun(t *testing.T) {
	for _, state := range []string{ToolStateAlways, ToolStateDenied} {
		t.Run(state, func(t *testing.T) {
			mcpLaunchConfig(t, "https://unused.example/mcp")
			owner, launcher := mcpLaunchAgent(t)
			owner.toolStates = map[string]string{launcher.Name(): state}
			input, _ := json.Marshal(startMcpServerInput{
				ServerKey: "/not/configured.json::example", Justification: "Inspect tickets",
			})
			calls := 0
			model := &toolRecoveryChatModel{generate: func(messages []*schema.Message) (*schema.Message, error) {
				calls++
				if calls == 1 {
					return schema.AssistantMessage("", []schema.ToolCall{{
						ID: "launch", Type: "function",
						Function: schema.FunctionCall{Name: launcher.Name(), Arguments: string(input)},
					}}), nil
				}
				last := messages[len(messages)-1]
				if calls != 2 || last.Role != schema.Tool || last.ToolCallID != "launch" {
					return nil, fmt.Errorf("lost failure history or retried startup: %+v", messages)
				}
				return schema.AssistantMessage("Continue without that server", nil), nil
			}}
			runtime := &einoRuntime{agent: owner, chatModel: model}
			tools, err := runtime.toolsConfig()
			if err != nil {
				t.Fatal(err)
			}
			runtime.agentReact, err = react.NewAgent(context.Background(), &react.AgentConfig{
				ToolCallingModel: model, ToolsConfig: tools, MaxStep: 12,
				ToolReturnDirectly: map[string]struct{}{TOOL_START_MCP_SERVER: {}},
			})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			stream := runtime.streamWithMcpLaunches(ctx, []*schema.Message{schema.UserMessage("Inspect tickets")})
			message, err := schema.ConcatMessageStream(stream)
			if err != nil {
				t.Fatal(err)
			}
			if message.Content != "Continue without that server" || calls != 2 || len(owner._mcpServers) != 0 {
				t.Fatalf("unexpected continuation: %+v; calls=%d", message, calls)
			}
		})
	}
}

func TestMcpLaunchParentCancellationStopsStartup(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	key := mcpLaunchConfig(t, server.URL)
	owner, launcher := mcpLaunchAgent(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := launcher.CallStructured(ctx, &startMcpServerInput{ServerKey: key, Justification: "Inspect tickets"})
		result <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("startup never contacted the configured server")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("startup did not preserve cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("startup ignored parent cancellation")
	}
	owner.mcpMu.Lock()
	defer owner.mcpMu.Unlock()
	if len(owner.mcpLoads) != 0 || len(owner._mcpServers) != 0 {
		t.Fatal("cancelled startup left a loaded server or a loading reservation")
	}
}
