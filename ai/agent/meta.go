package agent

import (
	"github.com/lmorg/murex/utils/lists"
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

	CmdLine       string
	Pwd           string
	OutputBlock   string
	InsertAtRowId uint64

	_mcpServers []string
	_tools      []Tool
}

func NewAgentMeta() *Meta {
	meta := &Meta{
		model: map[string]string{
			LLM_OPENAI:    models[LLM_OPENAI][0],
			LLM_ANTHROPIC: models[LLM_ANTHROPIC][1],
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

func (meta *Meta) Reload() {
	meta.executor = nil
}

func (meta *Meta) McpServerAdd(server string) {
	meta._mcpServers = append(meta._mcpServers, server)
}

func (meta *Meta) McpServerExists(server string) bool {
	return lists.Match(meta._mcpServers, server)
}
