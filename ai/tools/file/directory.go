package filetools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/find"
)

type Directory struct {
	agent   aitypes.Agent
	enabled bool
	exclude []string
}

func init() {
	agent.ToolsAdd(&Directory{})
}

//go:embed directory_description.md
var directoryDescription string

func (t Directory) New(agent aitypes.Agent) (aitypes.Tool, error) {
	exclude := make([]string, len(config.Config.Notes.ExcludeDirectories))
	i := 0
	for s := range config.Config.Notes.ExcludeDirectories {
		exclude[i] = s
		i++
	}

	return &Directory{
		agent:   agent,
		exclude: exclude,
		enabled: true,
	}, nil
}

func (t *Directory) Enabled() bool { return t.enabled }
func (t *Directory) Toggle()       { t.enabled = !t.enabled }

func (t *Directory) Name() string { return "readDirectory" }
func (t *Directory) Path() string { return "internal" }
func (t *Directory) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "allow"}
}
func (t *Directory) Description() string {
	return directoryDescription
}

type file struct {
	Name  string
	IsDir bool
}

func (t *Directory) Call(ctx context.Context, input string) (response string, err error) {
	/*pathname, err := resolveWorkspacePath(t.agent, input)
	if err != nil {
		return fmt.Sprintf("ERROR: %s\n", err), nil
	}*/
	pathname := t.agent.ProjectRoot()

	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" is querying directory: "+pathname)

	match, err := find.New(input)
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err), nil
	}

	l := len(pathname) + 1
	var files []file

	err = filepath.WalkDir(pathname, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		/*if len(t.Name()) > 1 && t.Name()[0] == '.' {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}*/

		if len(path) < l {
			return nil
		}

		if !match.MatchString(path) {
			return nil
		}

		isDir := d.IsDir()
		if isDir && slices.Contains(t.exclude, d.Name()) {
			return fs.SkipDir
		}

		files = append(files, file{
			Name:  path[l:],
			IsDir: isDir,
		})
		return nil
	})

	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	b, err := json.Marshal(&files)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return string(b), nil
}

func (t *Directory) Observation(input, output string, err error) aitypes.ToolObservation {
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok", DirectoriesListed: []string{t.agent.ProjectRoot()}}
	if input != "" {
		observation.Inputs = []string{input}
	}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
		return observation
	}

	var files []file
	if json.Unmarshal([]byte(output), &files) == nil {
		observation.Counts = map[string]int{"matches": len(files)}
	}
	return observation
}
