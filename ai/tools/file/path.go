package filetools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

// resolveWorkspacePath resolves name to an absolute path, rejecting anything
// outside the project root.
func resolveWorkspacePath(agt aitypes.Agent, name string) (string, error) {
	return resolvePath(agt, name, true)
}

// resolveAnyPath resolves name to an absolute path without the project-root
// restriction, for read-only tools that need to browse outside the workspace
// (e.g. the images this agent itself generates under ~/<app>/.images).
func resolveAnyPath(agt aitypes.Agent, name string) (string, error) {
	return resolvePath(agt, name, false)
}

func resolvePath(agt aitypes.Agent, name string, restrictToRoot bool) (string, error) {
	pwd := agt.ProjectRoot()
	root, err := filepath.Abs(filepath.Clean(pwd))
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	path := name
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err = filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(path)
		resolvedParent, parentErr := filepath.EvalSymlinks(parent)
		if parentErr != nil {
			return "", parentErr
		}
		resolvedPath = filepath.Join(resolvedParent, filepath.Base(path))
	}

	if !restrictToRoot {
		return path, nil
	}

	relative, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside the project root", name)
	}

	return path, nil
}
