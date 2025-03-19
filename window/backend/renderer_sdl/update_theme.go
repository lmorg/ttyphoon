package rendersdl

import (
	"os"
	"path/filepath"

	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/utils/themes/iterm2"
)

func (sr *sdlRender) UpdateTheme() {
	path, err := os.UserHomeDir()
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	files, err := filepath.Glob(path + "/*.itermcolors")
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	fnSelect := func(i int) {
		err := iterm2.GetTheme(files[i])
		if err != nil {
			sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
		sr.cacheBgTexture = nil
	}

	sr.DisplayMenu("Select a theme", files, fnSelect, nil, nil)
}
