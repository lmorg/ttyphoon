package agent

import (
	"context"
	"strings"
	"time"

	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

func llmOpenAI(meta *Meta) (llms.Model, error) {
	return openai.New(openai.WithModel(meta.ModelName()))
}

func llmAnthropic(meta *Meta) (llms.Model, error) {
	return anthropic.New(anthropic.WithModel(meta.ModelName()))
}

func initLLM(meta *Meta) error {
	var (
		model llms.Model
		err   error
	)

	switch meta.ServiceName() {
	case LLM_ANTHROPIC:
		model, err = llmAnthropic(meta)
	case LLM_OPENAI:
		model, err = llmOpenAI(meta)
	default:
		panic("unexpected branch")
	}
	if err != nil {
		return err
	}

	var agentTools []tools.Tool
	for _, tool := range meta._tools {
		if tool.Enabled() {
			agentTools = append(agentTools, tool)
		}
	}

	agent := agents.NewOneShotAgent(model, agentTools, agents.WithMaxIterations(100))
	meta.executor = agents.NewExecutor(agent)

	return nil
}

const _ERR_UNABLE_TO_PARSE_AGENT_OUTPUT = "unable to parse agent output: "

// RunLLM calls the LLM with the prompt string.
// Use `ai` package to create specific prompts.
func (meta *Meta) RunLLM(prompt string) (string, error) {
	if meta.fnCancel != nil {
		meta.fnCancel()
	}

	if meta.executor == nil {
		err := initLLM(meta)
		if err != nil {
			return "", err
		}
	}

	var ctx context.Context
	ctx, meta.fnCancel = context.WithTimeout(context.Background(), 5*time.Minute)

	result, err := chains.Run(ctx, meta.executor, prompt, chains.WithTemperature(1))
	if err == nil {
		return result, nil
	}

	if strings.HasPrefix(err.Error(), _ERR_UNABLE_TO_PARSE_AGENT_OUTPUT) {
		return err.Error()[len(_ERR_UNABLE_TO_PARSE_AGENT_OUTPUT):], nil
	}
	return result, err
}
