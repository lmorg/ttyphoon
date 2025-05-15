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

func llmOpenAI() (llms.Model, error) {
	return openai.New(openai.WithModel(Model()))
}

func llmAnthropic() (llms.Model, error) {
	return anthropic.New(anthropic.WithModel(Model()))
}

func runLLM(model llms.Model, meta *Meta, prompt string) (string, error) {
	/*webscraper, err := scraper.New()
	if err != nil {
		return "", err
	}*/

	/*ddg, err := duckduckgo.New(5, fmt.Sprintf("%s/%s", app.Name, app.Version()))
	if err != nil {
		return "", err
	}*/

	agentTools := []tools.Tool{
		LocalFile{meta: meta},
		//Directory{meta: meta},
		//webscraper,
		//ddg,
	}

	errHandler := agents.NewParserErrorHandler(func(s string) string { return "TODO" })

	agent := agents.NewOneShotAgent(model, agentTools)
	executor := agents.NewExecutor(agent, agents.WithMaxIterations(3), agents.WithParserErrorHandler(errHandler))

	return chains.Run(context.Background(), executor, prompt, chains.WithTemperature(1))
	//return := model.Call(context.Background(), prompt, llms.WithTemperature(1))
}
