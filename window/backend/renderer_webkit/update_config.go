package rendererwebkit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/integrations"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/file"
	"github.com/lmorg/ttyphoon/utils/themes/iterm2"
)

const _ITERMCOLORS_EXT = ".itermcolors"

func (wr *webkitRender) UpdateConfig() {
	if wr.termWin == nil || wr.termWin.Active == nil {
		return
	}

	tile := wr.termWin.Active
	meta := agent.Get(tile.Id())

	menu := wr.NewContextMenu()
	menu.Append([]types.MenuItem{
		{
			Title: fmt.Sprintf("%s == %v", "Terminal.ColorTheme", config.Config.Terminal.ColorTheme),
			Fn:    wr.updateThemeMenu,
			Icon:  0xf53f,
		},
		{
			Title: fmt.Sprintf("%s == %v", "Terminal.AutoHyperlink", config.Config.Terminal.AutoHyperlink),
			Fn: func() {
				config.Config.Terminal.AutoHyperlink = !config.Config.Terminal.AutoHyperlink
				wr.UpdateConfig()
			},
			Icon: 0xf0c1,
		},
		{
			Title: fmt.Sprintf("%s == %v", "TypeFace.Ligatures", config.Config.TypeFace.Ligatures),
			Fn: func() {
				config.Config.TypeFace.Ligatures = !config.Config.TypeFace.Ligatures
				wr.UpdateConfig()
			},
			Icon: 0xf035,
		},

		{
			Title: fmt.Sprintf("%s == %v", "Window.BellVisualNotification", config.Config.Window.BellVisualNotification),
			Fn: func() {
				config.Config.Window.BellVisualNotification = !config.Config.Window.BellVisualNotification
				wr.UpdateConfig()
			},
			Icon: 0xf0f3,
		},
		{
			Title: fmt.Sprintf("%s == %v", "Window.BellPlayAudio", config.Config.Window.BellPlayAudio),
			Fn: func() {
				config.Config.Window.BellPlayAudio = !config.Config.Window.BellPlayAudio
				wr.UpdateConfig()
			},
			Icon: 0xf0a1,
		},

		{
			Title: fmt.Sprintf("%s == %v", "Window.HoverEffectHighlight", config.Config.Window.HoverEffectHighlight),
			Fn: func() {
				config.Config.Window.HoverEffectHighlight = !config.Config.Window.HoverEffectHighlight
				wr.UpdateConfig()
			},
			Icon: 0xf591,
		},

		{
			Title: fmt.Sprintf("%s == %v", "Window.StatusBar", config.Config.Window.StatusBar),
			Fn: func() {
				config.Config.Window.StatusBar = !config.Config.Window.StatusBar
				wr.UpdateConfig()
			},
			Icon: 0xe59a,
		},

		{
			Title: types.MENU_SEPARATOR,
		},
		{
			Title: fmt.Sprintf("AI Model == %s: %s", meta.ServiceName(), meta.ModelName()),
			Fn:    func() { meta.SelectServiceModel(wr.UpdateConfig) },
			Icon:  0xe4f6,
		},
		{
			Title: "Enable or disable specific AI tools...",
			Fn:    func() { meta.ChooseTools(func(int) { wr.UpdateConfig() }) },
			Icon:  0xf7d9,
		},
		{
			Title: "Start MCP servers...",
			Fn:    func() { meta.McpMenu(func(int) { wr.UpdateConfig() }) },
			Icon:  0xf552,
		},
		{
			Title: "Set Anthropic (Claude) API Key",
			Fn:    func() { ai.EnvAnthropic(wr, wr.UpdateConfig) },
			Icon:  0xf084,
		},
		{
			Title: "Set OpenAI (ChatGPT) API Key",
			Fn:    func() { ai.EnvOpenAi(wr, wr.UpdateConfig) },
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

	menu.DisplayMenu("Settings")
}

func (wr *webkitRender) updateThemeMenu() {
	home, err := os.UserHomeDir()
	if err != nil {
		wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	themes, err := filepath.Glob(home + "/*" + _ITERMCOLORS_EXT)
	if err != nil {
		wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	themes = append(themes, file.GetConfigFiles("themes", _ITERMCOLORS_EXT)...)

	fnHighlight := func(i int) {
		err := iterm2.GetTheme(themes[i])
		if err != nil {
			wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}

		filename := themes[i]
		if strings.HasPrefix(themes[i], home) {
			filename = "~" + themes[i][len(home):]
		}
		config.Config.Terminal.ColorTheme = filename
		wr.RefreshWindowList()
	}

	fnSelect := func(int) {
		wr.UpdateConfig()
	}

	wr.DisplayMenu("Settings > Select a theme", themes, fnHighlight, fnSelect, fnSelect)
}
