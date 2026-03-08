package jupyter

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"text/template"

	"github.com/lmorg/ttyphoon/app"
)

type LanguageBindingT struct {
	Aliases        []string `yaml:"Aliases"`
	Description    string   `yaml:"Description"`
	Template       string   `yaml:"Template"`
	FileExtension  string   `yaml:"FileExtension"`  // Must exclude `.` prefix
	PreRunCommand  []string `yaml:"PreRunCommand"`  // `$FILE` is replaced with the filename
	ExecuteCommand []string `yaml:"ExecuteCommand"` // `$FILE` is replace with the filename
}

var Languages []*LanguageBindingT

type OutputT struct {
	Id     string
	Output string
	IsErr  bool
}

func RunNote(ctx context.Context, id, code, language string, ch chan<- *OutputT) {
	language = strings.ToLower(language)

	for _, binding := range Languages {
		if slices.Contains(binding.Aliases, language) {
			runNote(ctx, id, code, ch, binding)
			return
		}
	}
	ch <- &OutputT{
		Id:     id,
		Output: fmt.Sprintf("Unsupported language: %s", language),
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
	exe := expandVars(binding.ExecuteCommand, tempFile.Name())

	if len(pre) > 0 {
		execute(ctx, id, pre, ch)
	}

	execute(ctx, id, exe, ch)
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

	for i := range s {
		s[i] = os.Expand(s[i], func(val string) string {
			switch val {
			case "FILE":
				return filename
			default:
				return val
			}
		})
	}

	return s
}

func execute(ctx context.Context, id string, argv []string, ch chan<- *OutputT) {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error getting stdout: %v", err),
			IsErr:  true,
		}
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error getting stderr: %v", err),
			IsErr:  true,
		}
		return
	}

	err = cmd.Start()
	if err != nil {
		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error starting command: %v", err),
			IsErr:  true,
		}
		return
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
}

func readAndEmit(id string, ch chan<- *OutputT, reader io.Reader, isStderr bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		prefix := ""
		if isStderr {
			prefix = "[STDERR] "
		}
		ch <- &OutputT{
			Id:     id,
			Output: prefix + line,
			IsErr:  false,
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error reading output: %v", err),
			IsErr:  true,
		}
	}
}
