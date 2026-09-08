package shell

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/types"
)

type EnvVars struct {
	agent   aitypes.Agent
	enabled bool
	term    types.Term
}

func init() {
	agent.ToolsAdd(&EnvVars{})
}

//go:embed cmdline_description.md
var envVarsDescription string

func (t *EnvVars) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &EnvVars{agent: agent, enabled: false}, nil
}

func (t *EnvVars) Enabled() bool { return t.enabled }
func (t *EnvVars) Toggle()       { t.enabled = !t.enabled }

func (t *EnvVars) Name() string { return "envVars" }
func (t *EnvVars) Path() string { return "internal" }
func (t *EnvVars) Description() string {
	t.term = t.agent.Renderer().ActiveTile().GetTerm()
	return envVarsDescription
}
func (t *EnvVars) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

func (t *EnvVars) Call(ctx context.Context, input string) (string, error) {
	b, err := json.Marshal(t.term.GetEnvVars())
	if err != nil {
		return "", err
	}
	return string(b), nil
}
