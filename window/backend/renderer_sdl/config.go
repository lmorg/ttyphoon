package rendersdl

import (
	"fmt"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/integrations"
)

func (sr *sdlRender) UpdateConfig() {
	menu := contextMenuT{
		{
			Title: fmt.Sprintf("%s = %v", "Terminal.ColorTheme", config.Config.Terminal.ColorTheme),
			Fn:    sr.updateTheme,
			Icon:  0xf53f,
		},
		{
			Title: fmt.Sprintf("%s = %v", "Terminal.AutoHotlink", config.Config.Terminal.AutoHotlink),
			Fn:    func() { config.Config.Terminal.AutoHotlink = !config.Config.Terminal.AutoHotlink },
			Icon:  0xf0c1,
		},
		{
			Title: fmt.Sprintf("%s = %v", "Terminal.Widgets.AutoHotlink.IncLineNumbers", config.Config.Terminal.Widgets.AutoHotlink.IncLineNumbers),
			Fn: func() {
				config.Config.Terminal.Widgets.AutoHotlink.IncLineNumbers = !config.Config.Terminal.Widgets.AutoHotlink.IncLineNumbers
			},
			//Icon: 0xf0c1,
		},

		{
			Title: fmt.Sprintf("%s = %v", "TypeFace.DropShadow", config.Config.TypeFace.DropShadow),
			Fn: func() {
				config.Config.TypeFace.DropShadow = !config.Config.TypeFace.DropShadow
				sr.limiter.Lock()
				sr.fontCache = NewFontCache(sr)
				sr.limiter.Unlock()
			},
			Icon: 0xf12c,
		},
		{
			Title: fmt.Sprintf("%s = %v", "TypeFace.Ligatures", config.Config.TypeFace.Ligatures),
			Fn:    func() { config.Config.TypeFace.Ligatures = !config.Config.TypeFace.Ligatures },
			Icon:  0xf035,
		},

		{
			Title: fmt.Sprintf("%s = %v", "Window.StatusBar", config.Config.Window.StatusBar),
			Fn: func() {
				config.Config.Window.StatusBar = !config.Config.Window.StatusBar
				sr.initFooter()
			},
			Icon: 0xe59a,
		},

		{
			Title: MENU_SEPARATOR,
		},
		{
			Title: "Bash integration (written to shell)",
			Fn:    func() { sr.termWin.Active.GetTerm().Reply(integrations.Get("shell.bash")) },
			Icon:  0xf120,
		},
		{
			Title: "Zsh integration (written to shell)",
			Fn:    func() { sr.termWin.Active.GetTerm().Reply(integrations.Get("shell.zsh")) },
			Icon:  0xf120,
		},
	}
	sr.displayMenuWithIcons("Settings", menu.Options(), menu.Icons(), nil, menu.Callback, nil)
}
