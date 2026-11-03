package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/notes"
)

type Agent struct {
	runtime          agentRuntime
	serviceName      string
	modelName        string
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
	mcpMu             sync.RWMutex
	mcpLoads          map[string]context.CancelFunc
	mcpClosed         bool
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

func (agt *Agent) McpServerAdd(server, source string, connection client) {
	agt.mcpMu.Lock()
	defer agt.mcpMu.Unlock()
	if agt.mcpClosed {
		return
	}
	if agt._mcpServers == nil {
		agt._mcpServers = make(map[string]client)
		agt._mcpServerSources = make(map[string]string)
	}
	if _, exists := agt._mcpServers[server]; exists {
		return
	}
	agt._mcpServers[server] = connection
	agt._mcpServerSources[server] = source
}

func (agt *Agent) McpServerExists(server string) bool {
	agt.mcpMu.RLock()
	defer agt.mcpMu.RUnlock()
	_, ok := agt._mcpServers[server]
	return ok
}

func (agt *Agent) McpServerSource(server string) string {
	agt.mcpMu.RLock()
	defer agt.mcpMu.RUnlock()
	return agt._mcpServerSources[server]
}

func (agt *Agent) McpServerRemove(server string) {
	agt.mcpMu.Lock()
	defer agt.mcpMu.Unlock()
	delete(agt._mcpServers, server)
	delete(agt._mcpServerSources, server)
}

func (agt *Agent) Renderer() types.Renderer { return agt.renderer }
func (agt *Agent) Term() types.Term         { return agt.term }
func (agt *Agent) ProjectRoot() string {
	if agt == nil || agt.term == nil {
		return ""
	}
	tile := agt.term.Tile()
	if tile == nil {
		return ""
	}
	pwd := strings.TrimSpace(tile.Pwd())
	if pwd == "" {
		return ""
	}
	return notes.DirProjectRoot(pwd)
}

type toolPermission uint8

const (
	toolPermissionUndecided toolPermission = iota
	toolPermissionAllowedPrompt
	toolPermissionDeniedPrompt
)

// ErrToolPermissionRefused marks a refusal the LLM should be told about rather
// than one that should abort the agent run.
var ErrToolPermissionRefused = errors.New("tool call refused")

type toolPermissionState struct {
	decision  toolPermission
	reason    string
	prompting bool
	wait      chan struct{}
}

// Permissions are scoped to a single user prompt, so this must only be called
// when a new prompt starts — not per continuation.
func (agt *Agent) ResetToolPermissions() {
	agt.toolPermissionMu.Lock()
	defer agt.toolPermissionMu.Unlock()
	agt.toolPermissions = make(map[string]*toolPermissionState)
}

// writePermissionRequests tracks pending in-panel access-request prompts so the
// frontend can resolve them by request ID once the user clicks an option.
var writePermissionRequests = struct {
	mu sync.Mutex
	m  map[string]writePermissionRequestT
}{m: map[string]writePermissionRequestT{}}

type writePermissionDecisionT struct {
	decision string
	reason   string
}

type writePermissionRequestT struct {
	workspace string
	toolName  string
	response  chan writePermissionDecisionT
}

var writePermissionSeq int64

func newWritePermissionRequest(workspace, toolName string) (string, chan writePermissionDecisionT) {
	id := fmt.Sprintf("wp%d", atomic.AddInt64(&writePermissionSeq, 1))
	ch := make(chan writePermissionDecisionT, 1)
	writePermissionRequests.mu.Lock()
	writePermissionRequests.m[id] = writePermissionRequestT{workspace: workspace, toolName: toolName, response: ch}
	writePermissionRequests.mu.Unlock()
	return id, ch
}

func takeWritePermissionRequest(id string) (writePermissionRequestT, bool) {
	writePermissionRequests.mu.Lock()
	defer writePermissionRequests.mu.Unlock()
	request, ok := writePermissionRequests.m[id]
	if ok {
		delete(writePermissionRequests.m, id)
	}
	return request, ok
}

// ResolveWritePermissionRequest is called by the frontend when the user clicks
// one of the access-request options rendered in the AI panel output.
func ResolveWritePermissionRequest(id, decision, reason string) error {
	request, ok := takeWritePermissionRequest(id)
	if !ok {
		return fmt.Errorf("permission request %q not found or already resolved", id)
	}
	request.response <- writePermissionDecisionT{decision: decision, reason: strings.TrimSpace(reason)}
	return nil
}

func WritePermissionRequestDetails(id string) (workspace, toolName string, ok bool) {
	writePermissionRequests.mu.Lock()
	defer writePermissionRequests.mu.Unlock()
	request, ok := writePermissionRequests.m[id]
	return request.workspace, request.toolName, ok
}

func formatToolPermissionRequestMarkdown(toolName, requestID, description string) string {
	descriptionBlock := ""
	if description != "" {
		descriptionBlock = description + "\n\n"
	}

	return fmt.Sprintf(
		"\n\n**Access requested for tool %s**\n\n%s"+
			"- [Allow this invocation](ttyphoon://ai-tool-permission?request=%s&decision=allow-once)\n"+
			"- [Allow for this prompt](ttyphoon://ai-tool-permission?request=%s&decision=allow-prompt)\n"+
			"- [Deny this invocation](ttyphoon://ai-tool-permission?request=%s&decision=deny-once)\n"+
			"- [Deny for this prompt](ttyphoon://ai-tool-permission?request=%s&decision=deny-prompt)\n\n",
		toolName, descriptionBlock, requestID, requestID, requestID, requestID,
	)
}

func (agt *Agent) RequestToolPermission(ctx context.Context, toolName string) error {
	return agt.requestToolPermission(ctx, toolName, nil)
}

func (agt *Agent) requestToolPermission(ctx context.Context, toolName string, describeCall func() string) error {
	state := agt.ToolState(toolName)
	if state == ToolStateDisabled {
		return fmt.Errorf("%w: tool %q is disabled", ErrToolPermissionRefused, toolName)
	}
	if state == ToolStateAlways {
		return nil
	}
	if state == ToolStateSession {
		return nil
	}
	if state == ToolStateDenied {
		return fmt.Errorf("%w: tool %q is denied", ErrToolPermissionRefused, toolName)
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
		case toolPermissionAllowedPrompt:
			agt.toolPermissionMu.Unlock()
			return nil
		case toolPermissionDeniedPrompt:
			reason := permission.reason
			agt.toolPermissionMu.Unlock()
			return toolPermissionRefusal(toolName, reason)
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

		reqID, decisionCh := newWritePermissionRequest(agt.Workspace(), toolName)
		description := ""
		if describeCall != nil {
			description = describeCall()
		}
		permissionMarkdown := formatToolPermissionRequestMarkdown(toolName, reqID, description)
		emitAIStreamToolBlock(ctx, sessiondb.StreamBlockQuestion, permissionMarkdown)

		var decision writePermissionDecisionT
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

		// Once-decisions are deliberately not stored, so a concurrent or later
		// invocation of the same tool prompts again.
		var allowed bool
		agt.toolPermissionMu.Lock()
		refusalReason := decision.reason
		switch decision.decision {
		case "allow-prompt":
			permission.decision = toolPermissionAllowedPrompt
			allowed = true
		case "deny-prompt":
			permission.decision = toolPermissionDeniedPrompt
			permission.reason = refusalReason
		case "allow-once":
			allowed = true
		}
		permission.prompting = false
		close(wait)
		agt.toolPermissionMu.Unlock()

		if allowed {
			return nil
		}
		return toolPermissionRefusal(toolName, refusalReason)
	}
}

func toolPermissionRefusal(toolName, reason string) error {
	if reason == "" {
		return fmt.Errorf("%w: user denied tool %q", ErrToolPermissionRefused, toolName)
	}
	return fmt.Errorf("%w: user denied tool %q: %s", ErrToolPermissionRefused, toolName, reason)
}

func Close(tileId string) {
	agent, ok := allTheAgents.Get(tileId)
	if !ok {
		return
	}

	agent.mcpMu.Lock()
	agent.mcpClosed = true
	for _, cancel := range agent.mcpLoads {
		cancel()
	}
	connections := agent._mcpServers
	agent._mcpServers = make(map[string]client)
	agent._mcpServerSources = make(map[string]string)
	agent.toolMu.Lock()
	nativeTools := agent._tools[:0]
	for _, tool := range agent._tools {
		if _, isMcp := tool.(*mcpTool); !isMcp {
			nativeTools = append(nativeTools, tool)
		}
	}
	agent._tools = nativeTools
	agent.toolMu.Unlock()
	agent.mcpMu.Unlock()
	for server, connection := range connections {
		err := connection.Close()
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
