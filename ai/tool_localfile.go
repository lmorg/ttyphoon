package ai

import (
	"context"
	"os"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/callbacks"
)

type LocalFile struct {
	CallbacksHandler callbacks.Handler
	meta             *Meta
}

func (f LocalFile) Description() string {
	return `Open a local file and return its contents.
		Useful for debugging output that references local files.
		Returns "no such file or directory" if the file doesn't exist, otherwise returns contents of file.
		Can only return files that are with the same path which the executable was ran, or a sub-directory within it.`
}

func (f LocalFile) Name() string {
	return "file"
}

func (f LocalFile) Call(ctx context.Context, input string) (string, error) {
	if f.CallbacksHandler != nil {
		f.CallbacksHandler.HandleToolStart(ctx, input)
	}

	debug.Log(input)
	filename := f.meta.Pwd + "/" + input
	var result string

	f.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, UseService+" requesting file: "+filename)

	b, err := os.ReadFile(filename)
	if err != nil {
		result = err.Error()
	} else {
		result = string(b)
	}

	if f.CallbacksHandler != nil {
		f.CallbacksHandler.HandleToolEnd(ctx, result)
	}

	debug.Log(result)
	return result, nil
}
