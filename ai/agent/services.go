package agent

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

const (
	LLM_OPENAI    = "OpenAI"
	LLM_ANTHROPIC = "Anthropic"
	LLM_OLLAMA    = "Ollama"
)

var (
	models map[string][]string
)

func init() {
	refreshServiceList()
}

func (agt *Agent) ServiceName() string {
	return agt.serviceName
}

func (agt *Agent) ModelName() string {
	return agt.modelName
}

type ServiceModelIndexT struct {
	service string
	modelId int
}

func modelSelectionLabel(serviceName, modelName string) string {
	return fmt.Sprintf("%s: %s", serviceName, modelName)
}

func (agt *Agent) CurrentModelLabel() string {
	return modelSelectionLabel(agt.serviceName, agt.modelName)
}

// ListModelsInputVariable returns a pre-configured InputBox variable definition
// that can be used directly in InputBoxWT options.
func (agt *Agent) ListModelsInputVariable() types.InputBoxWTVariables {
	_, labels := agt.ListModels()

	return types.InputBoxWTVariables{
		Name:        "model",
		Label:       "Model",
		Description: "AI LLM model to use for query",
		Default:     agt.CurrentModelLabel(),
		Options:     labels,
		Type:        "list",
	}
}
func (agt *Agent) SetModelFromInputVariable(variables map[string]any) {
	if raw, ok := variables["model"]; ok {
		selectedModel := strings.TrimSpace(fmt.Sprint(raw))
		if selectedModel != "" {
			err := agt.SetServiceModelFromSelection(selectedModel)
			if err != nil {
				agt.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
				return
			}
		}
	}
}

func (agt *Agent) ListModels() ([]ServiceModelIndexT, []string) {
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
func (agt *Agent) SetServiceModelFromSelection(selection string) error {
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

	agt.serviceName = serviceName
	agt.modelName = modelName
	agt.Reload()

	return nil
}

func (agt *Agent) SwitchServiceModel(modelXRef []ServiceModelIndexT, i int) {
	agt.serviceName = modelXRef[i].service
	agt.modelName = models[modelXRef[i].service][modelXRef[i].modelId]
	agt.Reload()
}

func (agt *Agent) SelectServiceModel(returnFn func()) {
	modelXRef, labels := agt.ListModels()

	selectFn := func(i int) {
		agt.SwitchServiceModel(modelXRef, i)
		if returnFn != nil {
			returnFn()
		}
	}

	agt.renderer.DisplayMenu("Select model to use", labels, nil, selectFn, nil)
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

func (agt *Agent) setDefaultModels() {
	if len(models[config.Config.Ai.DefaultService]) != 0 {
		agt.serviceName = config.Config.Ai.DefaultService
	} else {
		for agt.serviceName = range models {
			// just get the first service, whatever that service might be
			break
		}
	}

	if config.Config.Ai.DefaultModels[agt.serviceName] != "" {
		agt.modelName = config.Config.Ai.DefaultModels[agt.serviceName]
	} else {
		agt.modelName = models[agt.serviceName][0]
	}
}
