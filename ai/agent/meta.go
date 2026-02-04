package agent

import (
	"context"
	"fmt"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/tmc/langchaingo/agents"
)

type Meta struct {
	executor      *agents.Executor
	service       int
	model         map[string]string
	maxIterations int
	History       HistoryT

	Term     types.Term
	Renderer types.Renderer

	CmdLine          string
	Pwd              string
	OutputBlock      string
	InsertAfterRowId uint64

	fnCancel context.CancelFunc

	_mcpServers map[string]client
	_tools      []Tool
}

func NewAgentMeta() *Meta {
	refreshServiceList()

	meta := &Meta{
		model:         map[string]string{},
		_mcpServers:   make(map[string]client),
		maxIterations: config.Config.Ai.MaxIterations,
	}

	setDefaultModels(meta)

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

func (meta *Meta) MaxIterations() int {
	return meta.maxIterations
}

func (meta *Meta) Reload() {
	meta.executor = nil
}

func (meta *Meta) McpServerAdd(server string, client client) {
	meta._mcpServers[server] = client
}

func (meta *Meta) McpServerExists(server string) bool {
	_, ok := meta._mcpServers[server]
	return ok
}

func Close(tileId string) {
	meta, ok := allTheAgents[tileId]
	if !ok {
		return
	}

	for server, client := range meta._mcpServers {
		err := client.Close()
		if err != nil {
			if meta.Renderer != nil {
				meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Error closing MCP Server '%s': %v", server, err))
			}
		} else {
			if meta.Renderer != nil {
				meta.Renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("Closing MCP Server '%s'", server))
			}
		}
	}
}
