package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
)

func (agent *Agent) prepareRuntime() {
	if agent.fnCancel != nil {
		agent.fnCancel()
		agent.fnCancel = nil
	}

	if agent.runtime == nil {
		agent.runtime = newPreferredRuntime(agent)
	}
}

// RunLLMWithMessageStream calls the configured runtime with structured messages
// and streams text chunks.
func (agent *Agent) RunLLMWithMessageStream(ctx context.Context, messages []*schema.Message, streamCallback func(string)) (result string, err error) {
	agent.prepareRuntime()
	if agent.Meta != nil {
		ctx = WithSkillResourceDirectories(ctx, agent.Meta.SkillResources)
	}
	return agent.runtime.RunLLMWithMessageStream(ctx, messages, streamCallback)
}

// RunLLMWithStream calls the configured runtime and streams text chunks.
// Use ai package helpers to construct provider-appropriate prompts.
func (agent *Agent) RunLLMWithStream(ctx context.Context, prompt string, streamCallback func(string)) (result string, err error) {
	agent.prepareRuntime()
	if agent.Meta != nil {
		ctx = WithSkillResourceDirectories(ctx, agent.Meta.SkillResources)
	}
	return agent.runtime.RunLLMWithStream(ctx, prompt, streamCallback)
}

func (agent *Agent) RunSkillSelection(ctx context.Context, messages []*schema.Message) (string, error) {
	agent.prepareRuntime()
	selector, ok := agent.runtime.(interface {
		RunSkillSelection(context.Context, []*schema.Message) (string, error)
	})
	if !ok {
		return "", fmt.Errorf("agent runtime does not support skill selection")
	}
	return selector.RunSkillSelection(ctx, messages)
}
