package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/callbacks"
)

type ReadFile struct {
	CallbacksHandler callbacks.Handler
	meta             *AgentMeta
	enabled          bool
}

func init() {
	ToolsAdd(&ReadFile{})
}

func (f *ReadFile) New(meta *AgentMeta) (tool, error) {
	return &ReadFile{meta: meta, enabled: true}, nil
}

func (f *ReadFile) Enabled() bool { return f.enabled }
func (f *ReadFile) Toggle()       { f.enabled = !f.enabled }

func (f *ReadFile) Description() string {
	return `Open a local file for reading and return its contents.
Useful for debugging output that references local files.`
}

func (f *ReadFile) Name() string {
	return "Read File"
}

func (f *ReadFile) Call(ctx context.Context, input string) (string, error) {
	if f.CallbacksHandler != nil {
		f.CallbacksHandler.HandleToolStart(ctx, input)
	}

	debug.Log(input)
	filename := f.meta.Pwd + "/" + input
	var result string
	var b []byte

	f.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, f.meta.ServiceName()+" requesting file: "+filename)

	info, err := os.Stat(filename)
	if err != nil {
		result = fmt.Sprintf("ERROR: %v", err)
		goto fin
	}

	if info.Name()[0] == '.' {
		result = "ERROR: You are not allowed to access files prefixed with a dot"
		goto fin
	}

	b, err = os.ReadFile(filename)
	if err != nil {
		result = err.Error()
	} else {
		result = string(b)
	}

fin:

	if f.CallbacksHandler != nil {
		f.CallbacksHandler.HandleToolEnd(ctx, result)
	}

	debug.Log(result)
	return result, nil
}
