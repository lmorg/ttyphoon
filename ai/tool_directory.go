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
	meta             *AgentMeta
}

func (d Directory) Description() string {
	return `Check the contents of a directory. 
Returns a bullet point list of all files and directories found inside a directory.`
}

func (d Directory) Name() string {
	return "directory"
}

func (d Directory) Call(ctx context.Context, input string) (string, error) {
	if d.CallbacksHandler != nil {
		d.CallbacksHandler.HandleToolStart(ctx, input)
	}

	debug.Log(input)

	var pathname string
	if strings.HasPrefix(input, d.meta.Pwd) {
		pathname = input
	} else {
		pathname = d.meta.Pwd + "/" + input
	}

	var result strings.Builder

	d.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, d.meta.ServiceName()+" is querying directory: "+pathname)

	err := filepath.Walk(pathname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.WriteString(fmt.Sprintf("- Error accessing '%s': %v\n", path, err))
			return err
		}

		if info.Name()[0] == '.' && len(info.Name()) > 1 {
			return fmt.Errorf("ignoring %s", info.Name())
		}

		if info.IsDir() {
			result.WriteString(fmt.Sprintf("- Directory: '%s'\n", path))
		} else {
			result.WriteString(fmt.Sprintf("- File: '%s'\n", path))
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
