package ai

import (
	"context"

	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

const _OPENAI_MODEL = "gpt-4" //"o4-mini"
const _ANTHROPIC_MODEL = "claude-3-7-sonnet-20250219"

func llmOpenAI() (llms.Model, error) {
	return openai.New(openai.WithModel(_OPENAI_MODEL))
}

func llmAnthropic() (llms.Model, error) {
	return anthropic.New(anthropic.WithModel(_ANTHROPIC_MODEL))
}

func RunLLM(model llms.Model, meta *Meta, userPrompt string) (string, error) {
	query := getPrompt(meta.CmdLine, meta.OutputBlock, userPrompt)

	agentTools := []tools.Tool{
		LocalFile{meta: meta},
		Directory{meta: meta},
	}

	errHandler := agents.NewParserErrorHandler(func(s string) string { return "TODO" })

	agent := agents.NewOneShotAgent(model, agentTools)
	executor := agents.NewExecutor(agent, agents.WithMaxIterations(10), agents.WithParserErrorHandler(errHandler))

	return chains.Run(context.Background(), executor, query, chains.WithTemperature(1))
	//answer, err := llm.Call(context.Background(), query, llms.WithTemperature(1), llms.WithTools(agentTools))
}
