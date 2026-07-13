package agent

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// agentRuntime is the minimal execution boundary used by Agent.
// Concrete SDK implementations are wired behind this boundary.
type agentRuntime interface {
	RunLLMWithMessageStream(ctx context.Context, messages []*schema.Message, streamCallback func(string)) (string, error)
	RunLLMWithStream(ctx context.Context, prompt string, streamCallback func(string)) (string, error)
	Reset()
}
