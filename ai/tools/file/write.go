package filetools

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"text/template"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/markdownutil"
	"github.com/lmorg/ttyphoon/utils/notes"
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

//go:embed write_auth_desc.md
var writeFileAuthDescription string

var writeFileAuthDescriptionTemplate = template.Must(template.New("write-file-auth-description").Funcs(markdownutil.TemplateFuncMap()).Parse(writeFileAuthDescription))

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
	return t.writeFiles(ctx, request.Archive)
}

func (t *Write) CallDescription(_ context.Context, input any) string {
	request, ok := input.(*writeFileInputT)
	if !ok || request == nil {
		return "Tool received an invalid structured input"
	}

	arc := txtar.Parse([]byte(request.Archive))
	descriptionInput := writeFileAuthDescriptionInput{Files: make([]writeFileAuthDescriptionEntry, 0, len(arc.Files))}
	for _, file := range arc.Files {
		entry := writeFileAuthDescriptionEntry{Name: file.Name, Content: string(file.Data)}
		filename, err := resolveWorkspacePath(t.agent, file.Name)
		if err != nil {
			entry.PathError = "Path cannot be resolved: " + err.Error()
		} else {
			_, statErr := os.Stat(filepath.Dir(filename))
			entry.CreatesDirectories = os.IsNotExist(statErr)
			if statErr != nil && !entry.CreatesDirectories {
				entry.PathError = "Parent directory cannot be inspected: " + statErr.Error()
			}
		}
		descriptionInput.Files = append(descriptionInput.Files, entry)
	}

	var description bytes.Buffer
	if err := writeFileAuthDescriptionTemplate.Execute(&description, descriptionInput); err != nil {
		return fmt.Sprintf("Unable to render description: %v", err)
	}
	return description.String()
}

func (t *Write) ObservationStructured(input any, output string, err error) aitypes.ToolObservation {
	request, ok := input.(*writeFileInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid structured input %T", input)}
	}
	return t.observation(request.Archive, output, err)
}

func (t *Write) writeFiles(_ context.Context, input string) (string, error) {
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
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			t.agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result += fmt.Sprintf("ERROR '%s': %s\n", filename, err)
			continue
		}

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
		if err == nil {
			notes.NotifyFileChanges(filename)
		}

		result += fmt.Sprintf("INFO '%s': file written successfully\n", filename)
	}

	debug.Log(result)
	return result, nil
}

type writeFileAuthDescriptionInput struct {
	Files []writeFileAuthDescriptionEntry
}

type writeFileAuthDescriptionEntry struct {
	Name               string
	Content            string
	CreatesDirectories bool
	PathError          string
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
