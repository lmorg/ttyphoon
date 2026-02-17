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

func (meta *Meta) ServiceName() string {
	return meta.serviceName
}

func (meta *Meta) ModelName() string {
	return meta.modelName
}

type selectServiceMenuItemT struct {
	service string
	modelId int
}

func (meta *Meta) SelectServiceModel(returnFn func()) {
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
		meta.serviceName = modelXRef[i].service
		meta.modelName = models[modelXRef[i].service][modelXRef[i].modelId]
		meta.Reload()
		if returnFn != nil {
			returnFn()
		}
	}

	meta.renderer.DisplayMenu("Select model to use", labels, nil, selectFn, nil)
}

func refreshServiceList() {
	models = config.Config.Ai.AvailableModels
	models[LLM_OLLAMA] = ollamaModels()
}

func setDefaultModels(meta *Meta) {
	if len(models[config.Config.Ai.DefaultService]) != 0 {
		meta.serviceName = config.Config.Ai.DefaultService
	} else {
		for meta.serviceName = range models {
			// just get the first service, whatever that service might be
			break
		}
	}

	if config.Config.Ai.DefaultModels[meta.serviceName] != "" {
		meta.modelName = config.Config.Ai.DefaultModels[meta.serviceName]
	} else {
		meta.modelName = models[meta.serviceName][0]
	}
}
