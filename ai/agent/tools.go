package agent

import (
	"fmt"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/types"
)

var _tools []aitypes.Tool

func (agent *Agent) toolsInit() {
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

	agent._tools = append(agent._tools, tool)
	agent.Reload()

	return nil
}

func (agent *Agent) ChooseTools(cancel types.MenuCallbackT) {
	s := make([]string, len(agent._tools))
	for i, tool := range agent._tools {
		s[i] = fmt.Sprintf("%s == %v", tool.Name(), tool.Enabled())
	}

	fnOk := func(i int) {
		agent._tools[i].Toggle()
		agent.Reload()
		agent.ChooseTools(cancel)
	}

	agent.renderer.DisplayMenu("AI tools", s, nil, fnOk, cancel)
}

func (agent *Agent) ListTools() []map[string]interface{} {
	tools := make([]map[string]interface{}, len(agent._tools))
	for i, tool := range agent._tools {
		tools[i] = map[string]interface{}{
			"name":    tool.Name(),
			"enabled": tool.Enabled(),
		}
	}
	return tools
}

func (agent *Agent) SetToolEnabled(toolName string, enabled bool) error {
	for _, tool := range agent._tools {
		if tool.Name() == toolName {
			currentEnabled := tool.Enabled()
			if currentEnabled != enabled {
				tool.Toggle()
				agent.Reload()
			}
			return nil
		}
	}
	return fmt.Errorf("tool %q not found", toolName)
}
