package filetools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/markdownutil"
	"github.com/lmorg/ttyphoon/utils/notes"
)

type DeleteDirectory struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&DeleteDirectory{})
}

func (t *DeleteDirectory) New(agt aitypes.Agent) (aitypes.Tool, error) {
	return &DeleteDirectory{agent: agt, enabled: true}, nil
}

func (t *DeleteDirectory) Enabled() bool { return t.enabled }
func (t *DeleteDirectory) Toggle()       { t.enabled = !t.enabled }
func (t *DeleteDirectory) Name() string  { return "deleteDirectory" }
func (t *DeleteDirectory) Path() string  { return "internal" }
func (t *DeleteDirectory) Description() string {
	return "Delete an empty directory inside the active project. Non-empty directories and symlinks are refused."
}
func (t *DeleteDirectory) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

type deleteDirectoryInputT struct {
	Path string `json:"path"`
}

func (t *DeleteDirectory) InputType() reflect.Type { return reflect.TypeOf(deleteDirectoryInputT{}) }

func (t *DeleteDirectory) CallStructured(_ context.Context, input any) (string, error) {
	request, ok := input.(*deleteDirectoryInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("deleteDirectory received an invalid structured input %T", input)
	}
	return t.remove(request.Path)
}

func (t *DeleteDirectory) CallDescription(_ context.Context, input any) string {
	request, ok := input.(*deleteDirectoryInputT)
	if !ok || request == nil {
		return "Tool received an invalid structured input"
	}
	path, err := resolveWorkspacePath(t.agent, request.Path)
	if err != nil {
		return "Delete directory: " + markdownutil.EscapeMarkdownText(request.Path) + "\nPath resolution failed: " + markdownutil.EscapeMarkdownText(err.Error())
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Sprintf("Delete directory: %s\nPath status: %s", markdownutil.EscapeMarkdownText(path), markdownutil.EscapeMarkdownText(err.Error()))
	}
	status := "Directory must be empty; its contents will not be deleted"
	if !info.IsDir() {
		status = "Path is not a directory; deletion will be refused"
	} else if isProjectRoot(t.agent, path) {
		status = "Path is the project root; deletion will be refused"
	}
	return fmt.Sprintf("Delete directory: %s\n%s", markdownutil.EscapeMarkdownText(path), markdownutil.EscapeMarkdownText(status))
}

func (t *DeleteDirectory) remove(requestedPath string) (string, error) {
	path, err := resolveWorkspacePath(t.agent, requestedPath)
	if err != nil {
		return "ERROR: " + err.Error(), nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Sprintf("ERROR inspecting %q: %v", path, err), nil
	}
	if !info.IsDir() {
		return fmt.Sprintf("ERROR: refusing to delete non-directory path %q", path), nil
	}
	if isProjectRoot(t.agent, path) {
		return "ERROR: refusing to delete the project root", nil
	}
	if err := os.Remove(path); err != nil {
		return fmt.Sprintf("ERROR deleting directory %q (only empty directories can be removed): %v", path, err), nil
	}
	notes.NotifyFileChanges(path)
	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" deleted directory: "+path)
	return fmt.Sprintf("INFO: deleted empty directory: %s", path), nil
}

func (t *DeleteDirectory) ObservationStructured(input any, output string, err error) aitypes.ToolObservation {
	request, ok := input.(*deleteDirectoryInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid input %T", input)}
	}
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok", Outputs: []string{request.Path}}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
	} else if strings.HasPrefix(output, "ERROR:") {
		observation.Status = "error"
		observation.Error = output
	}
	return observation
}

func isProjectRoot(agt aitypes.Agent, path string) bool {
	root, err := filepath.Abs(filepath.Clean(strings.TrimSpace(agt.ProjectRoot())))
	if err != nil || root == "." {
		return false
	}
	root, rootErr := filepath.EvalSymlinks(root)
	resolvedPath, pathErr := filepath.EvalSymlinks(path)
	return rootErr == nil && pathErr == nil && root == resolvedPath
}
