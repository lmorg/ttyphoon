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

func TestSubagentToolNames_OnlyIncludesEnabledAllowedTools(t *testing.T) {
	allowed := &toolStateTestTool{name: "allowed"}
	disabled := &toolStateTestTool{name: "disabled"}
	subagent := &toolStateTestTool{name: "subagent"}
	agent := &Agent{
		_tools:        []aitypes.Tool{allowed, disabled, subagent},
		toolStates:    map[string]string{"disabled": ToolStateDisabled},
		subagentTools: map[string]bool{"allowed": true, "disabled": true, "subagent": true},
	}

	if got := agent.SubagentToolNames(); len(got) != 1 || got[0] != "allowed" {
		t.Fatalf("SubagentToolNames() = %v, want [allowed]", got)
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
