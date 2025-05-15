package ai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/callbacks"
)

type Directory struct {
	CallbacksHandler callbacks.Handler
	meta             *Meta
}

func (d Directory) Description() string {
	return `Check the contents of a directory. 
		Returns a bullet point list of all files and directories found inside a directory.
		Files and directories prefixed with a dot or period will be ignored.`
}

func (d Directory) Name() string {
	return "directory"
}

func (d Directory) Call(ctx context.Context, input string) (string, error) {
	if d.CallbacksHandler != nil {
		d.CallbacksHandler.HandleToolStart(ctx, input)
	}

	debug.Log(input)
	pathname := d.meta.Pwd + "/" + input
	var result strings.Builder

	d.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, UseService+" is querying directory: "+pathname)

	err := filepath.Walk(pathname, func(path string, info os.FileInfo, err error) error {
		if info.Name()[0] == '.' {
			return fmt.Errorf("ignoring %s", info.Name())
		}

		if info.IsDir() {
			result.WriteString(fmt.Sprintf("- Directory: '%s'\n", path))
		} else {
			result.WriteString(fmt.Sprintf("- File: '%s'\n", path))
		}

		if err != nil {
			result.WriteString(fmt.Sprintf("- Error accessing '%s': %v\n", path, err))
		}

		return nil
	})

	if err != nil {
		result.WriteString(fmt.Sprintf("- Error: %v\n", err))
	}

	if d.CallbacksHandler != nil {
		d.CallbacksHandler.HandleToolEnd(ctx, result.String())
	}

	debug.Log(result.String())
	return result.String(), nil
}
