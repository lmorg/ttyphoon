package jupyter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/lmorg/murex/utils/lists"
)

type LanguageBindingT struct {
	Aliases           []string `yaml:"Aliases"`
	PathRegexp        string   `yaml:"PathRegexp"` // If an alias is ambiguous, you can also match based on a regex pattern
	pathRegexp        *regexp.Regexp
	Description       string   `yaml:"Description"`
	Template          string   `yaml:"Template"`
	FileExtension     string   `yaml:"FileExtension"`     // Must exclude `.` prefix
	FormatCommand     []string `yaml:"FormatCommand"`     // `$FILE` is replaced with the filename
	BuildCommand      []string `yaml:"BuildCommand"`      // `$FILE` is replaced with the filename
	BuildCommand2     []string `yaml:"BuildCommand2"`     // `$FILE` is replaced with the filename
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

func RunNote(ctx context.Context, bookId, codeId, pwd, code, langRuntime string, ch chan *OutputT) {
	binding := getBindingByDescription(langRuntime)
	if binding != nil {
		runNote(ctx, bookId, codeId, pwd, code, ch, binding)
		return
	}

	ch <- &OutputT{
		Id:     codeId,
		Output: fmt.Sprintf("Unsupported language: %s", langRuntime),
		IsErr:  true,
	}
}

const _PARAMETERS = "${PARAMETERS}"

func RunFunction(ctx context.Context, pwd, bookId, functionName, code string, parameters []string, langRuntime string) (string, error) {
	binding := getBindingByDescription(langRuntime)
	if binding != nil {
		var (
			ch  = make(chan *OutputT)
			out string
			err string
		)

		go runNote(ctx, bookId, functionName, pwd, code, ch, binding, parameters...)

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

func getBindingByDescription(langRuntime string) *LanguageBindingT {
	for _, binding := range Languages {
		if binding.Description == langRuntime {
			return binding
		}
	}

	return nil
}

func getBindingByAlias(lang string) *LanguageBindingT {
	for _, binding := range Languages {
		if lists.Match(binding.Aliases, lang) {
			return binding
		}
	}

	return nil
}

// resolveFormatBinding resolves a language binding for formatting. The language
// may be a runtime description (e.g. "Go (Golang)"), an alias (e.g. "go"), or a
// file extension (e.g. "go"). Returns nil when nothing matches.
func resolveFormatBinding(language string) *LanguageBindingT {
	language = strings.TrimSpace(language)
	if language == "" {
		return nil
	}
	if binding := getBindingByDescription(language); binding != nil {
		return binding
	}
	if binding := getBindingByAlias(language); binding != nil {
		return binding
	}
	return getBindingByExtension(language)
}

// FileExtensionForLanguage resolves the file extension (without leading dot)
// for a language runtime description, alias, or extension. Returns an empty
// string when nothing matches.
func FileExtensionForLanguage(language string) string {
	binding := resolveFormatBinding(language)
	if binding == nil {
		return ""
	}
	return strings.TrimPrefix(binding.FileExtension, ".")
}

func normalizeLanguage(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func canonicalAlias(binding *LanguageBindingT) string {
	if binding == nil {
		return ""
	}

	for _, alias := range binding.Aliases {
		if normalized := normalizeLanguage(alias); normalized != "" {
			return normalized
		}
	}

	return ""
}

func getBindingByExtension(ext string) *LanguageBindingT {
	ext = strings.TrimPrefix(normalizeLanguage(ext), ".")
	if ext == "" {
		return nil
	}

	for _, binding := range Languages {
		if binding.pathRegexp != nil {
			continue
		}
		if strings.EqualFold(strings.TrimPrefix(binding.FileExtension, "."), ext) {
			return binding
		}
	}

	return nil
}

// ResolveLanguageAlias returns a canonical language alias for a runtime name,
// alias, or file extension. Returns an empty string when no mapping exists.
func ResolveLanguageAlias(language, filePath string) string {
	if binding := getBindingByAlias(language); binding != nil {
		return canonicalAlias(binding)
	}

	for _, binding := range Languages {
		if binding.pathRegexp != nil {
			if binding.pathRegexp.MatchString(filePath) {
				return canonicalAlias(binding)
			} else {
				continue
			}
		}
		if strings.EqualFold(binding.Description, strings.TrimSpace(language)) {
			return canonicalAlias(binding)
		}
	}

	if binding := getBindingByExtension(language); binding != nil {
		return canonicalAlias(binding)
	}

	return ""
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
