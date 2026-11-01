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

type RenamePath struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&RenamePath{})
}

func (t *RenamePath) New(agt aitypes.Agent) (aitypes.Tool, error) {
	return &RenamePath{agent: agt, enabled: true}, nil
}

func (t *RenamePath) Enabled() bool { return t.enabled }
func (t *RenamePath) Toggle()       { t.enabled = !t.enabled }
func (t *RenamePath) Name() string  { return "renamePath" }
func (t *RenamePath) Path() string  { return "internal" }
func (t *RenamePath) Description() string {
	return "Rename or move a regular file or directory inside the active project. Missing destination parents are created. Existing destinations, symlinks, and project-root renames are refused."
}
func (t *RenamePath) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

type renamePathInputT struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

func (t *RenamePath) InputType() reflect.Type { return reflect.TypeOf(renamePathInputT{}) }

func (t *RenamePath) CallStructured(_ context.Context, input any) (string, error) {
	request, ok := input.(*renamePathInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("renamePath received an invalid structured input %T", input)
	}
	return t.rename(request.Source, request.Destination)
}

func (t *RenamePath) CallDescription(_ context.Context, input any) string {
	request, ok := input.(*renamePathInputT)
	if !ok || request == nil {
		return "Tool received an invalid structured input"
	}
	source, sourceErr := resolveWorkspacePath(t.agent, request.Source)
	destination, destinationErr := resolveWorkspacePath(t.agent, request.Destination)
	if sourceErr != nil || destinationErr != nil {
		return fmt.Sprintf("Rename path\nSource: %s\nDestination: %s\nPath resolution failed: %s",
			markdownutil.EscapeMarkdownText(request.Source),
			markdownutil.EscapeMarkdownText(request.Destination),
			markdownutil.EscapeMarkdownText(errorsJoin(sourceErr, destinationErr)))
	}
	status := "Destination does not exist"
	if _, err := os.Lstat(destination); err == nil {
		status = "Destination already exists; rename will be refused"
	} else if os.IsNotExist(err) {
		parent := filepath.Dir(destination)
		if _, parentErr := os.Stat(parent); os.IsNotExist(parentErr) {
			status = "Destination and missing parent directories will be created"
		} else {
			status = "Destination parent already exists; destination will be created"
		}
	} else {
		status = "Cannot inspect destination: " + err.Error()
	}
	return fmt.Sprintf("Rename/move path\nSource: %s\nDestination: %s\n%s",
		markdownutil.EscapeMarkdownText(source),
		markdownutil.EscapeMarkdownText(destination),
		markdownutil.EscapeMarkdownText(status))
}

func (t *RenamePath) rename(sourcePath, destinationPath string) (string, error) {
	source, err := resolveWorkspacePath(t.agent, sourcePath)
	if err != nil {
		return "ERROR resolving source: " + err.Error(), nil
	}
	destination, err := resolveWorkspacePath(t.agent, destinationPath)
	if err != nil {
		return "ERROR resolving destination: " + err.Error(), nil
	}
	if source == destination {
		return "INFO: source and destination are the same; no changes made", nil
	}

	sourceInfo, err := os.Lstat(source)
	if err != nil {
		return fmt.Sprintf("ERROR inspecting source %q: %v", source, err), nil
	}
	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		return "ERROR: refusing to rename a symlink", nil
	}
	if !sourceInfo.Mode().IsRegular() && !sourceInfo.IsDir() {
		return "ERROR: source must be a regular file or directory", nil
	}
	if sourceInfo.IsDir() && isProjectRoot(t.agent, source) {
		return "ERROR: refusing to rename the project root", nil
	}
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Sprintf("ERROR: destination already exists: %s", destination), nil
	} else if !os.IsNotExist(err) {
		return fmt.Sprintf("ERROR inspecting destination %q: %v", destination, err), nil
	}
	if sourceInfo.IsDir() {
		if err := destinationInsideDirectory(source, filepath.Dir(destination)); err != nil {
			return "ERROR: " + err.Error(), nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Sprintf("ERROR creating destination parent directory: %v", err), nil
	}
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Sprintf("ERROR: destination already exists: %s", destination), nil
	} else if !os.IsNotExist(err) {
		return fmt.Sprintf("ERROR inspecting destination %q: %v", destination, err), nil
	}
	if err := os.Rename(source, destination); err != nil {
		return fmt.Sprintf("ERROR renaming %q to %q: %v", source, destination, err), nil
	}
	notes.NotifyFileChanges(source, destination)
	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" renamed path: "+source+" -> "+destination)
	return fmt.Sprintf("INFO: renamed %s to %s", source, destination), nil
}

func destinationInsideDirectory(sourceDirectory, destinationParent string) error {
	sourceRoot, err := filepath.EvalSymlinks(sourceDirectory)
	if err != nil {
		return fmt.Errorf("resolve source directory: %w", err)
	}
	resolvedParent, err := resolveExistingPrefix(destinationParent)
	if err != nil {
		return fmt.Errorf("resolve destination parent: %w", err)
	}
	relative, err := filepath.Rel(sourceRoot, resolvedParent)
	if err != nil {
		return fmt.Errorf("compare source and destination paths: %w", err)
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return fmt.Errorf("cannot move a directory into itself")
	}
	return nil
}

func errorsJoin(first, second error) string {
	if first != nil && second != nil {
		return first.Error() + "; " + second.Error()
	}
	if first != nil {
		return first.Error()
	}
	if second != nil {
		return second.Error()
	}
	return "unknown path error"
}

func (t *RenamePath) ObservationStructured(input any, output string, err error) aitypes.ToolObservation {
	request, ok := input.(*renamePathInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid input %T", input)}
	}
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok", FilesModified: []string{request.Source, request.Destination}}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
	} else if strings.HasPrefix(output, "ERROR:") {
		observation.Status = "error"
		observation.Error = output
	}
	return observation
}
