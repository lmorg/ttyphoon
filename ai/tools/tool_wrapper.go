package tools

import (
	"context"
	"fmt"

	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/app"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/tools/duckduckgo"
	"github.com/tmc/langchaingo/tools/scraper"
)

type Wrapper struct {
	meta    *agent.Meta
	tool    tools.Tool
	invoker func() (tools.Tool, error)
	enabled bool
}

func init() {
	agent.ToolsAdd(&Wrapper{invoker: invokeDDG})
	agent.ToolsAdd(&Wrapper{invoker: invokeScaper})
}

func (wrapper *Wrapper) New(meta *agent.Meta) (agent.Tool, error) {
	tool, err := wrapper.invoker()
	if err != nil {
		return nil, err
	}
	return &Wrapper{meta: meta, tool: tool, enabled: true}, nil
}

func (wrapper *Wrapper) Enabled() bool { return wrapper.enabled }
func (wrapper *Wrapper) Toggle()       { wrapper.enabled = !wrapper.enabled }

func (wrapper *Wrapper) Name() string        { return wrapper.tool.Name() }
func (wrapper *Wrapper) Description() string { return wrapper.tool.Description() }

func (wrapper *Wrapper) Call(ctx context.Context, input string) (string, error) {
	wrapper.meta.Renderer.DisplayNotification(types.NOTIFY_INFO,
		fmt.Sprintf("%s is running a %s: %s", wrapper.meta.ServiceName(), wrapper.Name(), input))
	return wrapper.tool.Call(ctx, input)
}

/////

func invokeScaper() (tools.Tool, error) {
	return scraper.New()
}

func invokeDDG() (tools.Tool, error) {
	return duckduckgo.New(10, fmt.Sprintf("%s/%s", app.Name, app.Version()))
}
