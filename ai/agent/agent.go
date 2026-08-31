package agent

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/notes"
)

type Agent struct {
	runtime          agentRuntime
	serviceName      string
	modelName        string
	projectRoot      string
	maxIterations    int
	toolMu           sync.RWMutex
	toolPermissionMu sync.Mutex
	toolPermissions  map[string]*toolPermissionState
	toolStates       map[string]string
	subagentTools    map[string]bool

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
	agt := &Agent{
		_mcpServers:       make(map[string]client),
		_mcpServerSources: make(map[string]string),
		maxIterations:     config.Config.Ai.MaxIterations,
		term:              tile.GetTerm(),
		renderer:          renderer,
		projectRoot:       notes.DirProjectRoot(tile.Pwd()),
		toolStates:        make(map[string]string),
		subagentTools:     make(map[string]bool),
		toolPermissions:   make(map[string]*toolPermissionState),
	}

	//agent.setDefaultModel()
	agt.toolsInit()

	service := config.Config.Ai.Service(config.Config.Ai.DefaultService)
	agt.serviceName = service.Label
	agt.modelName = service.DefaultModel

	allTheAgents.Set(tile.Id(), agt)
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
func (agt *Agent) ProjectRoot() string      { return agt.projectRoot }

type toolPermission uint8

const (
	toolPermissionUndecided toolPermission = iota
	toolPermissionAllowedPrompt
	toolPermissionAllowedSession
	toolPermissionDeniedPrompt
)

type toolPermissionState struct {
	decision  toolPermission
	prompting bool
	wait      chan struct{}
}

func (agt *Agent) ResetToolPermissions() {
	agt.toolPermissionMu.Lock()
	defer agt.toolPermissionMu.Unlock()
	for toolName, state := range agt.toolPermissions {
		if state.decision != toolPermissionAllowedSession {
			delete(agt.toolPermissions, toolName)
		}
	}
}

// writePermissionRequests tracks pending in-panel access-request prompts so the
// frontend can resolve them by request ID once the user clicks an option.
var writePermissionRequests = struct {
	mu sync.Mutex
	m  map[string]chan string
}{m: map[string]chan string{}}

var writePermissionSeq int64

func newWritePermissionRequest() (string, chan string) {
	id := fmt.Sprintf("wp%d", atomic.AddInt64(&writePermissionSeq, 1))
	ch := make(chan string, 1)
	writePermissionRequests.mu.Lock()
	writePermissionRequests.m[id] = ch
	writePermissionRequests.mu.Unlock()
	return id, ch
}

func takeWritePermissionRequest(id string) (chan string, bool) {
	writePermissionRequests.mu.Lock()
	defer writePermissionRequests.mu.Unlock()
	ch, ok := writePermissionRequests.m[id]
	if ok {
		delete(writePermissionRequests.m, id)
	}
	return ch, ok
}

// ResolveWritePermissionRequest is called by the frontend when the user clicks
// one of the access-request options rendered in the AI panel output.
func ResolveWritePermissionRequest(id, decision string) error {
	ch, ok := takeWritePermissionRequest(id)
	if !ok {
		return fmt.Errorf("permission request %q not found or already resolved", id)
	}
	ch <- decision
	return nil
}

func formatToolPermissionRequestMarkdown(toolName, requestID string) string {
	return fmt.Sprintf(
		"\n\n**Access requested for tool %s**\n\n"+
			"- [Allow for this prompt](ttyphoon://ai-tool-permission?request=%s&decision=prompt)\n"+
			"- [Allow for this session](ttyphoon://ai-tool-permission?request=%s&decision=session)\n"+
			"- [Deny](ttyphoon://ai-tool-permission?request=%s&decision=deny)\n\n",
		toolName, requestID, requestID, requestID,
	)
}

func (agt *Agent) RequestToolPermission(ctx context.Context, toolName string) error {
	state := agt.ToolState(toolName)
	if state == ToolStateDisabled {
		return fmt.Errorf("tool %q is disabled", toolName)
	}
	if state == ToolStateAlways {
		return nil
	}
	if state == ToolStateSession {
		return nil
	}
	if state == ToolStateDenied {
		return fmt.Errorf("permission denied for tool %q", toolName)
	}
	for {
		agt.toolPermissionMu.Lock()
		if agt.toolPermissions == nil {
			agt.toolPermissions = make(map[string]*toolPermissionState)
		}
		permission := agt.toolPermissions[toolName]
		if permission == nil {
			permission = &toolPermissionState{}
			agt.toolPermissions[toolName] = permission
		}
		switch permission.decision {
		case toolPermissionAllowedPrompt, toolPermissionAllowedSession:
			agt.toolPermissionMu.Unlock()
			return nil
		case toolPermissionDeniedPrompt:
			agt.toolPermissionMu.Unlock()
			return fmt.Errorf("permission denied for tool %q", toolName)
		}

		if permission.prompting {
			wait := permission.wait
			agt.toolPermissionMu.Unlock()
			select {
			case <-wait:
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		permission.prompting = true
		permission.wait = make(chan struct{})
		wait := permission.wait
		agt.toolPermissionMu.Unlock()

		reqID, decisionCh := newWritePermissionRequest()
		emitAIStreamToolProgress(ctx, formatToolPermissionRequestMarkdown(toolName, reqID))

		var decision string
		select {
		case decision = <-decisionCh:
		case <-ctx.Done():
			takeWritePermissionRequest(reqID)
			agt.toolPermissionMu.Lock()
			permission.prompting = false
			close(wait)
			agt.toolPermissionMu.Unlock()
			return ctx.Err()
		}

		agt.toolPermissionMu.Lock()
		switch decision {
		case "session":
			permission.decision = toolPermissionAllowedSession
		case "prompt":
			permission.decision = toolPermissionAllowedPrompt
		default:
			permission.decision = toolPermissionDeniedPrompt
		}
		permission.prompting = false
		close(wait)
		agt.toolPermissionMu.Unlock()

		select {
		case <-wait:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

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
