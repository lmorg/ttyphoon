package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/callbacks"
	"golang.org/x/tools/txtar"
)

type Write struct {
	CallbacksHandler callbacks.Handler
	meta             *agent.Meta
	enabled          bool
}

func init() {
	agent.ToolsAdd(&Write{})
}

func (t *Write) New(meta *agent.Meta) (agent.Tool, error) {
	return &Write{meta: meta, enabled: false}, nil
}

func (t *Write) Enabled() bool { return t.enabled }
func (t *Write) Toggle()       { t.enabled = !t.enabled }

func (t *Write) Name() string { return "Write File" }
func (t *Write) Path() string { return "internal" }

func (t *Write) Description() string {
	return `Writes new files, overwrites an existing files.
Useful for making changes, correcting mistakes, and writing new code and configuration.
File contents should contain the entire file, including parts of the file that are not changing.
The input of this tool MUST conform to the ` + "`txtar`" + ` specification.
`
}

func (t *Write) Call(ctx context.Context, input string) (string, error) {
	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolStart(ctx, input)
	}

	debug.Log(input)

	var result string

	arc := txtar.Parse([]byte(input))
	for i := range arc.Files {
		var filename string
		if strings.HasPrefix(arc.Files[i].Name, t.meta.Pwd) {
			filename = arc.Files[i].Name
		} else {
			filename = t.meta.Pwd + "/" + arc.Files[i].Name
		}

		t.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, t.meta.ServiceName()+" writing file: "+filename)

		f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0664)
		if err != nil {
			t.meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result += fmt.Sprintf("ERROR '%s': %s\n", filename, err)
			continue
		}
		_, err = f.Write(arc.Files[i].Data)
		if err != nil {
			t.meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result += fmt.Sprintf("ERROR '%s': %s\n", filename, err)
			continue
		}

		err = f.Close()
		if err != nil {
			t.meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result += fmt.Sprintf("ERROR '%s': %s\n", filename, err)
			// continue // don't need to "continue" here
		}

		result += fmt.Sprintf("INFO '%s': file written successfully\n", filename)
	}

	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolEnd(ctx, result)
	}

	debug.Log(result)
	return result, nil
}
