package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/subagent"
)

type Subagent struct {
	agent   aitypes.Agent
	enabled bool
}

type subagentRequest struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

type subagentConfig interface {
	ProviderName() string
	ModelName() string
	EnvironmentValue(string) string
}

func init() {
	agent.ToolsAdd(&Subagent{})
}

func (t *Subagent) New(agt aitypes.Agent) (aitypes.Tool, error) {
	return &Subagent{agent: agt, enabled: true}, nil
}

func (t *Subagent) Enabled() bool       { return t.enabled }
func (t *Subagent) Toggle()             { t.enabled = !t.enabled }
func (t *Subagent) Name() string        { return "subagent" }
func (t *Subagent) Path() string        { return "internal" }
func (t *Subagent) StreamsOutput() bool { return true }
func (t *Subagent) Description() string {
	return "Runs a stateless, tool-free sub-agent with a user prompt. Input must be JSON with name and prompt strings."
}

func (t *Subagent) Call(ctx context.Context, input string) (string, error) {
	var request subagentRequest
	if err := json.Unmarshal([]byte(input), &request); err != nil {
		return "call the tool error: input must be valid json with name and prompt", nil
	}
	request.Name = strings.TrimSpace(request.Name)
	request.Prompt = strings.TrimSpace(request.Prompt)
	if request.Name == "" || request.Prompt == "" {
		return "call the tool error: name and prompt are required", nil
	}

	configured, ok := t.agent.(subagentConfig)
	if !ok {
		return "", fmt.Errorf("sub-agent is unavailable for this AI runtime")
	}

	return subagent.New(configured.ProviderName(), configured.ModelName(), configured.EnvironmentValue).Run(ctx, subagent.Request{
		Name:       request.Name,
		Prompt:     request.Prompt,
		EmitStream: agent.EmitAIStreamToolProgress(ctx),
	})
}
