package filetools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

type PatchFile struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&PatchFile{})
}

//go:embed patch_description.md
var patchFileDescription string

func (t *PatchFile) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &PatchFile{agent: agent, enabled: true}, nil
}

func (t *PatchFile) Enabled() bool { return t.enabled }
func (t *PatchFile) Toggle()       { t.enabled = !t.enabled }

func (t *PatchFile) Name() string        { return "patchFile" }
func (t *PatchFile) Path() string        { return "internal" }
func (t *PatchFile) Description() string { return patchFileDescription }
func (t *PatchFile) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

type patchEditT struct {
	Old string `json:"old"`
	New string `json:"new"`
}

type patchInputT struct {
	File  string       `json:"file"`
	Edits []patchEditT `json:"edits"`
}

func (t *PatchFile) Call(ctx context.Context, input string) (string, error) {
	debug.Log(input)

	var patch patchInputT
	if err := json.Unmarshal([]byte(input), &patch); err != nil {
		return fmt.Sprintf("ERROR: input must be valid JSON matching the tool schema: %s", err), nil
	}

	if strings.TrimSpace(patch.File) == "" {
		return "ERROR: 'file' is required", nil
	}
	if len(patch.Edits) == 0 {
		return "ERROR: 'edits' must contain at least one edit", nil
	}

	filename, err := resolveWorkspacePath(t.agent, patch.File)
	if err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': %s", patch.File, err)), nil
	}

	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" patching file: "+patch.File)

	info, err := os.Stat(filename)
	if err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': cannot open file for patching, use writeFile to create new files: %s", patch.File, err)), nil
	}
	if info.IsDir() {
		return t.fail(fmt.Sprintf("ERROR '%s': path is a directory", patch.File)), nil
	}

	b, err := os.ReadFile(filename)
	if err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': %s", patch.File, err)), nil
	}

	patched, err := applyPatchEdits(string(b), patch.Edits)
	if err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': %s", patch.File, err)), nil
	}

	// Written via a temp file in the same directory so a failed write can't
	// truncate the original.
	if err := writeFileAtomic(filename, []byte(patched), info.Mode().Perm()); err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': %s", patch.File, err)), nil
	}

	result := fmt.Sprintf("INFO '%s': %d edit(s) applied successfully\n", patch.File, len(patch.Edits))
	debug.Log(result)
	return result, nil
}

func (t *PatchFile) fail(message string) string {
	t.agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, message)
	debug.Log(message)
	return message + "\nNo changes were written to disk.\n"
}

// applyPatchEdits applies every edit or none: each `old` must match exactly
// once against the result of the preceding edits.
func applyPatchEdits(content string, edits []patchEditT) (string, error) {
	for i, edit := range edits {
		if edit.Old == "" {
			return "", fmt.Errorf("edit %d: 'old' cannot be empty", i+1)
		}

		switch count := strings.Count(content, edit.Old); count {
		case 1:
			content = strings.Replace(content, edit.Old, edit.New, 1)
		case 0:
			return "", fmt.Errorf("edit %d: 'old' text was not found; it must match the file exactly, including whitespace and indentation", i+1)
		default:
			return "", fmt.Errorf("edit %d: 'old' text matched %d locations; include more surrounding context to make it unique", i+1, count)
		}
	}

	return content, nil
}

func writeFileAtomic(filename string, data []byte, perm os.FileMode) error {
	dir, base := filepath.Split(filename)

	f, err := os.CreateTemp(dir, "."+base+".patch-*")
	if err != nil {
		return err
	}
	tmpName := f.Name()

	defer func() {
		_ = f.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, perm); err != nil {
		return err
	}

	return os.Rename(tmpName, filename)
}
