package filetools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

func resolveWorkspacePath(agt aitypes.Agent, name string) (string, error) {
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

	relative, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside the project root", name)
	}

	return path, nil
}
