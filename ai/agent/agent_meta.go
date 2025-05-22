package agent

import (
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/agents"
)

type Meta struct {
	executor *agents.Executor
	service  int
	model    map[string]string
	History  HistoryT

	Term     types.Term
	Renderer types.Renderer

	CmdLine      string
	Pwd          string
	OutputBlock  string
	InsertRowPos int32

	_tools []Tool
}

func NewAgentMeta() *Meta {
	meta := &Meta{
		model: map[string]string{
			LLM_OPENAI:    "gpt-4.1",
			LLM_ANTHROPIC: "claude-3-5-haiku-latest",
		},
	}

	meta.toolsInit()

	return meta
}

var allTheAgents = map[string]*Meta{}

func Get(tileId string) *Meta {
	meta, ok := allTheAgents[tileId]
	if !ok {
		meta = NewAgentMeta()
	}

	allTheAgents[tileId] = meta
	return meta
}
