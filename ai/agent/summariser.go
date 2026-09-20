package agent

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/lmorg/ttyphoon/ai/subagent"
)

// summariserPrefix is prepended to every summarised tool output so the main
// agent knows the payload it received is a compression rather than raw data.
// The system prompt tells the model to treat this prefix as a signal that
// details (JSON schemas, exact numbers, IDs, etc.) may need to be re-fetched.
const summariserPrefix = "[output summarised by subagent]\n\n"

//go:embed summariser_prompt.md
var summariserSystemPrompt string

// summariseToolOutput runs a stateless one-shot chat completion using the same
// LLM service/model as the main agent to compress a single tool's output.
// Called by einoAgentTool.InvokableRun when the raw output would push the main
// agent's conversation over its context budget.
func (r *einoRuntime) summariseToolOutput(ctx context.Context, toolName, toolInput, rawOutput string) (string, error) {
	content, err := subagent.New(r.agent.ProviderName(), r.agent.SummariseModelName(), r.agent.EnvironmentValue).Run(ctx, subagent.Request{
		SystemPrompt: summariserSystemPrompt,
		Prompt: fmt.Sprintf(
			"Tool name: %s\nTool input arguments (JSON):\n%s\n\nTool raw output:\n%s",
			toolName, toolInput, rawOutput,
		),
		EmitStream:   EmitAIStreamToolProgress(ctx),
		StreamPrefix: summariserStreamOpenMarkdown(),
		StreamSuffix: summariserStreamCloseMarkdown(),
		FormatStreamChunk: func(text string) string {
			return text
		},
	})
	if err != nil {
		return "", err
	}
	return summariserPrefix + content, nil
}
