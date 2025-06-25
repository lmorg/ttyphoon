package tools

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/callbacks"
)

type Directory struct {
	CallbacksHandler callbacks.Handler
	meta             *agent.Meta
	enabled          bool
}

func init() {
	agent.ToolsAdd(&Directory{})
}

func (d Directory) New(meta *agent.Meta) (agent.Tool, error) {
	return &Directory{meta: meta, enabled: true}, nil
}

func (d *Directory) Enabled() bool { return d.enabled }
func (d *Directory) Toggle()       { d.enabled = !d.enabled }

func (d *Directory) Description() string {
	return `Check the contents of a directory.
Returns a bullet point list of all files and directories found inside a directory.`
}

func (d *Directory) Name() string { return "Read Directory" }
func (d *Directory) Path() string { return "internal" }

func (d *Directory) Call(ctx context.Context, input string) (response string, err error) {
	if debug.Trace {
		log.Printf("Agent tool '%s' input:\n%s", d.Name(), input)
		defer func() {
			log.Printf("Agent tool '%s' response:\n%s", d.Name(), response)
			log.Printf("Agent tool '%s' error: %v", d.Name(), err)
		}()
	}

	if d.CallbacksHandler != nil {
		d.CallbacksHandler.HandleToolStart(ctx, input)
	}

	var pathname string
	if strings.HasPrefix(input, d.meta.Pwd) {
		pathname = input
	} else {
		pathname = d.meta.Pwd + "/" + input
	}

	var result strings.Builder

	d.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, d.meta.ServiceName()+" is querying directory: "+pathname)

	err = filepath.WalkDir(pathname, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if len(d.Name()) > 1 && d.Name()[0] == '.' {
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

	if d.CallbacksHandler != nil {
		d.CallbacksHandler.HandleToolEnd(ctx, response)
	}

	return response, nil
}
