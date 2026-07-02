package agent

import "context"

// RunLLMWithStream calls the configured runtime and streams text chunks.
// Use ai package helpers to construct provider-appropriate prompts.
func (agent *Agent) RunLLMWithStream(ctx context.Context, prompt string, streamCallback func(string)) (result string, err error) {
	if agent.fnCancel != nil {
		agent.fnCancel()
		agent.fnCancel = nil
	}

	if agent.runtime == nil {
		agent.runtime = newPreferredRuntime(agent)
	}

	return agent.runtime.RunLLMWithStream(ctx, prompt, streamCallback)
}
