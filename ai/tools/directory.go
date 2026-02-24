package tools

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"github.com/tmc/langchaingo/callbacks"
)

type Directory struct {
	CallbacksHandler callbacks.Handler
	agent            *agent.Agent
	enabled          bool
}

func init() {
	agent.ToolsAdd(&Directory{})
}

func (t Directory) New(agent *agent.Agent) (agent.Tool, error) {
	return &Directory{agent: agent, enabled: true}, nil
}

func (t *Directory) Enabled() bool { return t.enabled }
func (t *Directory) Toggle()       { t.enabled = !t.enabled }

func (t *Directory) Description() string {
	return `Check the contents of a directory.
Returns a bullet point list of all files and directories found inside a directory.`
}

func (t *Directory) Name() string { return "Read Directory" }
func (t *Directory) Path() string { return "internal" }

func (t *Directory) Call(ctx context.Context, input string) (response string, err error) {
	if debug.Trace {
		log.Printf("Agent tool '%s' input:\n%s", t.Name(), input)
		defer func() {
			log.Printf("Agent tool '%s' response:\n%s", t.Name(), response)
			log.Printf("Agent tool '%s' error: %v", t.Name(), err)
		}()
	}

	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolStart(ctx, input)
	}

	var pathname string
	if strings.HasPrefix(input, t.agent.Meta.Pwd) {
		pathname = input
	} else {
		pathname = t.agent.Meta.Pwd + "/" + input
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
			result.WriteString(fmt.Sprintf("- Directory: '%s'\n", path))
		} else {
			result.WriteString(fmt.Sprintf("- File: '%s'\n", path))
		}

		return nil
	})

	if err != nil {
		result.WriteString(fmt.Sprintf("- Error: %v\n", err))
	}

	response = result.String()

	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolEnd(ctx, response)
	}

	return response, nil
}
