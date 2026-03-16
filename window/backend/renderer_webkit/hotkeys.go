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

		switch hk.Function {
		/*case "Settings":
			fn = func() { sr.UpdateConfig() }
			desc = "Settings..."
		case "ReloadSettings":
			fn = func() {
				if err := config.LoadConfig(); err != nil {
					sr.DisplayNotification(types.NOTIFY_DEBUG, err.Error())
				}
				sr.initFooter()
				updateBlendMode()
				sr.fontCache.Reallocate()
				sr.cacheBgTexture.Destroy(sr)
				sr.DisplayNotification(types.NOTIFY_INFO, "Settings have been reloaded from disk")
			}
			desc = "Reload settings from disk"*/

		case "Paste":
			fn = func() { wr.clipboardPaste() }
			desc = "Paste from clipboard"
		/*case "VisualEditor":
		fn = func() { wr.VisualEditor() }
		desc = "Visual editor..."*/

		case "AskAI":
			fn = func() { askAi(wr) }
			desc = "Ask AI..."
		case "AgentSkills":
			fn = func() { askAiSkill(wr) }
			desc = "Ask AI with Agent Skill..."

		case "SearchRegex":
			fn = func() { wr.termWin.Active.GetTerm().Search(types.SEARCH_REGEX) }
			desc = "Search terminal output for regex match..."
		case "SearchResults":
			fn = func() { wr.termWin.Active.GetTerm().Search(types.SEARCH_RESULTS) }
			desc = "View search results..."
		case "SearchClear":
			fn = func() { wr.termWin.Active.GetTerm().Search(types.SEARCH_CLEAR) }
			desc = "Clear search results"
		case "SearchAIPrompts":
			fn = func() { wr.termWin.Active.GetTerm().Search(types.SEARCH_AI_PROMPTS) }
			desc = "Search AI prompts..."
		case "SearchCommandLines":
			fn = func() { wr.termWin.Active.GetTerm().Search(types.SEARCH_CMD_LINES) }
			desc = "Search command line history..."
		/*case "OpenHistory":
			fn = func() { wr.OpenHistory(sr.termWin.Active) }
			desc = "Open history..."
		case "OpenNotes":
			fn = func() { wr.openNotes() }
			desc = "Open notes..."*/

		default:
			wr.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("unknown hotkey function: '%s'", hk.Function))
			continue
		}
		hotkeys.Add(hk.Prefix, hk.Hotkey, fn, desc)

	}
}
