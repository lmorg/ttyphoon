package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/callbacks"
	"golang.org/x/tools/txtar"
)

type Write struct {
	CallbacksHandler callbacks.Handler
	meta             *AgentMeta
}

func (w Write) Description() string {
	return `Writes a new file or overwrites an existing file, to local disk.
Useful for making changes, correcting my mistakes, and writing new code and configuration.
You should only use this if told to write or update a file, or if you have asked for permission and I have consented.
File contents should contain the entire file, including parts of the file that are not changing.
The input of this tool MUST conform to the ` + "`txtar`" + ` specification.
`
}

func (w Write) Name() string {
	return "Write File"
}

func (w Write) Call(ctx context.Context, input string) (string, error) {
	if w.CallbacksHandler != nil {
		w.CallbacksHandler.HandleToolStart(ctx, input)
	}

	debug.Log(input)

	var result string

	arc := txtar.Parse([]byte(input))
	for i := range arc.Files {
		var filename string
		if strings.HasPrefix(arc.Files[i].Name, w.meta.Pwd) {
			filename = arc.Files[i].Name
		} else {
			filename = w.meta.Pwd + "/" + arc.Files[i].Name
		}

		w.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, w.meta.ServiceName()+" writing file: "+filename)

		f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0664)
		if err != nil {
			w.meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result += fmt.Sprintf("ERROR '%s': %s\n", filename, err)
			continue
		}
		_, err = f.Write(arc.Files[i].Data)
		if err != nil {
			w.meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result += fmt.Sprintf("ERROR '%s': %s\n", filename, err)
			continue
		}

		err = f.Close()
		if err != nil {
			w.meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result += fmt.Sprintf("ERROR '%s': %s\n", filename, err)
			// continue // don't need to "continue" here
		}

		result += fmt.Sprintf("INFO '%s': file written successfully\n", filename)
	}

	if w.CallbacksHandler != nil {
		w.CallbacksHandler.HandleToolEnd(ctx, result)
	}

	debug.Log(result)
	return result, nil
}
