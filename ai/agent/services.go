package agent

import (
	"slices"

	"github.com/lmorg/mxtty/config"
)

const (
	LLM_OPENAI    = "ChatGPT"
	LLM_ANTHROPIC = "Claude"
	LLM_OLLAMA    = "Ollama"
)

var (
	services []string
	models   map[string][]string
)

func (meta *Meta) ServiceName() string {
	return services[meta.service]
}

func (meta *Meta) ServiceNext() {
	refreshServiceList()
	meta.service++
	if meta.service >= len(services) {
		meta.service = 0
	}
	meta.Reload()
}

func (meta *Meta) ModelName() string {
	return meta.model[meta.ServiceName()]
}

func (meta *Meta) ModelNext() {
	refreshServiceList()
	meta.model[meta.ServiceName()] = meta._modelNext()
	meta.Reload()
}

func (meta *Meta) _modelNext() string {
	for i := range models[meta.ServiceName()] {
		if models[meta.ServiceName()][i] != meta.ModelName() {
			continue
		}

		i++
		if i == len(models[meta.ServiceName()]) {
			return models[meta.ServiceName()][0]
		}

		return models[meta.ServiceName()][i]
	}

	return models[meta.ServiceName()][0]
}

func refreshServiceList() {
	models = config.Config.Ai.AvailableModels
	services = []string{}
	for service := range config.Config.Ai.AvailableModels {
		services = append(services, service)
	}
}

func setDefaultModels(meta *Meta) {
	for service, model := range config.Config.Ai.DefaultModels {
		if slices.Contains(models[service], model) {
			meta.model[service] = model
		}
	}

	for i := range services {
		if services[i] == config.Config.Ai.DefaultService {
			meta.service = i
			break
		}
	}
}
