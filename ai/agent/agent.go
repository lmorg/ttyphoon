package agent

import (
	"context"
	"fmt"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/tmc/langchaingo/agents"
)

type Agent struct {
	executor      *agents.Executor
	serviceName   string
	modelName     string
	maxIterations int
	History       HistoryT

	term     types.Term
	renderer types.Renderer

	Meta *Meta

	fnCancel context.CancelFunc

	_mcpServers map[string]client
	_tools      []Tool
}

type Meta struct {
	CmdLine     string
	Pwd         string
	OutputBlock string
}

var allTheAgents = map[string]*Agent{}

func New(renderer types.Renderer, tile types.Tile) {
	agent := &Agent{
		_mcpServers:   make(map[string]client),
		maxIterations: config.Config.Ai.MaxIterations,
		term:          tile.GetTerm(),
		renderer:      renderer,
	}

	agent.setDefaultModels()
	agent.toolsInit()

	allTheAgents[tile.Id()] = agent
}

func Get(tileId string) *Agent {
	agent, ok := allTheAgents[tileId]
	if !ok {
		panic("agent not initialized")
	}

	return agent
}

func (agent *Agent) MaxIterations() int {
	return agent.maxIterations
}

func (agent *Agent) Reload() {
	agent.executor = nil
}

func (agent *Agent) McpServerAdd(server string, client client) {
	agent._mcpServers[server] = client
}

func (agent *Agent) McpServerExists(server string) bool {
	_, ok := agent._mcpServers[server]
	return ok
}

func (agent *Agent) Renderer() types.Renderer { return agent.renderer }
func (agent *Agent) Term() types.Term         { return agent.term }

func Close(tileId string) {
	agent, ok := allTheAgents[tileId]
	if !ok {
		return
	}

	for server, client := range agent._mcpServers {
		err := client.Close()
		if err != nil {
			if agent.renderer != nil {
				agent.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Error closing MCP Server '%s': %v", server, err))
			}
		} else {
			if agent.renderer != nil {
				agent.renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("Closing MCP Server '%s'", server))
			}
		}
	}
}
