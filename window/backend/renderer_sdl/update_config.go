package rendersdl

import (
	"fmt"

	"github.com/lmorg/mxtty/ai"
	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/integrations"
	"github.com/lmorg/mxtty/types"
)

func (sr *sdlRender) UpdateConfig() {
	tile := sr.termWin.Active
	meta := agent.Get(tile.Id())
	meta.Renderer = sr
	meta.Term = tile.GetTerm()
	meta.Pwd = tile.Pwd()

	menu := sr.NewContextMenu()
	menu.Append([]types.MenuItem{
		{
			Title: fmt.Sprintf("%s == %v", "Terminal.ColorTheme", config.Config.Terminal.ColorTheme),
			Fn:    sr.updateThemeMenu,
			Icon:  0xf53f,
		},
		{
			Title: fmt.Sprintf("%s == %v", "Terminal.AutoHotlink", config.Config.Terminal.AutoHotlink),
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
			Title: fmt.Sprintf("%s == %v", "TypeFace.DropShadow", config.Config.TypeFace.DropShadow),
			Fn: func() {
				config.Config.TypeFace.DropShadow = !config.Config.TypeFace.DropShadow
				sr.fontCache.Reallocate()
				sr.UpdateConfig()
			},
			Icon: 0xf12c,
		},
		{
			Title: fmt.Sprintf("%s == %v", "TypeFace.Ligatures", config.Config.TypeFace.Ligatures),
			Fn: func() {
				config.Config.TypeFace.Ligatures = !config.Config.TypeFace.Ligatures
				sr.UpdateConfig()
			},
			Icon: 0xf035,
		},

		{
			Title: fmt.Sprintf("%s == %v", "Window.BellVisualNotification", config.Config.Window.BellVisualNotification),
			Fn: func() {
				config.Config.Window.BellVisualNotification = !config.Config.Window.BellVisualNotification
				sr.UpdateConfig()
			},
			Icon: 0xf0f3,
		},
		{
			Title: fmt.Sprintf("%s == %v", "Window.BellPlayAudio", config.Config.Window.BellPlayAudio),
			Fn: func() {
				config.Config.Window.BellPlayAudio = !config.Config.Window.BellPlayAudio
				sr.UpdateConfig()
			},
			Icon: 0xf0a1,
		},

		{
			Title: fmt.Sprintf("%s == %v", "Window.HoverEffectHighlight", config.Config.Window.HoverEffectHighlight),
			Fn: func() {
				config.Config.Window.HoverEffectHighlight = !config.Config.Window.HoverEffectHighlight
				sr.initFooter()
				sr.UpdateConfig()
			},
			Icon: 0xf591,
		},

		{
			Title: fmt.Sprintf("%s == %v", "Window.StatusBar", config.Config.Window.StatusBar),
			Fn: func() {
				config.Config.Window.StatusBar = !config.Config.Window.StatusBar
				sr.initFooter()
				sr.UpdateConfig()
			},
			Icon: 0xe59a,
		},
		{
			Title: fmt.Sprintf("%s == %v", "Window.TabBarFrame", config.Config.Window.TabBarFrame),
			Fn: func() {
				config.Config.Window.TabBarFrame = !config.Config.Window.TabBarFrame
				sr.initFooter()
				sr.UpdateConfig()
			},
			//Icon: 0xe59a,
		},
		{
			Title: fmt.Sprintf("%s == %v", "Window.TabBarActiveHighlight", config.Config.Window.TabBarActiveHighlight),
			Fn: func() {
				config.Config.Window.TabBarActiveHighlight = !config.Config.Window.TabBarActiveHighlight
				sr.initFooter()
				sr.UpdateConfig()
			},
			//Icon: 0xe59a,
		},

		{
			Title: fmt.Sprintf("%s == %v", "Window.TileHighlightFill", config.Config.Window.TileHighlightFill),
			Fn: func() {
				config.Config.Window.TileHighlightFill = !config.Config.Window.TileHighlightFill
				sr.initFooter()
				sr.UpdateConfig()
			},
			Icon: 0xf009,
		},

		{
			Title: types.MENU_SEPARATOR,
		},
		{
			Title: fmt.Sprintf("AI service == %s", agent.Get(sr.termWin.Active.Id()).ServiceName()),
			Fn:    func() { meta.ServiceNext(); sr.UpdateConfig() },
			Icon:  0xe4f6,
		},
		{
			Title: fmt.Sprintf("Model == %s", agent.Get(sr.termWin.Active.Id()).ModelName()),
			Fn:    func() { meta.ModelNext(); sr.UpdateConfig() },
			Icon:  0xe699,
		},
		{
			Title: "Enable or disable specific AI tools...",
			Fn:    func() { meta.ChooseTools(func(int) { sr.UpdateConfig() }) },
			Icon:  0xf7d9,
		},
		{
			Title: "Start MCP servers...",
			Fn:    func() { ai.StartMcp(sr, meta, func(int) { sr.UpdateConfig() }) },
			Icon:  0xf552,
		},
		{
			Title: "Set Anthropic (Claude) API Key",
			Fn:    func() { ai.EnvAnthropic(sr, sr.UpdateConfig) },
			Icon:  0xf084,
		},
		{
			Title: "Set OpenAI (ChatGPT) API Key",
			Fn:    func() { ai.EnvOpenAi(sr, sr.UpdateConfig) },
			Icon:  0xf084,
		},

		{
			Title: types.MENU_SEPARATOR,
		},
		{
			Title: "Bash integration (written to shell)",
			Fn:    func() { tile.GetTerm().Reply(integrations.Get("shell.bash")) },
			Icon:  0xf120,
		},
		{
			Title: "Zsh integration (written to shell)",
			Fn:    func() { tile.GetTerm().Reply(integrations.Get("shell.zsh")) },
			Icon:  0xf120,
		},
	}...)

	sr.displayMenuWithIcons("Settings", menu.Options(), menu.Icons(), nil, menu.Callback, nil)
}
