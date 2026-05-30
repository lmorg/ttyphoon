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
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	pwd := cwd
	home, _ := os.UserHomeDir()
	for {
		if _, err := os.Stat(filepath.Join(cwd, ".git")); err == nil {
			return pwd
		}
		parent := filepath.Dir(cwd)
		if parent == pwd || parent == home {
			return ""
		}
		pwd = parent
	}
}
