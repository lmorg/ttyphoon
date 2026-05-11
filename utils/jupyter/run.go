package jupyter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/lmorg/ttyphoon/app"
)

func runNote(ctx context.Context, id string, code string, ch chan<- *OutputT, binding *LanguageBindingT, parameters ...string) {
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, fmt.Sprintf("%s-note-*.%s", app.DirName, binding.FileExtension))
	if err != nil {
		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error creating temp file: %v", err),
			IsErr:  true,
		}
		return
	}
	defer os.Remove(tempFile.Name())

	buf := bytes.NewBuffer([]byte{})
	tmpl, err := template.New(id).Funcs(templateFuncs(code)).Parse(binding.Template)
	if err != nil {
		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error writing temp file: %v", err),
			IsErr:  true,
		}
		return
	}
	err = tmpl.Execute(buf, map[string]string{"Code": code})
	if err != nil {
		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error writing temp file: %v", err),
			IsErr:  true,
		}
		return
	}

	_, err = tempFile.Write(buf.Bytes())
	tempFile.Close()
	if err != nil {
		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error writing temp file: %v", err),
			IsErr:  true,
		}
		return
	}

	pre1 := expandVars(binding.PreRunCommand, tempFile.Name())
	pre2 := expandVars(binding.PreRunCommand, tempFile.Name())
	var exe []string
	if id == _ID_FUNCTION {
		exe = expandParameters(binding.ExecuteParameters, tempFile.Name(), parameters)
	} else {
		exe = expandVars(binding.ExecuteCommand, tempFile.Name())
	}

	var exitCode int
	if len(pre1) > 0 {
		exitCode = execute(ctx, id, pre1, ch)
	}
	if len(pre2) > 0 && exitCode == 0 {
		exitCode = execute(ctx, id, pre2, ch)
	}
	if exitCode == 0 {
		_ = execute(ctx, id, exe, ch)
	}

	time.Sleep(500 * time.Millisecond) // just to avoid any chance of the channel closing before the output has finished being flushed
	close(ch)
}
