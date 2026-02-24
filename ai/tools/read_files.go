package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"github.com/tmc/langchaingo/callbacks"
	"golang.org/x/tools/txtar"
)

type ReadFiles struct {
	CallbacksHandler callbacks.Handler
	agent            *agent.Agent
	enabled          bool
}

func init() {
	agent.ToolsAdd(&ReadFiles{})
}

func (f *ReadFiles) New(agent *agent.Agent) (agent.Tool, error) {
	return &ReadFiles{agent: agent, enabled: true}, nil
}

func (t *ReadFiles) Enabled() bool { return t.enabled }
func (t *ReadFiles) Toggle()       { t.enabled = !t.enabled }

func (t *ReadFiles) Name() string { return "Read Files" }
func (t *ReadFiles) Path() string { return "internal" }
func (t *ReadFiles) Description() string {
	return `Open a local files for reading and return their contents.
Useful for debugging output that references local files.
The output of this tool will conform to the ` + "`txtar`" + ` specification.
Any files that could not be opened will be returned with the contents saying "!!! Cannot open file"
The input for this tool MUST be a JSON array of strings. Each array item will be a file you want the contents of.
`
}

func (t *ReadFiles) Call(ctx context.Context, input string) (response string, err error) {
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

	var files []string
	jsonErr := json.Unmarshal([]byte(input), &files)
	if jsonErr != nil {
		return "call the tool error: input must be valid json, retry tool calling with correct json", nil
	}

	var archive txtar.Archive

	for i := range files {
		filename := files[i]

		if !strings.HasPrefix(filename, t.agent.Meta.Pwd) {
			filename = t.agent.Meta.Pwd + "/" + files[i]
		}

		t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" requesting file: "+filename[len(t.agent.Meta.Pwd):])

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

	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolEnd(ctx, response)
	}

	return response, nil
}
