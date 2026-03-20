package agent

import (
	"context"
	"log"
	"strings"

	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

func llmOpenAI(agent *Agent) (llms.Model, error) {
	return openai.New(openai.WithModel(agent.ModelName()))
}

func llmAnthropic(agent *Agent) (llms.Model, error) {
	return anthropic.New(anthropic.WithModel(agent.ModelName()))
}

func llmOllama(agent *Agent) (llms.Model, error) {
	return ollama.New(ollama.WithModel(agent.ModelName()))
}

func initLLM(agent *Agent) error {
	var (
		model llms.Model
		err   error
	)

	switch agent.ServiceName() {
	case LLM_ANTHROPIC:
		model, err = llmAnthropic(agent)
	case LLM_OPENAI:
		model, err = llmOpenAI(agent)
	case LLM_OLLAMA:
		model, err = llmOllama(agent)
	default:
		panic("unexpected branch")
	}
	if err != nil {
		return err
	}

	var agentTools []tools.Tool
	for _, tool := range agent._tools {
		if tool.Enabled() {
			agentTools = append(agentTools, tool)
		}
	}

	agent.executor = agents.NewExecutor(
		agents.NewOneShotAgent(model, agentTools, agents.WithMaxIterations(agent.MaxIterations())),
		agents.WithMaxIterations(agent.MaxIterations()),
	)

	return nil
}

const _ERR_UNABLE_TO_PARSE_AGENT_OUTPUT = "unable to parse agent output: "

// RunLLMWithStream calls the LLM with the prompt string and streams responses.
// Use `ai` package to create specific prompts.
func (agent *Agent) RunLLMWithStream(ctx context.Context, prompt string, streamCallback func(string)) (result string, err error) {
	/*if debug.Trace {
		log.Printf("RunLLMWithStream prompt:\n%s", prompt)
		defer func() {
			log.Printf("RunLLMWithStream result:\n%s", result)
			log.Printf("RunLLMWithStream error: %v", err)
		}()
	}*/

	if agent.fnCancel != nil {
		agent.fnCancel()
		agent.fnCancel = nil
	}

	if agent.executor == nil {
		err := initLLM(agent)
		if err != nil {
			return "", err
		}
	}

	result, err = chains.Run(
		ctx,
		agent.executor,
		prompt,
		chains.WithTemperature(1),
		chains.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if streamCallback != nil && len(chunk) > 0 {
				streamCallback(string(chunk))
			}
			return nil
		}),
	)

	if err == nil {
		return result, nil
	}

	if strings.HasPrefix(err.Error(), _ERR_UNABLE_TO_PARSE_AGENT_OUTPUT) {
		log.Println(err)
		response := err.Error()[len(_ERR_UNABLE_TO_PARSE_AGENT_OUTPUT):]
		if streamCallback != nil {
			streamCallback(response)
		}
		return response, nil
	}
	return result, err
}
