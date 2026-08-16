package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

type Agent struct {
	runtime       agentRuntime
	serviceName   string
	modelName     string
	maxIterations int

	term     types.Term
	renderer types.Renderer

	Meta *aitypes.Meta

	fnCancel context.CancelFunc

	_mcpServers       map[string]client
	_mcpServerSources map[string]string
	_tools            []aitypes.Tool
}

type allTheAgentsT struct {
	_map   map[string]*Agent
	_mutex sync.Mutex
}

func (ata *allTheAgentsT) Get(key string) (*Agent, bool) {
	ata._mutex.Lock()
	defer ata._mutex.Unlock()
	agent, ok := ata._map[key]
	return agent, ok
}

func (ata *allTheAgentsT) Set(key string, agent *Agent) {
	ata._mutex.Lock()
	defer ata._mutex.Unlock()
	ata._map[key] = agent
}

func (ata *allTheAgentsT) Delete(key string) {
	ata._mutex.Lock()
	defer ata._mutex.Unlock()
	delete(ata._map, key)
}

var allTheAgents = allTheAgentsT{_map: map[string]*Agent{}}

func New(renderer types.Renderer, tile types.Tile) {
	agent := &Agent{
		_mcpServers:       make(map[string]client),
		_mcpServerSources: make(map[string]string),
		maxIterations:     config.Config.Ai.MaxIterations,
		term:              tile.GetTerm(),
		renderer:          renderer,
	}

	agent.setDefaultModels()
	agent.toolsInit()

	allTheAgents.Set(tile.Id(), agent)
}

func Get(tileId string) *Agent {
	agt, ok := allTheAgents.Get(tileId)
	if !ok {
		panic("agent not initialized")
	}

	return agt
}

func TryGet(tileId string) (*Agent, bool) {
	return allTheAgents.Get(tileId)
}

func (agt *Agent) MaxIterations() int {
	return agt.maxIterations
}

func (agt *Agent) Reload() {
	if agt.runtime != nil {
		agt.runtime.Reset()
	}
	agt.runtime = nil
}

func (agt *Agent) McpServerAdd(server, source string, client client) {
	agt._mcpServers[server] = client
	agt._mcpServerSources[server] = source
}

func (agt *Agent) McpServerExists(server string) bool {
	_, ok := agt._mcpServers[server]
	return ok
}

func (agt *Agent) McpServerSource(server string) string {
	return agt._mcpServerSources[server]
}

func (agt *Agent) McpServerRemove(server string) {
	delete(agt._mcpServers, server)
	delete(agt._mcpServerSources, server)
}

func (agt *Agent) Renderer() types.Renderer { return agt.renderer }
func (agt *Agent) Term() types.Term         { return agt.term }

func Close(tileId string) {
	agent, ok := allTheAgents.Get(tileId)
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

func (agt *Agent) GetMeta() *aitypes.Meta {
	return agt.Meta
}

func (agt *Agent) Workspace() string {
	if agt == nil {
		return ""
	}
	term := agt.Term()
	if term == nil {
		return ""
	}

	tile := term.Tile()
	if tile == nil {
		return ""
	}

	return tile.GroupName()
}

func (agt *Agent) IsWorkspaceActive() bool {
	if agt == nil {
		return false
	}
	renderer := agt.Renderer()
	if renderer == nil {
		return false
	}

	activeTile := renderer.ActiveTile()
	if activeTile == nil {
		return false
	}

	return activeTile.GroupName() == agt.Workspace()
}
