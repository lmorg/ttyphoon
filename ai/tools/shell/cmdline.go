package shell

import (
	"context"
	_ "embed"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

type CommandLine struct {
	agent   aitypes.Agent
	enabled bool
	term    types.Term
}

func init() {
	agent.ToolsAdd(&CommandLine{})
}

//go:embed cmdline_description.md
var cmdlineDescription string

func (t *CommandLine) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &CommandLine{agent: agent, enabled: false}, nil
}

func (t *CommandLine) Enabled() bool { return t.enabled }
func (t *CommandLine) Toggle()       { t.enabled = !t.enabled }

func (t *CommandLine) Name() string { return "commandLine" }
func (t *CommandLine) Path() string { return "internal" }
func (t *CommandLine) Description() string {
	t.term = t.agent.Renderer().ActiveTile().GetTerm()
	return cmdlineDescription + "\n$SHELL=" + t.shellName()
}
func (t *CommandLine) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

func (t *CommandLine) shellName() string {
	return t.term.GetEnvVars()["SHELL"]
}

func (t *CommandLine) Call(ctx context.Context, input string) (string, error) {
	debug.Log(input)

	/*var request cmdlineT
	if err := json.Unmarshal([]byte(input), &request); err != nil {
		return fmt.Sprintf("ERROR: input must be valid JSON matching the tool schema: %s", err), nil
	}*/

	t.term.Reply([]byte(input))

	result := "todo"
	//////
	debug.Log(result)
	return result, nil
}
