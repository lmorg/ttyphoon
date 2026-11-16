// Package envvars resolves feature-scoped environment variables.
package envvars

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
)

// Environment holds only the values explicitly configured for a feature.
type Environment struct {
	values map[string]string
}

// Resolve reads the configured files in order, then applies explicit values.
// Relative files are read from every matching application config directory.
func Resolve(envFiles []string, envs map[string]string) (*Environment, error) {
	return resolve(envFiles, envs, xdg.ConfigDirs)
}

// Lookup returns the scoped value when present, otherwise the host value.
func (e *Environment) Lookup(name string) (string, bool) {
	if value, ok := e.values[name]; ok {
		return value, true
	}
	return os.LookupEnv(name)
}

// Value returns the resolved value, or an empty string when it is absent.
func (e *Environment) Value(name string) string {
	value, _ := e.Lookup(name)
	return value
}

// Overlay returns a copy of base with scoped values applied. Passing nil
// returns only the scoped values.
func (e *Environment) Overlay(base map[string]string) map[string]string {
	result := make(map[string]string, len(base)+len(e.values))
	maps.Copy(result, base)
	maps.Copy(result, e.values)
	return result
}

// Redactor returns a function that removes scoped values from logs or errors.
func (e *Environment) Redactor() func(string) string {
	values := make([]string, 0, len(e.values))
	for _, value := range e.values {
		if value != "" {
			values = append(values, value)
		}
	}
	slices.SortFunc(values, func(a, b string) int { return len(b) - len(a) })
	replacements := make([]string, 0, 2*len(values))
	for _, value := range values {
		replacements = append(replacements, value, "[REDACTED]")
	}
	return strings.NewReplacer(replacements...).Replace
}

func resolve(envFiles []string, envs map[string]string, configDirs []string) (*Environment, error) {
	values := make(map[string]string)
	for _, entry := range envFiles {
		if filepath.IsAbs(entry) {
			if err := readFile(entry, values); err != nil {
				return nil, err
			}
			continue
		}
		if !filepath.IsLocal(entry) {
			return nil, fmt.Errorf("invalid relative environment file path %q", entry)
		}
		found := false
		for _, dir := range configDirs {
			root := filepath.Join(dir, app.DirName)
			path := filepath.Join(root, entry)
			_, err := os.Lstat(path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("inspect environment file %q: %w", path, err)
			}
			found = true
			if err := checkContained(root, path); err != nil {
				return nil, err
			}
			if err := readFile(path, values); err != nil {
				return nil, err
			}
		}
		if !found {
			return nil, fmt.Errorf("environment file %q not found in application config directories", entry)
		}
	}
	for name, value := range envs {
		if !validName(name) || strings.ContainsRune(value, 0) {
			return nil, errors.New("invalid explicit environment variable")
		}
		values[name] = value
	}
	return &Environment{values: values}, nil
}

func checkContained(root, path string) error {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve environment config directory %q: %w", root, err)
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve environment file %q: %w", path, err)
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || !filepath.IsLocal(relative) {
		return fmt.Errorf("environment file %q escapes its application config directory", path)
	}
	return nil
}

func readFile(path string, values map[string]string) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open environment file %q: %w", path, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close environment file %q: %w", path, closeErr))
		}
	}()

	reader := bufio.NewReader(f)
	for lineNumber := 1; ; lineNumber++ {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return fmt.Errorf("read environment file %q: %w", path, readErr)
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			name, value, ok := strings.Cut(line, "=")
			if !ok || !validName(name) || strings.ContainsRune(value, 0) {
				return fmt.Errorf("invalid environment file %q at line %d", path, lineNumber)
			}
			values[name] = value
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
	}
}

func validName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}
