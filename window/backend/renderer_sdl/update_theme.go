package rendersdl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/utils/themes/iterm2"
)

const _ITERMCOLORS_EXT = ".itermcolors"

func (sr *sdlRender) updateThemeMenu() {
	home, err := os.UserHomeDir()
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	files, err := filepath.Glob(home + "/*" + _ITERMCOLORS_EXT)
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	fnHighlight := func(i int) {
		err := iterm2.GetTheme(files[i])
		if err != nil {
			sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
		sr.cacheBgTexture.Destroy(sr)
		filename := files[i]
		if strings.HasPrefix(files[i], home) {
			filename = "~" + files[i][len(home):]
		}
		filename = strings.TrimSuffix(filename, _ITERMCOLORS_EXT)
		config.Config.Terminal.ColorTheme = filename
		updateBlendMode()
	}

	fnSelect := func(int) {
		sr.fontCache.Reallocate()
		sr.UpdateConfig()
	}

	sr.DisplayMenu("Settings > Select a theme", files, fnHighlight, fnSelect, fnSelect)
}
