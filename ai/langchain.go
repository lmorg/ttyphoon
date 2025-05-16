package ai

import (
	"context"
	"fmt"
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

func llmOpenAI() (llms.Model, error) {
	return openai.New(openai.WithModel(Model()))
}

func llmAnthropic() (llms.Model, error) {
	return anthropic.New(anthropic.WithModel(Model()))
}

func runLLM(model llms.Model, meta *Meta, prompt string) (string, error) {
	webscraper, err := scraper.New()
	if err != nil {
		return "", err
	}

	ddg, err := duckduckgo.New(5, fmt.Sprintf("%s/%s", app.Name, app.Version()))
	if err != nil {
		return "", err
	}

	agentTools := []tools.Tool{
		LocalFile{meta: meta},
		Directory{meta: meta},
		Wrapper{meta, webscraper},
		Wrapper{meta, ddg},
	}

	errHandler := agents.NewParserErrorHandler(func(s string) string { return "TODO" })

	//agent := agents.NewOneShotAgent(model, agentTools, agents.WithMaxIterations(3), agents.WithParserErrorHandler(errHandler))
	agent := agents.NewConversationalAgent(model, agentTools, agents.WithMaxIterations(3), agents.WithParserErrorHandler(errHandler))
	executor := agents.NewExecutor(agent)

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)
	return chains.Run(ctx, executor, prompt, chains.WithTemperature(1))
	//return := model.Call(context.Background(), prompt, llms.WithTemperature(1))
}

/*func runLLM(model llms.Model, meta *Meta, prompt string) (string, error) {
	webscraper, err := scraper.New()
	if err != nil {
		return "", err
	}

	ddg, err := duckduckgo.New(5, fmt.Sprintf("%s/%s", app.Name, app.Version()))
	if err != nil {
		return "", err
	}

	agentTools := []tools.Tool{
		LocalFile{meta: meta},
		Directory{meta: meta},
		Wrapper{meta, webscraper},
		Wrapper{meta, ddg},
	}

	errHandler := agents.NewParserErrorHandler(func(s string) string { return "TODO" })

	agent := agents.NewOneShotAgent(model, agentTools, agents.WithMaxIterations(3), agents.WithParserErrorHandler(errHandler))
	executor := agents.NewExecutor(agent)

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)
	return chains.Run(ctx, executor, prompt, chains.WithTemperature(1))
	//return := model.Call(context.Background(), prompt, llms.WithTemperature(1))
}
*/
