package filetools

import (
	"context"
	_ "embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/types"
)

type Directory struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&Directory{})
}

//go:embed directory_description.md
var directoryDescription string

func (t Directory) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &Directory{agent: agent, enabled: true}, nil
}

func (t *Directory) Enabled() bool { return t.enabled }
func (t *Directory) Toggle()       { t.enabled = !t.enabled }

func (t *Directory) Name() string { return "readDirectory" }
func (t *Directory) Path() string { return "internal" }
func (t *Directory) Description() string {
	return ``
}

func (t *Directory) Call(ctx context.Context, input string) (response string, err error) {
	pathname, err := resolveWorkspacePath(t.agent, input)
	if err != nil {
		return fmt.Sprintf("ERROR: %s\n", err), nil
	}

	var result strings.Builder

	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" is querying directory: "+pathname)

	err = filepath.WalkDir(pathname, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if len(t.Name()) > 1 && t.Name()[0] == '.' {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			fmt.Fprintf(&result, "- Directory: '%s'\n", path)
		} else {
			fmt.Fprintf(&result, "- File: '%s'\n", path)
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(&result, "- Error: %v\n", err)
	}

	response = result.String()

	return response, nil
}
