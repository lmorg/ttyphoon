package agent

import (
	"fmt"

	"github.com/lmorg/ttyphoon/config"
)

const (
	LLM_OPENAI    = "ChatGPT"
	LLM_ANTHROPIC = "Claude"
	LLM_OLLAMA    = "Ollama"
)

var (
	models map[string][]string
)

func init() {
	refreshServiceList()
}

func (agent *Agent) ServiceName() string {
	return agent.serviceName
}

func (agent *Agent) ModelName() string {
	return agent.modelName
}

type selectServiceMenuItemT struct {
	service string
	modelId int
}

func (agent *Agent) SelectServiceModel(returnFn func()) {
	var (
		modelXRef []selectServiceMenuItemT
		labels    []string
	)

	for serviceName := range models {
		for modelId, modelName := range models[serviceName] {
			modelXRef = append(modelXRef, selectServiceMenuItemT{
				service: serviceName,
				modelId: modelId,
			})
			labels = append(labels, fmt.Sprintf("%s: %s", serviceName, modelName))
		}
	}

	selectFn := func(i int) {
		agent.serviceName = modelXRef[i].service
		agent.modelName = models[modelXRef[i].service][modelXRef[i].modelId]
		agent.Reload()
		if returnFn != nil {
			returnFn()
		}
	}

	agent.renderer.DisplayMenu("Select model to use", labels, nil, selectFn, nil)
}

func refreshServiceList() {
	models = config.Config.Ai.AvailableModels
	go func() {
		ollama := ollamaModels()
		if len(ollama) > 0 {
			models[LLM_OLLAMA] = ollamaModels()
		}
	}()
}

func (agent *Agent) setDefaultModels() {
	if len(models[config.Config.Ai.DefaultService]) != 0 {
		agent.serviceName = config.Config.Ai.DefaultService
	} else {
		for agent.serviceName = range models {
			// just get the first service, whatever that service might be
			break
		}
	}

	if config.Config.Ai.DefaultModels[agent.serviceName] != "" {
		agent.modelName = config.Config.Ai.DefaultModels[agent.serviceName]
	} else {
		agent.modelName = models[agent.serviceName][0]
	}
}
