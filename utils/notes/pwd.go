package notes

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
)

const _FUNCTION = "notes"

func DirGlobal() string {
	path := fmt.Sprintf("%s/Documents/%s/%s/", xdg.Home, app.DirName, _FUNCTION)

	/*err :=*/
	_ = os.MkdirAll(path, 0700)
	/*if err != nil {
		return err
	}*/

	return path
}

func DirProjectRoot(cwd string) string {
	home, _ := os.UserHomeDir()
	path := dirProjectRoot(cwd, home)

	if path == home {
		path = xdg.Home + "/Documents/"
	}

	return path
}

// DirProjectRoot finds the root of a git project by searching for a .git directory
// starting from cwd and moving up the directory tree.
// Returns:
// - The directory containing .git if found
// - The original cwd if root or home directory is reached without finding .git
// - Empty string if os.Getwd() fails when cwd is empty
func dirProjectRoot(cwd string, home string) string {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return ""
		}
	}

	current := cwd

	for {
		// Check if current directory contains .git
		if hasGitDirectory(current) || hasProjectConfig(current) {
			return current
		}

		// Move to parent directory
		parent := filepath.Dir(current)

		// Check stopping conditions
		if parent == current {
			// Reached filesystem root (Dir("/") returns "/")
			return cwd
		}
		if current == home {
			// Reached home directory without finding .git
			return cwd
		}

		// Move up one level
		current = parent
	}
}

// hasGitDirectory checks if a directory contains a .git subdirectory
func hasGitDirectory(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func hasProjectConfig(path string) bool {
	gitPath := filepath.Join(path, "project.ttyphoon")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}
