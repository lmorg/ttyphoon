package ai

import (
	"context"
	"fmt"

	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/tools"
)

type Wrapper struct {
	meta *AgentMeta
	tool tools.Tool
}

func (wrapper Wrapper) Description() string {
	return wrapper.tool.Description()
}

func (wrapper Wrapper) Name() string {
	return wrapper.tool.Name()
}

func (wrapper Wrapper) Call(ctx context.Context, input string) (string, error) {
	wrapper.meta.Renderer.DisplayNotification(types.NOTIFY_INFO,
		fmt.Sprintf("%s is running a %s: %s", wrapper.meta.ServiceName(), wrapper.Name(), input))
	return wrapper.tool.Call(ctx, input)
}
