package cmdline

import (
	"context"
	_ "embed"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
)

type CommandLine struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&CommandLine{})
}

//go:embed description.md
var Description string

func (t *CommandLine) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &CommandLine{agent: agent, enabled: true}, nil
}

func (t *CommandLine) Enabled() bool { return t.enabled }
func (t *CommandLine) Toggle()       { t.enabled = !t.enabled }

func (t *CommandLine) Name() string        { return "insertLines" }
func (t *CommandLine) Path() string        { return "internal" }
func (t *CommandLine) Description() string { return Description }
func (t *CommandLine) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

/*type cmdlineT struct {
	CmdLine string `json:"commandLine"`
}*/

func (t *CommandLine) Call(ctx context.Context, input string) (string, error) {
	debug.Log(input)

	/*var request cmdlineT
	if err := json.Unmarshal([]byte(input), &request); err != nil {
		return fmt.Sprintf("ERROR: input must be valid JSON matching the tool schema: %s", err), nil
	}*/

	t.agent.Renderer().ActiveTile().GetTerm().Reply([]byte(input))

	//////
	debug.Log(result)
	return result, nil
}
