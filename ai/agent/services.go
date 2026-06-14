package agent

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
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

type ServiceModelIndexT struct {
	service string
	modelId int
}

func modelSelectionLabel(serviceName, modelName string) string {
	return fmt.Sprintf("%s: %s", serviceName, modelName)
}

func (agent *Agent) CurrentModelLabel() string {
	return modelSelectionLabel(agent.serviceName, agent.modelName)
}

// ListModelsInputVariable returns a pre-configured InputBox variable definition
// that can be used directly in InputBoxWT options.
func (agent *Agent) ListModelsInputVariable() types.InputBoxWTVariables {
	_, labels := agent.ListModels()

	return types.InputBoxWTVariables{
		Name:        "model",
		Label:       "Model",
		Description: "AI LLM model to use for query",
		Default:     agent.CurrentModelLabel(),
		Options:     labels,
		Type:        "list",
	}
}

func (agent *Agent) ListModels() ([]ServiceModelIndexT, []string) {
	var (
		modelXRef []ServiceModelIndexT
		labels    []string
	)

	for serviceName := range models {
		for modelId, modelName := range models[serviceName] {
			modelXRef = append(modelXRef, ServiceModelIndexT{
				service: serviceName,
				modelId: modelId,
			})
			labels = append(labels, modelSelectionLabel(serviceName, modelName))
		}
	}

	return modelXRef, labels
}

// SetServiceModelFromSelection sets the active service/model from a list label
// formatted as "<Service>: <Model>".
func (agent *Agent) SetServiceModelFromSelection(selection string) error {
	parts := strings.SplitN(strings.TrimSpace(selection), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid model selection format: %q", selection)
	}

	serviceName := strings.TrimSpace(parts[0])
	modelName := strings.TrimSpace(parts[1])
	if serviceName == "" || modelName == "" {
		return fmt.Errorf("invalid model selection format: %q", selection)
	}

	modelsByService, ok := models[serviceName]
	if !ok {
		return fmt.Errorf("unknown model service: %q", serviceName)
	}

	if !slices.Contains(modelsByService, modelName) {
		return fmt.Errorf("unknown model for service %q: %q", serviceName, modelName)
	}

	agent.serviceName = serviceName
	agent.modelName = modelName
	agent.Reload()

	return nil
}

func (agent *Agent) SwitchServiceModel(modelXRef []ServiceModelIndexT, i int) {
	agent.serviceName = modelXRef[i].service
	agent.modelName = models[modelXRef[i].service][modelXRef[i].modelId]
	agent.Reload()
}

func (agent *Agent) SelectServiceModel(returnFn func()) {
	modelXRef, labels := agent.ListModels()

	selectFn := func(i int) {
		agent.SwitchServiceModel(modelXRef, i)
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
