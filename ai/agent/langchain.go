package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
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

// RunLLM calls the LLM with the prompt string.
// Use `ai` package to create specific prompts.
func (agent *Agent) RunLLM(prompt string, sticky types.Notification) (result string, err error) {
	if debug.Trace {
		log.Printf("RunLLM prompt:\n%s", prompt)
		defer func() {
			log.Printf("RunLLM result:\n%s", result)
			log.Printf("RunLLM error: %v", err)
		}()
	}

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

	var ctx context.Context
	ctx, agent.fnCancel = context.WithTimeout(context.Background(), 5*time.Minute)
	sticky.UpdateCanceller(agent.fnCancel)

	result, err = chains.Run(ctx, agent.executor, prompt, chains.WithTemperature(1))
	if strings.Contains(result, "<max_iterations_reached/>") {
		go agent.renderer.DisplayNotification(types.NOTIFY_DEBUG, "Max iterations reached, resuming with updated context")
		return agent.RunLLM(fmt.Sprintf("%s\n\n# What's been learned so far\n%s", prompt, result), sticky)
	}

	if err == nil {
		return result, nil
	}

	if strings.HasPrefix(err.Error(), _ERR_UNABLE_TO_PARSE_AGENT_OUTPUT) {
		log.Println(err)
		return err.Error()[len(_ERR_UNABLE_TO_PARSE_AGENT_OUTPUT):], nil // bit of a kludge this one
	}
	return result, err
}
