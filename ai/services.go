package ai

const (
	LLM_OPENAI    = "ChatGPT"
	LLM_ANTHROPIC = "Claude"
)

var services = []string{
	LLM_ANTHROPIC,
	LLM_OPENAI,
}

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

func (meta *AgentMeta) ServiceName() string {
	return services[meta.service]
}

func (meta *AgentMeta) ServiceNext() {
	meta.executor = nil
	meta.service++
	if meta.service >= len(services) {
		meta.service = 0
	}
}

func (meta *AgentMeta) ModelName() string {
	return meta.model[meta.ServiceName()]
}

func (meta *AgentMeta) ModelNext() {
	meta.executor = nil
	meta.model[meta.ServiceName()] = meta._modelNext()
}

func (meta *AgentMeta) _modelNext() string {
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
