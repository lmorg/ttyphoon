package rendersdl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/utils/themes/iterm2"
	"github.com/veandco/go-sdl2/sdl"
)

const _ITERMCOLORS_EXT = ".itermcolors"

func updateBlendMode() {
	//textShadow[_HLTEXTURE_NONE].Alpha = types.COLOR_TEXT_SHADOW.Alpha

	if types.THEME_LIGHT {
		highlightBlendMode = sdl.BLENDMODE_BLEND
		notifyColour, notifyBorderColour = _notifyColourLight, _notifyColourDark
		//questionColor, questionColorBorder = _questionColorLight, types.SGR_COLOR_BLACK //_questionColorDark

	} else {
		highlightBlendMode = sdl.BLENDMODE_ADD
		notifyColour, notifyBorderColour = _notifyColourDark, _notifyColourLight
		//questionColor, questionColorBorder = _questionColorDark, types.SGR_COLOR_BLACK //_questionColorLight
	}
}

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
		updateBlendMode()
		sr.fontCache.Reallocate()
		sr.cacheBgTexture.Destroy(sr)

		filename := files[i]
		if strings.HasPrefix(files[i], home) {
			filename = "~" + files[i][len(home):]
		}
		filename = strings.TrimSuffix(filename, _ITERMCOLORS_EXT)
		config.Config.Terminal.ColorTheme = filename
	}

	fnSelect := func(int) {
		sr.fontCache.Reallocate()
		sr.UpdateConfig()
	}

	sr.DisplayMenu("Settings > Select a theme", files, fnHighlight, fnSelect, fnSelect)
}
