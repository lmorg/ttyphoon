package jupyter

import (
	"context"
	"errors"
	"fmt"
	"os"
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

func RunNote(ctx context.Context, id, pwd, code, langRuntime string, ch chan<- *OutputT) {
	for _, binding := range Languages {
		if binding.Description != langRuntime {
			continue
		}

		runNote(ctx, id, pwd, code, ch, binding)
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

func RunFunction(ctx context.Context, pwd, code string, parameters []string, langRuntime string) (string, error) {
	for _, binding := range Languages {
		if binding.Description != langRuntime {
			continue
		}

		var (
			ch  = make(chan *OutputT)
			out string
			err string
		)

		go runNote(ctx, _ID_FUNCTION, pwd, code, ch, binding, parameters...)

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
