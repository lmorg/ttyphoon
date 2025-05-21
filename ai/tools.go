package ai

import (
	"context"

	"github.com/lmorg/mxtty/types"
)

type tool interface {
	New(*AgentMeta) (tool, error)
	Enabled() bool
	Toggle()
	Name() string
	Description() string
	Call(context.Context, string) (string, error)
}

var _tools []tool

func ToolsAdd(t tool) {
	_tools = append(_tools, t)
}

func (meta *AgentMeta) toolsInit() {
	for i := range _tools {
		newTool, err := _tools[i].New(meta)
		if err != nil {
			meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			continue
		}
		meta._tools = append(meta._tools, newTool)
	}
}
