package filetools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"golang.org/x/tools/txtar"
)

type ReadFiles struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&ReadFiles{})
}

//go:embed read_description.md
var readDescription string

func (f *ReadFiles) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &ReadFiles{agent: agent, enabled: true}, nil
}

func (t *ReadFiles) Enabled() bool { return t.enabled }
func (t *ReadFiles) Toggle()       { t.enabled = !t.enabled }

func (t *ReadFiles) Name() string { return "readFiles" }
func (t *ReadFiles) Path() string { return "internal" }
func (t *ReadFiles) Description() string {
	return readDescription
}

func (t *ReadFiles) Call(ctx context.Context, input string) (response string, err error) {
	if debug.Trace {
		log.Printf("Agent tool '%s' input:\n%s", t.Name(), input)
		defer func() {
			log.Printf("Agent tool '%s' response:\n%s", t.Name(), response)
			log.Printf("Agent tool '%s' error: %v", t.Name(), err)
		}()
	}

	var files []string
	jsonErr := json.Unmarshal([]byte(input), &files)
	if jsonErr != nil {
		return "call the tool error: input must be valid json, retry tool calling with correct json", nil
	}

	var archive txtar.Archive

	for i := range files {
		filename, pathErr := resolveWorkspacePath(t.agent.GetMeta().Pwd, files[i])
		if pathErr != nil {
			archive.Files = append(archive.Files, txtar.File{
				Name: files[i],
				Data: []byte(fmt.Sprintf("!!! Cannot open file: %v", pathErr)),
			})
			continue
		}

		t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" requesting file: "+filename[len(t.agent.GetMeta().Pwd):])

		var b []byte
		info, err := os.Stat(filename)
		if err != nil {
			b = fmt.Appendf(nil, "!!! Cannot open file: %v", err)

		} else if info.Name()[0] == '.' {
			b = fmt.Appendf(nil, "!!! Cannot open file: %s", "file hidden")

		} else if b, err = os.ReadFile(filename); err != nil {
			b = fmt.Appendf(nil, "!!! Cannot open file: %v", err)
		}

		archive.Files = append(archive.Files, txtar.File{
			Name: files[i],
			Data: b,
		})

		err = nil
	}

	response = string(txtar.Format(&archive))

	return response, nil
}
