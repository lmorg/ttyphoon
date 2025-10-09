package agent

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

func llmOpenAI(meta *Meta) (llms.Model, error) {
	return openai.New(openai.WithModel(meta.ModelName()))
}

func llmAnthropic(meta *Meta) (llms.Model, error) {
	return anthropic.New(anthropic.WithModel(meta.ModelName()))
}

func llmOllama(meta *Meta) (llms.Model, error) {
	return ollama.New(ollama.WithModel(meta.ModelName()))
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
	case LLM_OLLAMA:
		model, err = llmOllama(meta)
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

// RunLLM calls the LLM with the prompt string.
// Use `ai` package to create specific prompts.
func (meta *Meta) RunLLM(prompt string, sticky types.Notification) (result string, err error) {
	if debug.Trace {
		log.Printf("RunLLM prompt:\n%s", prompt)
		defer func() {
			log.Printf("RunLLM result:\n%s", result)
			log.Printf("RunLLM error: %v", err)
		}()
	}

	if meta.fnCancel != nil {
		meta.fnCancel()
		meta.fnCancel = nil
	}

	if meta.executor == nil {
		err := initLLM(meta)
		if err != nil {
			return "", err
		}
	}

	var ctx context.Context
	ctx, meta.fnCancel = context.WithTimeout(context.Background(), 5*time.Minute)
	sticky.UpdateCanceller(meta.fnCancel)

	result, err = chains.Run(ctx, meta.executor, prompt, chains.WithTemperature(1))

	switch {
	case err == nil:
		return result, nil

	case strings.HasPrefix(err.Error(), agents.ErrUnableToParseOutput.Error()):
		log.Println(err)
		return err.Error()[len(agents.ErrUnableToParseOutput.Error())+2:], nil // bit of a kludge this one

	case err == agents.ErrNotFinished:
		return "", err //meta.oneShot(prompt, sticky)

	default:
		return result, err
	}
}

/*
func (meta *Meta) oneShot(prompt string, sticky types.Notification) (result string, err error) {

}
*/
