package rendererwebkit

import (
	"fmt"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/skills"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/hotkeys"
	"github.com/lmorg/ttyphoon/integrations"
	"github.com/lmorg/ttyphoon/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type commandPaletteEvent struct {
	Title   string                 `json:"title"`
	Options []commandPaletteOption `json:"options"`
}

type commandPaletteOption struct {
	Title     string `json:"title"`
	Icon      rune   `json:"icon"`
	Separator bool   `json:"separator"`
}

func (wr *webkitRender) commandPaletteItems() []types.MenuItem {
	if wr.termWin == nil || wr.termWin.Active == nil {
		return nil
	}

	tile := wr.termWin.Active
	meta := agent.Get(tile.Id())

	menu := []types.MenuItem{
		{
			Title: "Application settings...",
			Fn:    wr.UpdateConfig,
			Icon:  0xf013,
		},
		{
			Title: fmt.Sprintf("Change theme (currently %s)...", config.Config.Terminal.ColorTheme),
			Fn:    wr.updateThemeMenu,
			Icon:  0xf53f,
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
	}

	// terminal tabs:

	menu = append(menu, types.MenuItem{Title: types.MENU_SEPARATOR})
	for _, tab := range wr.tmuxTabs() {
		menu = append(menu, types.MenuItem{
			Title: fmt.Sprintf("Switch to terminal tab: %s (%s)", tab.Name, tab.ID),
			Fn:    func() { _ = wr.tmux.SelectAndResizeWindow(tab.ID, wr.windowCells) },
			Icon:  0xf2d2,
		})
	}
	for _, tab := range wr.auxTerminalTabs {
		menu = append(menu, types.MenuItem{
			Title: fmt.Sprintf("Switch to terminal tab: %s", tab.Name, tab.ID),
			Fn:    func() { _ = wr.tmux.SelectAndResizeWindow(tab.ID, wr.windowCells) },
			Icon:  0xf2d2,
		})
	}

	// 0xf2d2 // tmux
	// 0xf11c // hotkey

	// AI skills

	skills := skills.ReadSkills()
	if len(skills) > 0 {
		menu = append(menu, types.MenuItem{Title: types.MENU_SEPARATOR})
	}
	for _, skill := range skills {
		menu = append(menu, types.MenuItem{
			Title: fmt.Sprintf("AI Skill /%s: %s", skill.FunctionName, skill.Description),
			Fn:    func() { askAiSkill(wr, skill) },
			Icon:  0xf544,
		})
	}

	menu = append(menu, []types.MenuItem{
		{
			Title: types.MENU_SEPARATOR,
		},
		{
			Title: fmt.Sprintf("AI: Change AI Model (currently %s)", meta.ModelName()),
			Fn:    func() { meta.SelectServiceModel(wr.UpdateConfig) },
			Icon:  0xe4f6,
		},
		{
			Title: "AI: Enable or disable specific AI tools...",
			Fn:    func() { meta.ChooseTools(func(int) { wr.UpdateConfig() }) },
			Icon:  0xf7d9,
		},
		{
			Title: "AI: Start MCP servers...",
			Fn:    func() { meta.McpMenu(func(int) { wr.UpdateConfig() }) },
			Icon:  0xf552,
		},
		{
			Title: "AI: Set Anthropic (Claude) API Key",
			Fn:    func() { ai.EnvAnthropic(wr, wr.UpdateConfig) },
			Icon:  0xf084,
		},
		{
			Title: "AI: Set OpenAI (ChatGPT) API Key",
			Fn:    func() { ai.EnvOpenAi(wr, wr.UpdateConfig) },
			Icon:  0xf084,
		},
	}...)

	// Hotkeys

	menu = append(menu, types.MenuItem{Title: types.MENU_SEPARATOR})
	for _, hk := range hotkeys.List() {
		menu = append(menu, types.MenuItem{
			Title: hk.Description,
			Fn:    func() { hotkeys.KeyPressWithPrefix(hk.Prefix, hk.Hotkey) },
			//Icon:  0xf11c,
		})
	}

	return menu
}

// ShowCommandPalette emits the frontend event that opens the command palette UI
// with all available options. Frontend filtering is done locally.
func (wr *webkitRender) ShowCommandPalette() {
	if wr.wapp == nil {
		return
	}

	items := wr.commandPaletteItems()
	wr.SetCommandPaletteItems(items)

	options := make([]commandPaletteOption, 0, len(items))
	for i := range items {
		options = append(options, commandPaletteOption{
			Title:     items[i].Title,
			Icon:      items[i].Icon,
			Separator: items[i].Title == types.MENU_SEPARATOR,
		})
	}

	runtime.EventsEmit(wr.wapp, "commandPaletteOpen", commandPaletteEvent{
		Title:   "Command Palette",
		Options: options,
	})
}

// SetCommandPaletteItems caches the current item set so that highlight/select
// callbacks can reach the Go function closures (which are not serialisable).
func (wr *webkitRender) SetCommandPaletteItems(items []types.MenuItem) {
	wr.commandPalette = &contextMenuStub{items: items, renderer: wr}
}

// CommandPaletteSelect executes the item at index and clears the session.
func (wr *webkitRender) CommandPaletteSelect(index int) {
	if wr.commandPalette == nil {
		return
	}
	wr.commandPalette.Callback(index)
	wr.commandPalette = nil
}
