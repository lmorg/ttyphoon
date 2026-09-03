package userquestion

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

type AskUser struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&AskUser{})
}

//go:embed description.md
var description string

func (t *AskUser) New(agt aitypes.Agent) (aitypes.Tool, error) {
	return &AskUser{agent: agt, enabled: true}, nil
}

func (t *AskUser) Enabled() bool { return t.enabled }
func (t *AskUser) Toggle()       { t.enabled = !t.enabled }
func (t *AskUser) Name() string  { return "askUser" }
func (t *AskUser) Path() string  { return "internal" }
func (t *AskUser) Description() string {
	return strings.TrimSpace(description)
}
func (t *AskUser) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "deny"}
}

type askUserInput struct {
	Question string   `json:"question"`
	Choices  []string `json:"choices,omitempty"`
}

func (t *AskUser) Call(ctx context.Context, input string) (string, error) {
	var req askUserInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return fmt.Sprintf("ERROR: input must be valid JSON matching the tool schema: %s", err), nil
	}

	question := strings.TrimSpace(req.Question)
	if question == "" {
		return "ERROR: 'question' is required", nil
	}

	answer, err := agent.RequestUserQuestion(ctx, question, req.Choices)
	if err != nil {
		return fmt.Sprintf("ERROR: %s", err), nil
	}
	return fmt.Sprintf("User response: %s", answer), nil
}
