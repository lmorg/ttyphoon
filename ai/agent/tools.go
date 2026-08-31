package agent

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/types"
)

var _tools []aitypes.Tool

const (
	ToolStateDisabled = "disabled"
	ToolStateApproval = "approval"
	ToolStatePrompt   = "prompt"
	ToolStateSession  = "session"
	ToolStateDenied   = "denied"
	ToolStateAlways   = "always"
)

func (agent *Agent) toolsInit() {
	if agent.toolStates == nil {
		agent.toolStates = make(map[string]string)
	}
	for i := range _tools {
		newTool, err := _tools[i].New(agent)
		if err != nil {
			agent.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			continue
		}
		agent._tools = append(agent._tools, newTool)
	}
}

func ToolsAdd(t aitypes.Tool) {
	_tools = append(_tools, t)
}

func (agent *Agent) ToolsAdd(t aitypes.Tool) error {
	tool, err := t.New(agent)
	if err != nil {
		return err
	}

	agent.toolMu.Lock()
	agent._tools = append(agent._tools, tool)
	agent.toolMu.Unlock()
	agent.Reload()

	return nil
}

func (agent *Agent) ChooseTools(cancel types.MenuCallbackT) {
	s := make([]string, len(agent._tools))
	for i, tool := range agent._tools {
		s[i] = fmt.Sprintf("%s == %s", tool.Name(), agent.ToolState(tool.Name()))
	}

	fnOk := func(i int) {
		state := agent.ToolState(agent._tools[i].Name())
		if state == ToolStateDisabled {
			state = agent.defaultToolState(agent._tools[i])
		} else {
			state = ToolStateDisabled
		}
		agent.toolStates[agent._tools[i].Name()] = state
		agent.Reload()
		agent.ChooseTools(cancel)
	}

	agent.renderer.DisplayMenu("AI tools", s, nil, fnOk, cancel)
}

func (agent *Agent) ListTools() []map[string]interface{} {
	agent.toolMu.RLock()
	registeredTools := append([]aitypes.Tool(nil), agent._tools...)
	agent.toolMu.RUnlock()
	tools := make([]map[string]interface{}, len(registeredTools))
	for i, tool := range registeredTools {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type":        "string",
					"description": "Raw input string for the tool.",
				},
			},
			"required": []string{"input"},
		}

		if mcp, ok := tool.(*mcpTool); ok && len(mcp.schema) > 0 {
			var parsed any
			if err := json.Unmarshal(mcp.schema, &parsed); err == nil {
				if parsedMap, isMap := parsed.(map[string]any); isMap {
					schema = parsedMap
				}
			}
		}

		schemaJSON, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			schemaJSON = []byte("{}")
		}

		tools[i] = map[string]interface{}{
			"name":            tool.Name(),
			"enabled":         agent.ToolState(tool.Name()) != ToolStateDisabled,
			"state":           agent.ToolState(tool.Name()),
			"allowInSubagent": agent.ToolAllowedInSubagent(tool.Name()),
			"description":     tool.Description(),
			"schema":          string(schemaJSON),
		}
	}
	return tools
}

func (agent *Agent) SetToolEnabled(toolName string, enabled bool) error {
	if enabled {
		return agent.SetToolState(toolName, agent.defaultToolStateByName(toolName))
	}
	return agent.SetToolState(toolName, ToolStateDisabled)
}

func (agent *Agent) ToolAllowedInSubagent(toolName string) bool {
	agent.toolMu.RLock()
	defer agent.toolMu.RUnlock()
	return agent.toolAllowedInSubagentLocked(toolName)
}

func (agent *Agent) toolAllowedInSubagentLocked(toolName string) bool {
	if allowed, ok := agent.subagentTools[toolName]; ok {
		return allowed
	}
	for _, tool := range agent._tools {
		if tool.Name() == toolName {
			return toolDefaultPermissions(tool).Subagents == "allow"
		}
	}
	return false
}

func (agent *Agent) SetToolAllowedInSubagent(toolName string, allowed bool) error {
	agent.toolMu.Lock()
	defer agent.toolMu.Unlock()
	for _, tool := range agent._tools {
		if tool.Name() == toolName {
			if agent.subagentTools == nil {
				agent.subagentTools = make(map[string]bool)
			}
			agent.subagentTools[toolName] = allowed
			agent.Reload()
			return nil
		}
	}
	return fmt.Errorf("tool %q not found", toolName)
}

func (agent *Agent) SubagentToolNames() []string {
	agent.toolMu.RLock()
	defer agent.toolMu.RUnlock()
	names := make([]string, 0)
	for _, tool := range agent._tools {
		if tool.Name() != "subagent" && agent.toolAllowedInSubagentLocked(tool.Name()) && agent.toolStateLocked(tool.Name()) != ToolStateDisabled {
			names = append(names, tool.Name())
		}
	}
	slices.Sort(names)
	return names
}

func (agent *Agent) ToolState(toolName string) string {
	agent.toolMu.RLock()
	defer agent.toolMu.RUnlock()
	return agent.toolStateLocked(toolName)
}

func (agent *Agent) toolStateLocked(toolName string) string {
	if state := agent.toolStates[toolName]; state != "" {
		return state
	}
	for _, tool := range agent._tools {
		if tool.Name() == toolName {
			if !tool.Enabled() {
				return ToolStateDisabled
			}
			return agent.defaultToolState(tool)
		}
	}
	return ToolStateDisabled
}

func (agent *Agent) defaultToolState(tool aitypes.Tool) string {
	switch toolDefaultPermissions(tool).Invocation {
	case "askPermission":
		return ToolStateApproval
	case "alwaysAllow", "":
		return ToolStateAlways
	default:
		return ToolStateDisabled
	}
}

func toolDefaultPermissions(tool aitypes.Tool) aitypes.DefaultPermissions {
	if configured, ok := tool.(interface {
		DefaultPermissions() aitypes.DefaultPermissions
	}); ok {
		return configured.DefaultPermissions()
	}
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "deny"}
}

func (agent *Agent) defaultToolStateByName(toolName string) string {
	agent.toolMu.RLock()
	defer agent.toolMu.RUnlock()
	return agent.defaultToolStateByNameLocked(toolName)
}

func (agent *Agent) defaultToolStateByNameLocked(toolName string) string {
	for _, tool := range agent._tools {
		if tool.Name() == toolName {
			return agent.defaultToolState(tool)
		}
	}
	return ToolStateDisabled
}

func (agent *Agent) SetToolState(toolName, state string) error {
	if state != ToolStateDisabled && state != ToolStateApproval && state != ToolStatePrompt && state != ToolStateSession && state != ToolStateDenied && state != ToolStateAlways {
		return fmt.Errorf("invalid tool state %q", state)
	}
	agent.toolMu.Lock()
	defer agent.toolMu.Unlock()
	for _, tool := range agent._tools {
		if tool.Name() == toolName {
			agent.toolStates[toolName] = state
			agent.Reload()
			return nil
		}
	}
	return fmt.Errorf("tool %q not found", toolName)
}

func (agent *Agent) ShowToolStateMenu(toolName string, x, y int, changed func(string)) {
	current := agent.ToolState(toolName)
	states := []struct {
		value string
		label string
	}{
		{ToolStateAlways, "Enabled: always allow"},
		{ToolStateSession, "Enabled: current session"},
		{ToolStateApproval, "Enabled: ask for permission"},
	}

	menu := agent.renderer.NewContextMenu()
	for _, option := range states {
		option := option
		item := types.MenuItem{Title: option.label, Fn: func() {
			if err := agent.SetToolState(toolName, option.value); err == nil && changed != nil {
				changed(option.value)
			}
		}}
		if option.value == current {
			item.Icon = 0xf00c
		}
		menu.Append(item)
	}
	menu.Append(types.MenuItem{Title: types.MENU_SEPARATOR})
	menu.Append(types.MenuItem{Title: "Disabled", Icon: func() rune {
		if current == ToolStateDisabled {
			return 0xf00c
		}
		return 0
	}(), Fn: func() {
		if err := agent.SetToolState(toolName, ToolStateDisabled); err == nil && changed != nil {
			changed(ToolStateDisabled)
		}
	}})
	menu.DisplayMenuAt(fmt.Sprintf("Tool state: %s", toolName), x, y)
}
