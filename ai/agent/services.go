package agent

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

const (
	LLM_OPENAI    = "openai"
	LLM_ANTHROPIC = "anthropic"
	LLM_OLLAMA    = "ollama"
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

func (agt *Agent) SummariseModelName() string {
	service := findService(agt.serviceName)
	if service == nil {
		return agt.modelName
	}
	return service.SummariseModel()
}

func (agt *Agent) ProviderName() string {
	service := findService(agt.serviceName)
	if service == nil || service.Provider == "" {
		return agt.serviceName
	}
	return service.Provider
}

func (agt *Agent) serviceEnv(name string) string {
	service := findService(agt.serviceName)
	if service != nil {
		if value, ok := service.Env[name]; ok {
			return value
		}
	}
	return ""
}

func (agt *Agent) EnvironmentValue(name string) string {
	if value := agt.serviceEnv(name); value != "" {
		return value
	}
	return os.Getenv(name)
}

func (agt *Agent) ImageGenerationEnvironmentValue(name string) string {
	service := findService(agt.serviceName)
	if service != nil && service.ImageGenService != "" {
		imageService := findService(service.ImageGenService)
		if imageService != nil {
			if value := imageService.Env[name]; value != "" {
				return value
			}
		}
		return os.Getenv(name)
	}

	return agt.EnvironmentValue(name)
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

	for _, service := range config.Config.Ai.Services() {
		for modelId, modelName := range service.Models {
			modelXRef = append(modelXRef, ServiceModelIndexT{
				service: service.Label,
				modelId: modelId,
			})
			labels = append(labels, modelSelectionLabel(service.Label, modelName))
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

	service := findService(serviceName)
	if service == nil {
		return fmt.Errorf("unknown model service: %q", serviceName)
	}

	if !slices.Contains(service.Models, modelName) {
		return fmt.Errorf("unknown model for service %q: %q", serviceName, modelName)
	}

	agt.serviceName = serviceName
	agt.modelName = modelName
	agt.Reload()

	return nil
}

func (agt *Agent) SwitchServiceModel(modelXRef []ServiceModelIndexT, i int) {
	agt.serviceName = modelXRef[i].service
	service := findService(modelXRef[i].service)
	agt.modelName = service.Models[modelXRef[i].modelId]
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
	/*go func() {
		ollama := ollamaModels()
		if len(ollama) > 0 {
			models[LLM_OLLAMA] = ollamaModels()
		}
	}()*/
}

func findService(serviceName string) *config.AIServiceT {
	for _, service := range config.Config.Ai.Services() {
		if service.Label == serviceName {
			return service
		}
	}

	panic(fmt.Sprintf("service not found: '%s'", serviceName))
}
