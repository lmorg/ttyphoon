package tools

import (
	"context"
	"fmt"
	"log"

	"github.com/lmorg/mcp-web-scraper/langchain"
	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/app"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/tools/duckduckgo"
)

type Wrapper struct {
	CallbacksHandler callbacks.Handler
	meta             *agent.Meta
	tool             tools.Tool
	invoker          func() (tools.Tool, error)
	addDescription   string
	enabled          bool
}

func init() {
	agent.ToolsAdd(&Wrapper{invoker: invokeDDG, addDescription: "Only search the web if you are not already confident with an answer"})
	agent.ToolsAdd(&Wrapper{invoker: invokeScraper})
}

func (t *Wrapper) New(meta *agent.Meta) (agent.Tool, error) {
	tool, err := t.invoker()
	if err != nil {
		return nil, err
	}
	return &Wrapper{meta: meta, tool: tool, enabled: true}, nil
}

func (t *Wrapper) Enabled() bool { return t.enabled }
func (t *Wrapper) Toggle()       { t.enabled = !t.enabled }

func (t *Wrapper) Name() string        { return t.tool.Name() }
func (t *Wrapper) Path() string        { return "internal" }
func (t *Wrapper) Description() string { return t.tool.Description() + "\n" + t.addDescription }

func (t *Wrapper) Call(ctx context.Context, input string) (response string, err error) {
	if debug.Trace {
		log.Printf("Agent tool '%s' input:\n%s", t.Name(), input)
		defer func() {
			log.Printf("Agent tool '%s' response:\n%s", t.Name(), response)
			log.Printf("Agent tool '%s' error: %v", t.Name(), err)
		}()
	}

	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolStart(ctx, input)
	}

	t.meta.Renderer.DisplayNotification(types.NOTIFY_INFO,
		fmt.Sprintf("%s is running a %s: %s", t.meta.ServiceName(), t.Name(), input))

	response, err = t.tool.Call(ctx, input)

	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolEnd(ctx, response)
	}

	return
}

/////

func invokeDDG() (tools.Tool, error) {
	return duckduckgo.New(10, fmt.Sprintf("%s/%s", app.Name, app.Version()))
}

func invokeScraper() (tools.Tool, error) {
	return langchain.NewScraper(), nil
}
