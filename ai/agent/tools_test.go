package agent

import (
	"context"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

type toolStateTestTool struct{ name string }

func (t *toolStateTestTool) New(aitypes.Agent) (aitypes.Tool, error) { return t, nil }
func (t *toolStateTestTool) Enabled() bool                           { return true }
func (t *toolStateTestTool) Toggle()                                 {}
func (t *toolStateTestTool) Name() string                            { return t.name }
func (t *toolStateTestTool) Path() string                            { return "internal" }
func (t *toolStateTestTool) Description() string                     { return "test tool" }
func (t *toolStateTestTool) Call(context.Context, string) (string, error) {
	return "", nil
}

func (t *toolStateTestTool) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "allow"}
}

func TestToolDefaultPermissions(t *testing.T) {
	tool := &toolStateTestTool{name: "test"}
	agent := &Agent{_tools: []aitypes.Tool{tool}, toolStates: map[string]string{}}

	if got := agent.ToolState(tool.Name()); got != ToolStateApproval {
		t.Fatalf("ToolState() = %q, want %q", got, ToolStateApproval)
	}
	if !agent.ToolAllowedInSubagent(tool.Name()) {
		t.Fatal("ToolAllowedInSubagent() = false, want true")
	}
}

// Sub-agents run non-interactively, so only always-allowed tools qualify.
func TestSubagentToolNames_OnlyIncludesAlwaysAllowedTools(t *testing.T) {
	names := []string{"always", "approval", "session", "disabled", "notAllowed"}
	tools := make([]aitypes.Tool, 0, len(names))
	for _, name := range names {
		tools = append(tools, &toolStateTestTool{name: name})
	}

	agent := &Agent{
		_tools: tools,
		toolStates: map[string]string{
			"always":     ToolStateAlways,
			"approval":   ToolStateApproval,
			"session":    ToolStateSession,
			"disabled":   ToolStateDisabled,
			"notAllowed": ToolStateAlways,
		},
		subagentTools: map[string]bool{
			"always":     true,
			"approval":   true,
			"session":    true,
			"disabled":   true,
			"notAllowed": false,
		},
	}

	if got := agent.SubagentToolNames(); len(got) != 1 || got[0] != "always" {
		t.Fatalf("SubagentToolNames() = %v, want [always]", got)
	}
}

// askUser and the sub-agent tools must stay out even when explicitly configured in.
func TestSubagentForbiddenToolsCannotBeEnabled(t *testing.T) {
	forbidden := []string{TOOL_DELEGATE, TOOL_REPORT, TOOL_ASK_USER}

	tools := make([]aitypes.Tool, 0, len(forbidden))
	states := map[string]string{}
	allowed := map[string]bool{}
	for _, name := range forbidden {
		tools = append(tools, &toolStateTestTool{name: name})
		states[name] = ToolStateAlways
		allowed[name] = true
	}

	agent := &Agent{_tools: tools, toolStates: states, subagentTools: allowed}

	if got := agent.SubagentToolNames(); len(got) != 0 {
		t.Fatalf("SubagentToolNames() = %v, want none", got)
	}

	for _, name := range forbidden {
		if agent.ToolAllowedInSubagent(name) {
			t.Fatalf("ToolAllowedInSubagent(%q) = true, want false", name)
		}
		if err := agent.SetToolAllowedInSubagent(name, true); err == nil {
			t.Fatalf("SetToolAllowedInSubagent(%q, true) = nil, want an error", name)
		}
	}
}

func TestMCPToolDefaultPermissions(t *testing.T) {
	tool := &mcpTool{}
	defaults := tool.DefaultPermissions()
	if defaults.Invocation != "alwaysAllow" || defaults.Subagents != "allow" {
		t.Fatalf("MCP defaults = %#v, want alwaysAllow/allow", defaults)
	}
}

func TestSetToolState_KeepsWriteToolsIndependent(t *testing.T) {
	agent := &Agent{
		toolStates: map[string]string{},
		_tools: []aitypes.Tool{
			&toolStateTestTool{name: "writeFile"},
			&toolStateTestTool{name: "patchFile"},
			&toolStateTestTool{name: "insertLines"},
		},
	}

	if err := agent.SetToolState("patchFile", ToolStateAlways); err != nil {
		t.Fatalf("SetToolState() error = %v", err)
	}
	if got := agent.ToolState("patchFile"); got != ToolStateAlways {
		t.Fatalf("patchFile state = %q, want %q", got, ToolStateAlways)
	}
	if got := agent.ToolState("writeFile"); got != ToolStateApproval {
		t.Fatalf("writeFile state = %q, want %q", got, ToolStateApproval)
	}
	if got := agent.ToolState("insertLines"); got != ToolStateApproval {
		t.Fatalf("insertLines state = %q, want %q", got, ToolStateApproval)
	}
}
