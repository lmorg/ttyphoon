package mcp

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

//go:embed description.md
var description string

type serverManager interface {
	ConfiguredMcpServers() ([]aitypes.McpServer, error)
	StartConfiguredMcpServer(context.Context, string) (string, error)
}

type StartServer struct {
	agent   serverManager
	enabled bool
}

type StartServerInput struct {
	ServerKey     string `json:"serverKey"`
	Justification string `json:"justification"`
}

func (t *StartServer) New(agt aitypes.Agent) (aitypes.Tool, error) {
	manager, ok := agt.(serverManager)
	if !ok {
		return nil, fmt.Errorf("startMcpServer requires an agent with MCP server management")
	}
	return &StartServer{agent: manager, enabled: true}, nil
}

func (t *StartServer) Enabled() bool { return t.enabled }
func (t *StartServer) Toggle()       { t.enabled = !t.enabled }
func (t *StartServer) Name() string  { return aitypes.ToolStartMcpServer }
func (t *StartServer) Path() string  { return "internal" }
func (t *StartServer) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}
func (t *StartServer) InputType() reflect.Type {
	return reflect.TypeOf(StartServerInput{})
}

func (t *StartServer) Description() string {
	text := strings.TrimSpace(description) + "\n\nConfigured MCP servers:\n"
	servers, err := t.agent.ConfiguredMcpServers()
	if err != nil {
		return text + "Catalog unavailable: " + err.Error()
	}
	catalog, err := json.Marshal(servers)
	if err != nil {
		return text + "Catalog encoding failed: " + err.Error()
	}
	return text + string(catalog)
}

func (t *StartServer) CallDescription(_ context.Context, input any) string {
	request, ok := input.(*StartServerInput)
	if !ok || request == nil {
		return "Enable a configured MCP server (invalid request)"
	}
	servers, err := t.agent.ConfiguredMcpServers()
	if err != nil {
		return "MCP server catalog unavailable: " + err.Error()
	}
	for _, server := range servers {
		if server.Key == request.ServerKey {
			return fmt.Sprintf("Enable MCP server %q\n\nJustification: %s", server.Name, request.Justification)
		}
	}
	return fmt.Sprintf("Invalid MCP server key: %q", request.ServerKey)
}

func (t *StartServer) CallStructured(ctx context.Context, input any) (string, error) {
	request, ok := input.(*StartServerInput)
	if !ok || request == nil {
		return "", fmt.Errorf("startMcpServer received invalid input %T", input)
	}
	if strings.TrimSpace(request.Justification) == "" {
		return "", fmt.Errorf("a justification is required to enable an MCP server")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	name, err := t.agent.StartConfiguredMcpServer(ctx, request.ServerKey)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("MCP server %q is enabled. Its tools are now available; use them to continue the task.", name), nil
}
