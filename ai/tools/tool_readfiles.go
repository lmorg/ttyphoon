package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/callbacks"
	"golang.org/x/tools/txtar"
)

type ReadFiles struct {
	CallbacksHandler callbacks.Handler
	meta             *agent.Meta
	enabled          bool
}

func init() {
	agent.ToolsAdd(&ReadFiles{})
}

func (f *ReadFiles) New(meta *agent.Meta) (agent.Tool, error) {
	return &ReadFiles{meta: meta, enabled: true}, nil
}

func (f *ReadFiles) Enabled() bool { return f.enabled }
func (f *ReadFiles) Toggle()       { f.enabled = !f.enabled }

func (f *ReadFiles) Name() string { return "Read Files" }
func (f *ReadFiles) Path() string { return "internal" }
func (f *ReadFiles) Description() string {
	return `Open a local files for reading and return their contents.
Useful for debugging output that references local files.
The output of this tool will conform to the ` + "`txtar`" + ` specification.
Any files that could not be opened will be returned with the contents saying "!!! Cannot open file"
The input for this tool MUST be a JSON array of strings. Each array item will be a file you want the contents of.
`
}

func (f *ReadFiles) Call(ctx context.Context, input string) (response string, err error) {
	if debug.Trace {
		log.Printf("Agent tool '%s' input:\n%s", f.Name(), input)
		defer func() {
			log.Printf("Agent tool '%s' response:\n%s", f.Name(), response)
			log.Printf("Agent tool '%s' error: %v", f.Name(), err)
		}()
	}

	if f.CallbacksHandler != nil {
		f.CallbacksHandler.HandleToolStart(ctx, input)
	}

	var files []string
	jsonErr := json.Unmarshal([]byte(input), &files)
	if jsonErr != nil {
		return "call the tool error: input must be valid json, retry tool calling with correct json", nil
	}

	var archive txtar.Archive

	for i := range files {
		filename := files[i]

		if !strings.HasPrefix(filename, f.meta.Pwd) {
			filename = f.meta.Pwd + "/" + files[i]
		}

		f.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, f.meta.ServiceName()+" requesting file: "+filename[len(f.meta.Pwd):])

		var b []byte
		info, err := os.Stat(filename)
		if err != nil {
			b = []byte(fmt.Sprintf("!!! Cannot open file: %v", err))

		} else if info.Name()[0] == '.' {
			b = []byte(fmt.Sprintf("!!! Cannot open file: %s", "file hidden"))

		} else if b, err = os.ReadFile(filename); err != nil {
			b = []byte(fmt.Sprintf("!!! Cannot open file: %v", err))
		}

		archive.Files = append(archive.Files, txtar.File{
			Name: files[i],
			Data: b,
		})

		err = nil
	}

	response = string(txtar.Format(&archive))

	if f.CallbacksHandler != nil {
		f.CallbacksHandler.HandleToolEnd(ctx, response)
	}

	return response, nil
}
