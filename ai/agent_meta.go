package ai

import (
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/agents"
)

type AgentMeta struct {
	executor *agents.Executor
	service  int
	model    map[string]string
	history  historyT

	Term     types.Term
	Renderer types.Renderer

	CmdLine      string
	Pwd          string
	OutputBlock  string
	InsertRowPos int32
}

func NewAgentMeta() *AgentMeta {
	return &AgentMeta{
		model: map[string]string{
			LLM_OPENAI:    "gpt-4.1",
			LLM_ANTHROPIC: "claude-3-5-haiku-latest",
		},
	}
}

var allTheAgents = map[string]*AgentMeta{}

func Agent(tileId string) *AgentMeta {
	meta, ok := allTheAgents[tileId]
	if !ok {
		meta = NewAgentMeta()
	}

	allTheAgents[tileId] = meta
	return meta
}
