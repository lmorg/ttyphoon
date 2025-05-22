package agent

import (
	"context"

	"github.com/lmorg/mxtty/types"
)

type Tool interface {
	New(*Meta) (Tool, error)
	Enabled() bool
	Toggle()
	Name() string
	Description() string
	Call(context.Context, string) (string, error)
}

var _tools []Tool

func ToolsAdd(t Tool) {
	_tools = append(_tools, t)
}

func (meta *Meta) toolsInit() {
	for i := range _tools {
		newTool, err := _tools[i].New(meta)
		if err != nil {
			meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			continue
		}
		meta._tools = append(meta._tools, newTool)
	}
}
