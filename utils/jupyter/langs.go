package jupyter

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/lmorg/ttyphoon/app"
)

type LanguageBindingT struct {
	Aliases        []string `yaml:"Aliases"`
	Description    string   `yaml:"Description"`
	Template       string   `yaml:"Template"`
	FileExtension  string   `yaml:"FileExtension"`  // Must exclude `.` prefix
	PreRunCommand  []string `yaml:"PreRunCommand"`  // `$FILE` is replaced with the filename
	PreRunCommand2 []string `yaml:"PreRunCommand2"` // `$FILE` is replaced with the filename
	ExecuteCommand []string `yaml:"ExecuteCommand"` // `$FILE` is replace with the filename
}

var Languages []*LanguageBindingT

type OutputT struct {
	Id     string
	Output string
	IsErr  bool
}

func GetLanguageDescriptions(language string) []string {
	language = strings.ToLower(language)

	var descriptions []string
	for _, binding := range Languages {
		if slices.Contains(binding.Aliases, language) {
			descriptions = append(descriptions, binding.Description)
		}
	}

	return descriptions
}

func RunNote(ctx context.Context, id, code, langRuntime string, ch chan<- *OutputT) {
	for _, binding := range Languages {
		if binding.Description != langRuntime {
			continue
		}

		runNote(ctx, id, code, ch, binding)
		return
	}

	ch <- &OutputT{
		Id:     id,
		Output: fmt.Sprintf("Unsupported language: %s", langRuntime),
		IsErr:  true,
	}
}

func runNote(ctx context.Context, id string, code string, ch chan<- *OutputT, binding *LanguageBindingT) {
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

	pre := expandVars(binding.PreRunCommand, tempFile.Name())
	pre2 := expandVars(binding.PreRunCommand, tempFile.Name())
	exe := expandVars(binding.ExecuteCommand, tempFile.Name())

	var exitCode int
	if len(pre) > 0 {
		exitCode = execute(ctx, id, pre, ch)
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

func templateFuncs(code string) template.FuncMap {
	return template.FuncMap{
		"begins":   func(s string) bool { return strings.HasPrefix(code, s) },
		"contains": func(s string) bool { return strings.Contains(code, s) },
		"ends":     func(s string) bool { return strings.HasSuffix(code, s) },
	}
}

func expandVars(slice []string, filename string) []string {
	s := slices.Clone(slice)
	dir := filepath.Dir(filename)

	for i := range s {
		s[i] = os.Expand(s[i], func(val string) string {
			switch val {
			case "FILE":
				return filename
			case "DIR":
				return dir
			default:
				return val
			}
		})
	}

	return s
}

func execute(ctx context.Context, id string, argv []string, ch chan<- *OutputT) int {
	select {
	case <-ctx.Done():
		return 1
	default:
	}

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error getting stdout: %v", err),
			IsErr:  true,
		}
		return 1
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error getting stderr: %v", err),
			IsErr:  true,
		}
		return 1
	}

	err = cmd.Start()
	if err != nil {
		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error starting command: %v", err),
			IsErr:  true,
		}
		return 1
	}

	go readAndEmit(id, ch, stdout, false)
	go readAndEmit(id, ch, stderr, true)

	err = cmd.Wait()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			ch <- &OutputT{
				Id:     id,
				Output: fmt.Sprintf("Error starting command: %v", err),
				IsErr:  true,
			}
		}
	}

	return cmd.ProcessState.ExitCode()
}

func readAndEmit(id string, ch chan<- *OutputT, reader io.Reader, isStderr bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		ch <- &OutputT{
			Id:     id,
			Output: line,
			IsErr:  isStderr,
		}
	}

	if err := scanner.Err(); err != nil {
		if strings.Contains(err.Error(), "file already closed") {
			return
		}

		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error reading output: %v", err),
			IsErr:  true,
		}
	}
}
