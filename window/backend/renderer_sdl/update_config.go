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
			Fn:    sr.updateThemeMenu,
			Icon:  0xf53f,
		},
		{
			Title: fmt.Sprintf("%s = %v", "Terminal.AutoHotlink", config.Config.Terminal.AutoHotlink),
			Fn: func() {
				config.Config.Terminal.AutoHotlink = !config.Config.Terminal.AutoHotlink
				sr.UpdateConfig()
			},
			Icon: 0xf0c1,
		},
		/*{
			Title: fmt.Sprintf("%s = %v", "Terminal.Widgets.AutoHotlink.IncLineNumbers", config.Config.Terminal.Widgets.AutoHotlink.IncLineNumbers),
			Fn: func() {
				config.Config.Terminal.Widgets.AutoHotlink.IncLineNumbers = !config.Config.Terminal.Widgets.AutoHotlink.IncLineNumbers
				sr.UpdateConfig()
			},
			//Icon: 0xf0c1,
		},*/

		{
			Title: fmt.Sprintf("%s = %v", "TypeFace.DropShadow", config.Config.TypeFace.DropShadow),
			Fn: func() {
				config.Config.TypeFace.DropShadow = !config.Config.TypeFace.DropShadow
				sr.fontCache.Reallocate()
				sr.UpdateConfig()
			},
			Icon: 0xf12c,
		},
		{
			Title: fmt.Sprintf("%s = %v", "TypeFace.Ligatures", config.Config.TypeFace.Ligatures),
			Fn: func() {
				config.Config.TypeFace.Ligatures = !config.Config.TypeFace.Ligatures
				sr.UpdateConfig()
			},
			Icon: 0xf035,
		},

		{
			Title: fmt.Sprintf("%s = %v", "Window.StatusBar", config.Config.Window.StatusBar),
			Fn: func() {
				config.Config.Window.StatusBar = !config.Config.Window.StatusBar
				sr.initFooter()
				sr.UpdateConfig()
			},
			Icon: 0xe59a,
		},
		{
			Title: fmt.Sprintf("%s = %v", "Window.TabBarFrame", config.Config.Window.TabBarFrame),
			Fn: func() {
				config.Config.Window.TabBarFrame = !config.Config.Window.TabBarFrame
				sr.initFooter()
				sr.UpdateConfig()
			},
			//Icon: 0xe59a,
		},
		{
			Title: fmt.Sprintf("%s = %v", "Window.TabBarActiveHighlight", config.Config.Window.TabBarActiveHighlight),
			Fn: func() {
				config.Config.Window.TabBarActiveHighlight = !config.Config.Window.TabBarActiveHighlight
				sr.initFooter()
				sr.UpdateConfig()
			},
			//Icon: 0xe59a,
		},
		{
			Title: fmt.Sprintf("%s = %v", "Window.TabBarHoverHighlight", config.Config.Window.TabBarHoverHighlight),
			Fn: func() {
				config.Config.Window.TabBarHoverHighlight = !config.Config.Window.TabBarHoverHighlight
				sr.initFooter()
				sr.UpdateConfig()
			},
			//Icon: 0xe59a,
		},
		{
			Title: fmt.Sprintf("%s = %v", "Window.TileHighlightFill", config.Config.Window.TileHighlightFill),
			Fn: func() {
				config.Config.Window.TileHighlightFill = !config.Config.Window.TileHighlightFill
				sr.initFooter()
				sr.UpdateConfig()
			},
			//Icon: 0xe59a,
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
