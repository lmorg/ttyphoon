package jupyter

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

type LanguageBindingT struct {
	Aliases           []string `yaml:"Aliases"`
	Description       string   `yaml:"Description"`
	Template          string   `yaml:"Template"`
	FileExtension     string   `yaml:"FileExtension"`     // Must exclude `.` prefix
	PreRunCommand     []string `yaml:"PreRunCommand"`     // `$FILE` is replaced with the filename
	PreRunCommand2    []string `yaml:"PreRunCommand2"`    // `$FILE` is replaced with the filename
	ExecuteCommand    []string `yaml:"ExecuteCommand"`    // `$FILE` is replace with the filename
	ExecuteParameters []string `yaml:"ExecuteParameters"` // `$FILE` is replace with the filename
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

func GetAllLanguageDescriptions() []string {
	var descriptions []string
	seen := make(map[string]bool)

	for _, binding := range Languages {
		if !seen[binding.Description] {
			descriptions = append(descriptions, binding.Description)
			seen[binding.Description] = true
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

const _ID_FUNCTION = "#function"

const _PARAMETERS = "${PARAMETERS}"

func RunFunction(ctx context.Context, code string, parameters []string, langRuntime string) (string, error) {
	for _, binding := range Languages {
		if binding.Description != langRuntime {
			continue
		}

		var (
			ch  = make(chan *OutputT)
			out string
			err string
		)

		go runNote(ctx, _ID_FUNCTION, code, ch, binding, parameters...)

		for output := range ch {
			if output.IsErr {
				err += "\n" + output.Output
			} else {
				out += "\n" + output.Output
			}
		}

		if err != "" {
			return out, errors.New(err)
		}

		return out, nil
	}

	return "", fmt.Errorf("Unsupported language: %s", langRuntime)
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
			case "PARAMETERS":
				return _PARAMETERS
			default:
				return val
			}
		})
	}

	return s
}

func expandParameters(slice []string, filename string, parameters []string) []string {
	s := expandVars(slice, filename)

	for i := range s {
		if s[i] == _PARAMETERS {
			return slices.Replace(s, i, i+1, parameters...)
		}
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
