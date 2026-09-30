package filetools

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"reflect"

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

//go:embed write_description.md
var writeFileDescription string

func (t *Write) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &Write{agent: agent, enabled: true}, nil
}

func (t *Write) Enabled() bool { return t.enabled }
func (t *Write) Toggle()       { t.enabled = !t.enabled }

func (t *Write) Name() string { return "writeFile" }
func (t *Write) Path() string { return "internal" }
func (t *Write) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

type writeFileInputT struct {
	Archive string `json:"archive"`
}

func (t *Write) Description() string {
	return writeFileDescription
}

func (t *Write) InputType() reflect.Type { return reflect.TypeOf(writeFileInputT{}) }

func (t *Write) CallStructured(ctx context.Context, input any) (string, error) {
	request, ok := input.(*writeFileInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("writeFile received an invalid structured input %T", input)
	}
	return t.Call(ctx, request.Archive)
}

func (t *Write) ObservationStructured(input any, output string, err error) aitypes.ToolObservation {
	request, ok := input.(*writeFileInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid structured input %T", input)}
	}
	return t.observation(request.Archive, output, err)
}

func (t *Write) Call(ctx context.Context, input string) (string, error) {
	debug.Log(input)

	var result string

	arc := txtar.Parse([]byte(input))
	for i := range arc.Files {
		filename, err := resolveWorkspacePath(t.agent, arc.Files[i].Name)
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

func (t *Write) Observation(input, output string, err error) aitypes.ToolObservation {
	return t.observation(input, output, err)
}

func (t *Write) observation(input, output string, err error) aitypes.ToolObservation {
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok"}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
		return observation
	}
	arc := txtar.Parse([]byte(input))
	files := make([]string, 0, len(arc.Files))
	for _, file := range arc.Files {
		files = append(files, file.Name)
	}
	observation.FilesModified = files
	observation.Counts = map[string]int{"files": len(files), "bytes": len(input)}
	return observation
}
