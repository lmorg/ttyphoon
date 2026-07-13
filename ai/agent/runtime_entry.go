package agent

import (
	"context"

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
	return agent.runtime.RunLLMWithMessageStream(ctx, messages, streamCallback)
}

// RunLLMWithStream calls the configured runtime and streams text chunks.
// Use ai package helpers to construct provider-appropriate prompts.
func (agent *Agent) RunLLMWithStream(ctx context.Context, prompt string, streamCallback func(string)) (result string, err error) {
	agent.prepareRuntime()
	return agent.runtime.RunLLMWithStream(ctx, prompt, streamCallback)
}
