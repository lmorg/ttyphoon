package agent

import "context"

// agentRuntime is the minimal execution boundary used by Agent.
// Concrete SDK implementations are wired behind this boundary.
type agentRuntime interface {
	RunLLMWithStream(ctx context.Context, prompt string, streamCallback func(string)) (string, error)
	Reset()
}
