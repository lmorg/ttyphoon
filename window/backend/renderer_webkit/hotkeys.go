package rendererwebkit

import (
	"fmt"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/hotkeys"
	"github.com/lmorg/ttyphoon/types"
)

func (wr *webkitRender) hotkeys() {
	var (
		conf = config.Config.Hotkeys.Functions.Scan()
		fn   func()
		desc string
	)

	for _, hk := range conf {
		var icon rune

		switch hk.Function {
		case "CommandPalette":
			fn = func() { wr.ShowCommandPalette() }
			desc = "Open command palette"
		case "Settings":
			fn = func() { wr.UpdateConfig() }
			desc = "Settings..."

		case "ReloadSettings":
			fn = func() {
				if err := config.LoadConfig(); err != nil {
					wr.DisplayNotification(types.NOTIFY_DEBUG, err.Error())
				}
				wr.EmitStyleUpdate()
				wr.DisplayNotification(types.NOTIFY_INFO, "Settings have been reloaded from disk")
			}
			desc = "Reload settings from disk"

		case "Paste":
			fn = func() { wr.clipboardPaste() }
			desc = "Paste from clipboard"
			icon = 0xf0ea
		case "VisualEditor":
			fn = func() { wr.VisualEditor() }
			desc = "Visual editor..."
			icon = 0xf11c

		case "AskAI":
			fn = func() { askAi(wr) }
			desc = "Ask AI..."
			icon = 0xf544
		case "AgentSkills":
			fn = func() { askAiSkills(wr) }
			desc = "Ask AI with Agent Skill..."
			icon = 0xf544

		case "SearchRegex":
			fn = func() { wr.termWin.Active.GetTerm().Search(types.SEARCH_REGEX) }
			desc = "Search terminal output for regex match (find)..."
			icon = 0xf002
		case "SearchResults":
			fn = func() { wr.termWin.Active.GetTerm().Search(types.SEARCH_RESULTS) }
			desc = "View search results..."
			icon = 0xe521
		case "SearchClear":
			fn = func() { wr.termWin.Active.GetTerm().Search(types.SEARCH_CLEAR) }
			desc = "Clear search results"
			icon = 0xf010
		case "SearchCommandLines":
			fn = func() { wr.termWin.Active.GetTerm().Search(types.SEARCH_CMD_LINES) }
			desc = "Search command line history..."
			icon = 0xf002

			/*case "OpenHistory":
			fn = func() { wr.OpenHistory(sr.termWin.Active) }
			desc = "Open history..."*/
		case "ShowHideNotes":
			fn = func() { wr.toggleNotesPane() }
			desc = "Show / hide notes..."
			icon = 0xf518
		case "OpenFile":
			fn = func() {
				if app, ok := wr.app.(interface{ ViewFileInNotes() }); ok {
					app.ViewFileInNotes()
				}
			}
			desc = "Open file..."
			icon = 0xf07c

		default:
			wr.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("unknown hotkey function: '%s'", hk.Function))
			continue
		}
		hotkeys.Add(hk.Prefix, hk.Hotkey, fn, desc, icon)

	}
}
