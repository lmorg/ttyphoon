package filetools

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/markdownutil"
)

type CreateDirectory struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&CreateDirectory{})
}

func (t *CreateDirectory) New(agt aitypes.Agent) (aitypes.Tool, error) {
	return &CreateDirectory{agent: agt, enabled: true}, nil
}

func (t *CreateDirectory) Enabled() bool { return t.enabled }
func (t *CreateDirectory) Toggle()       { t.enabled = !t.enabled }
func (t *CreateDirectory) Name() string  { return "createDirectory" }
func (t *CreateDirectory) Path() string  { return "internal" }
func (t *CreateDirectory) Description() string {
	return "Create a directory inside the active project. Missing parent directories are created too."
}
func (t *CreateDirectory) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

type createDirectoryInputT struct {
	Path string `json:"path"`
}

func (t *CreateDirectory) InputType() reflect.Type { return reflect.TypeOf(createDirectoryInputT{}) }

func (t *CreateDirectory) CallStructured(_ context.Context, input any) (string, error) {
	request, ok := input.(*createDirectoryInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("createDirectory received an invalid structured input %T", input)
	}
	return t.create(request.Path)
}

func (t *CreateDirectory) CallDescription(_ context.Context, input any) string {
	request, ok := input.(*createDirectoryInputT)
	if !ok || request == nil {
		return "Tool received an invalid structured input"
	}
	path, err := resolveWorkspacePath(t.agent, request.Path)
	if err != nil {
		return "Create directory: " + markdownutil.EscapeMarkdownText(request.Path) + "\nPath resolution failed: " + markdownutil.EscapeMarkdownText(err.Error())
	}
	status := "Directory and any missing parents will be created"
	if info, statErr := os.Stat(path); statErr == nil {
		if info.IsDir() {
			status = "Directory already exists; no changes will be made"
		} else {
			status = "Path already exists and is not a directory"
		}
	} else if !os.IsNotExist(statErr) {
		status = "Cannot inspect path: " + statErr.Error()
	}
	return fmt.Sprintf("Create directory: %s\n%s", markdownutil.EscapeMarkdownText(path), markdownutil.EscapeMarkdownText(status))
}

func (t *CreateDirectory) create(requestedPath string) (string, error) {
	path, err := resolveWorkspacePath(t.agent, requestedPath)
	if err != nil {
		return "ERROR: " + err.Error(), nil
	}
	if info, statErr := os.Stat(path); statErr == nil && !info.IsDir() {
		return fmt.Sprintf("ERROR: path already exists and is not a directory: %s", path), nil
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return "ERROR: " + statErr.Error(), nil
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Sprintf("ERROR creating directory %q: %v", path, err), nil
	}
	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" created directory: "+path)
	return fmt.Sprintf("INFO: directory is ready: %s", path), nil
}

func (t *CreateDirectory) ObservationStructured(input any, output string, err error) aitypes.ToolObservation {
	request, ok := input.(*createDirectoryInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid input %T", input)}
	}
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok", Outputs: []string{request.Path}}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
	} else if len(output) >= 6 && output[:6] == "ERROR:" {
		observation.Status = "error"
		observation.Error = output
	}
	return observation
}
