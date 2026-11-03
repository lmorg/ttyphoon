package filetools

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/markdownutil"
	"github.com/lmorg/ttyphoon/utils/notes"
)

type DeleteFile struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&DeleteFile{})
}

func (t *DeleteFile) New(agt aitypes.Agent) (aitypes.Tool, error) {
	return &DeleteFile{agent: agt, enabled: true}, nil
}

func (t *DeleteFile) Enabled() bool { return t.enabled }
func (t *DeleteFile) Toggle()       { t.enabled = !t.enabled }
func (t *DeleteFile) Name() string  { return "deleteFile" }
func (t *DeleteFile) Path() string  { return "internal" }
func (t *DeleteFile) Description() string {
	return "Delete one regular file inside the active project. Directories and symlinks are not deleted by this tool."
}
func (t *DeleteFile) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

type deleteFileInputT struct {
	Path string `json:"path"`
}

func (t *DeleteFile) InputType() reflect.Type { return reflect.TypeOf(deleteFileInputT{}) }

func (t *DeleteFile) CallStructured(_ context.Context, input any) (string, error) {
	request, ok := input.(*deleteFileInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("deleteFile received an invalid structured input %T", input)
	}
	return t.remove(request.Path)
}

func (t *DeleteFile) CallDescription(_ context.Context, input any) string {
	request, ok := input.(*deleteFileInputT)
	if !ok || request == nil {
		return "Tool received an invalid structured input"
	}
	path, err := resolveWorkspacePath(t.agent, request.Path)
	if err != nil {
		return "Delete file: " + markdownutil.EscapeMarkdownText(request.Path) + "\nPath resolution failed: " + markdownutil.EscapeMarkdownText(err.Error())
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Sprintf("Delete file: %s\nPath status: %s", markdownutil.EscapeMarkdownText(path), markdownutil.EscapeMarkdownText(err.Error()))
	}
	status := "Regular file will be deleted"
	if !info.Mode().IsRegular() {
		status = "Path is not a regular file; deletion will be refused"
	}
	return fmt.Sprintf("Delete file: %s\n%s", markdownutil.EscapeMarkdownText(path), markdownutil.EscapeMarkdownText(status))
}

func (t *DeleteFile) remove(requestedPath string) (string, error) {
	path, err := resolveWorkspacePath(t.agent, requestedPath)
	if err != nil {
		return "ERROR: " + err.Error(), nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Sprintf("ERROR inspecting %q: %v", path, err), nil
	}
	if !info.Mode().IsRegular() {
		return fmt.Sprintf("ERROR: refusing to delete non-regular file %q", path), nil
	}
	if err := os.Remove(path); err != nil {
		return fmt.Sprintf("ERROR deleting %q: %v", path, err), nil
	}
	notes.NotifyFileChanges(path)
	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" deleted file: "+path)
	return fmt.Sprintf("INFO: deleted file: %s", path), nil
}

func (t *DeleteFile) ObservationStructured(input any, output string, err error) aitypes.ToolObservation {
	request, ok := input.(*deleteFileInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid input %T", input)}
	}
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok", FilesModified: []string{request.Path}}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
	} else if strings.HasPrefix(output, "ERROR:") {
		observation.Status = "error"
		observation.Error = output
	}
	return observation
}
