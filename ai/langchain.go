package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lmorg/mxtty/app"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/tools/duckduckgo"
	"github.com/tmc/langchaingo/tools/scraper"
)

func llmOpenAI(meta *AgentMeta) (llms.Model, error) {
	return openai.New(openai.WithModel(meta.ModelName()))
}

func llmAnthropic(meta *AgentMeta) (llms.Model, error) {
	return anthropic.New(anthropic.WithModel(meta.ModelName()))
}

func (meta *AgentMeta) initLLM() error {
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

	webscraper, err := scraper.New()
	if err != nil {
		return err
	}

	ddg, err := duckduckgo.New(5, fmt.Sprintf("%s/%s", app.Name, app.Version()))
	if err != nil {
		return err
	}

	//history := memory.NewSimple()

	agentTools := []tools.Tool{
		LocalFile{meta: meta},
		Directory{meta: meta},
		ChatHistoryDetail{meta: meta},
		Wrapper{meta, webscraper},
		Wrapper{meta, ddg},
		//Write{meta: meta},
	}

	//errHandler := agents.NewParserErrorHandler(func(s string) string { return "TODO" })

	agent := agents.NewOneShotAgent(model, agentTools, agents.WithMaxIterations(50)) //, agents.WithParserErrorHandler(errHandler))
	//agent := agents.NewConversationalAgent(model, agentTools, agents.WithMaxIterations(3))
	meta.executor = agents.NewExecutor(agent)

	return nil
}

const _ERR_UNABLE_TO_PARSE_AGENT_OUTPUT = "unable to parse agent output: "

func (meta *AgentMeta) runLLM(prompt string) (string, error) {
	if meta.executor == nil {
		err := meta.initLLM()
		if err != nil {
			return "", nil
		}
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)
	result, err := chains.Run(ctx, meta.executor, prompt, chains.WithTemperature(1))
	if err == nil {
		return result, nil
	}

	if strings.HasPrefix(err.Error(), _ERR_UNABLE_TO_PARSE_AGENT_OUTPUT) {
		return err.Error()[len(_ERR_UNABLE_TO_PARSE_AGENT_OUTPUT):], nil
	}
	return result, err
}
