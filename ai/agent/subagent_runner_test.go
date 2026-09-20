package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

func TestSubagentToolSet_MatchesSubagentToolNames(t *testing.T) {
	names := []string{"always", "approval", "forbidden"}
	tools := make([]aitypes.Tool, 0, len(names))
	for _, name := range names {
		tools = append(tools, &toolStateTestTool{name: name})
	}

	agent := &Agent{
		_tools: tools,
		toolStates: map[string]string{
			"always":   ToolStateAlways,
			"approval": ToolStateApproval,
			"askUser":  ToolStateAlways,
		},
		subagentTools: map[string]bool{"always": true, "approval": true, "askUser": true},
	}

	set := agent.subagentToolSet()
	if len(set) != 1 || set[0].Name() != "always" {
		got := make([]string, len(set))
		for i, tool := range set {
			got[i] = tool.Name()
		}
		t.Fatalf("subagentToolSet() = %v, want [always]", got)
	}

	// The delegable set and the advertised names must not drift apart.
	if advertised := agent.SubagentToolNames(); len(advertised) != len(set) || advertised[0] != set[0].Name() {
		t.Fatalf("SubagentToolNames() = %v, subagentToolSet() = %v", advertised, set[0].Name())
	}
}

// Without this the sub-agent would build a react agent with no tools and quietly
// answer from the prompt alone.
func TestRunSubagentWithTools_ErrorsWhenNoToolsDelegable(t *testing.T) {
	agent := &Agent{
		_tools:     []aitypes.Tool{&toolStateTestTool{name: "approval"}},
		toolStates: map[string]string{"approval": ToolStateApproval},
	}

	_, err := agent.RunSubagentWithTools(context.Background(), "system", "prompt", func(string) {})
	if err == nil {
		t.Fatal("RunSubagentWithTools() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "no tools are enabled for sub-agents") {
		t.Fatalf("RunSubagentWithTools() error = %q, want no-tools message", err)
	}
}
