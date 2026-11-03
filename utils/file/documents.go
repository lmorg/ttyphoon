package file

import (
	"path/filepath"

	"github.com/lmorg/ttyphoon/app"
)

func DocumentsDir(home string) string {
	return filepath.Join(home, "Documents", app.DirName)
}
