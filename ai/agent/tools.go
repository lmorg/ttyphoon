package agent

import (
	"context"
	"fmt"

	"github.com/lmorg/mxtty/debug"
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

func ToolsAdd(t Tool) {
	_tools = append(_tools, t)
}

func (meta *Meta) ChooseTools() {
	s := make([]string, len(meta._tools))
	for i, tool := range meta._tools {
		s[i] = fmt.Sprintf("%s == %v", tool.Name(), tool.Enabled())
	}

	fnOk := func(i int) {
		meta._tools[i].Toggle()
		meta.executor = nil
		meta.ChooseTools()
	}

	debug.Log(s)

	meta.Renderer.DisplayMenu("AI tools", s, nil, fnOk, nil)
}
