package ai

const (
	LLM_OPENAI    = "ChatGPT"
	LLM_ANTHROPIC = "Claude"
)

var services = []string{
	LLM_OPENAI,
	LLM_ANTHROPIC,
}

var service = LLM_ANTHROPIC

var models = map[string][]string{
	LLM_OPENAI: {
		"gpt-4.1",
		"gpt-4",
		"o4-mini",
	},
	LLM_ANTHROPIC: {
		"claude-3-5-haiku-latest",
		"claude-3-7-sonnet-latest",
		"claude-3-opus-latest",
	},
}

var model = map[string]string{
	LLM_OPENAI:    "gpt-4.1",
	LLM_ANTHROPIC: "claude-3-5-haiku-latest",
}

func Service() string {
	return service
}

func NextService() {
	service = nextService()
}

func nextService() string {
	for i := range services {
		if services[i] != service {
			continue
		}

		i++
		if i == len(services) {
			return services[0]
		}

		return services[i]
	}

	return services[0]
}

func Model() string {
	return model[service]
}

func NextModel() {
	model[service] = nextModel()
}

func nextModel() string {
	for i := range models[service] {
		if models[service][i] != model[service] {
			continue
		}

		i++
		if i == len(models[service]) {
			return models[service][0]
		}

		return models[service][i]
	}

	return models[service][0]
}
