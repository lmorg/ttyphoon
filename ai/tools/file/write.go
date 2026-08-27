package filetools

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"golang.org/x/tools/txtar"
)

type Write struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&Write{})
}

//go:embed read_description.md
var writeFileDescription string

func (t *Write) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &Write{agent: agent, enabled: false}, nil
}

func (t *Write) Enabled() bool { return t.enabled }
func (t *Write) Toggle()       { t.enabled = !t.enabled }

func (t *Write) Name() string { return "writeFile" }
func (t *Write) Path() string { return "internal" }
func (t *Write) Description() string {
	return writeFileDescription
}

func (t *Write) Call(ctx context.Context, input string) (string, error) {
	debug.Log(input)

	var result string

	arc := txtar.Parse([]byte(input))
	for i := range arc.Files {
		filename, err := resolveWorkspacePath(t.agent.GetMeta().Pwd, arc.Files[i].Name)
		if err != nil {
			result += fmt.Sprintf("ERROR '%s': %s\n", arc.Files[i].Name, err)
			continue
		}

		t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" writing file: "+filename)

		f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0664)
		if err != nil {
			t.agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result += fmt.Sprintf("ERROR '%s': %s\n", filename, err)
			continue
		}
		_, err = f.Write(arc.Files[i].Data)
		if err != nil {
			t.agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result += fmt.Sprintf("ERROR '%s': %s\n", filename, err)
			continue
		}

		err = f.Close()
		if err != nil {
			t.agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result += fmt.Sprintf("ERROR '%s': %s\n", filename, err)
			// continue // don't need to "continue" here
		}

		result += fmt.Sprintf("INFO '%s': file written successfully\n", filename)
	}

	debug.Log(result)
	return result, nil
}
