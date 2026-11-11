package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

type testManager struct {
	aitypes.Agent
	servers []aitypes.McpServer
	err     error
	started string
}

func (m *testManager) ConfiguredMcpServers() ([]aitypes.McpServer, error) {
	return m.servers, m.err
}

func (m *testManager) StartConfiguredMcpServer(_ context.Context, key string) (string, error) {
	m.started = key
	return "example", m.err
}

func TestStartServerUsesCapabilityAndEmbeddedDescription(t *testing.T) {
	manager := &testManager{servers: []aitypes.McpServer{{
		Key: "config.json::example", Name: "config::example", Description: "Inspect tickets",
	}}}
	created, err := (&StartServer{}).New(manager)
	if err != nil {
		t.Fatal(err)
	}
	tool := created.(*StartServer)
	if !tool.Enabled() || tool.DefaultPermissions().Invocation != "askPermission" || tool.DefaultPermissions().Subagents != "deny" {
		t.Fatal("startup defaults changed")
	}
	if !strings.Contains(tool.Description(), "- `config::example`: Inspect tickets") {
		t.Fatalf("configured server list is missing or malformed: %s", tool.Description())
	}
	if strings.Contains(tool.Description(), "config.json") || strings.Contains(tool.Description(), `"key"`) {
		t.Fatal("tool description exposed the internal config filename or key")
	}
	input := &StartServerInput{ServerKey: "config::example", Justification: "Find ticket requirements"}
	details := tool.CallDescription(context.Background(), input)
	if !strings.Contains(details, `Enable MCP server "config::example"`) || !strings.Contains(details, input.Justification) {
		t.Fatalf("approval details = %q", details)
	}
	output, err := tool.CallStructured(context.Background(), input)
	if err != nil || manager.started != input.ServerKey || !strings.Contains(output, `"example" is enabled`) {
		t.Fatalf("startup result = %q, %v; key = %q", output, err, manager.started)
	}
	tool.Toggle()
	if tool.Enabled() {
		t.Fatal("toggle did not disable tool")
	}
}

func TestStartServerReportsCapabilityAndBackendErrors(t *testing.T) {
	if _, err := (&StartServer{}).New(nil); err == nil {
		t.Fatal("accepted an agent without MCP capabilities")
	}
	manager := &testManager{err: errors.New("configuration unavailable")}
	created, err := (&StartServer{}).New(manager)
	if err != nil {
		t.Fatal(err)
	}
	tool := created.(*StartServer)
	if !strings.Contains(tool.Description(), "Catalog unavailable: configuration unavailable") {
		t.Fatal("catalog error was hidden")
	}
	input := &StartServerInput{ServerKey: "config::example", Justification: "Inspect tickets"}
	if !strings.Contains(tool.CallDescription(context.Background(), input), manager.err.Error()) {
		t.Fatal("approval details hid the catalog error")
	}
	if _, err := tool.CallStructured(context.Background(), input); !errors.Is(err, manager.err) {
		t.Fatalf("backend error = %v", err)
	}
	manager.started = ""
	for _, invalid := range []any{nil, "invalid", (*StartServerInput)(nil), &StartServerInput{Justification: " "}} {
		if _, err := tool.CallStructured(context.Background(), invalid); err == nil {
			t.Fatalf("accepted invalid request %#v", invalid)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := tool.CallStructured(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled call = %v", err)
	}
	if manager.started != "" {
		t.Fatal("invalid or cancelled call reached backend")
	}
}
