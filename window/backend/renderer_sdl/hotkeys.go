package rendersdl

import (
	"fmt"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/hotkeys"
	"github.com/lmorg/ttyphoon/types"
	"golang.design/x/hotkey"
)

func (sr *sdlRender) _registerHotkey() {
	sr.hk = hotkey.New([]hotkey.Modifier{}, hotkey.KeyF12)
	err := sr.hk.Register()
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Unable to set hotkey: %s", err.Error()))
	}
}

func (sr *sdlRender) pollEventHotkey() <-chan hotkey.Event {
	if sr.hk != nil {
		return sr.hk.Keydown()
	}
	return nil
}

func (sr *sdlRender) eventHotkey() {
	if sr.hkToggle {
		sr.hideWindow()
	} else {
		sr.ShowAndFocusWindow()
	}
	sr.hkToggle = !sr.hkToggle
}

func (sr *sdlRender) hotkeys() {
	var (
		conf = config.Config.Hotkeys.Functions.Scan()
		fn   func()
		desc string
	)

	for _, hk := range conf {

		switch hk.Function {
		case "Settings":
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
			desc = "Reload settings from disk"

		case "Paste":
			fn = func() { sr.clipboardPaste() }
			desc = "Paste from clipboard"
		case "VisualEditor":
			fn = func() { sr.visualEditor() }
			desc = "Visual editor..."

		case "AskAI":
			fn = func() { askAi(sr, &types.XY{Y: sr.termWin.Active.GetTerm().GetSize().Y - 1}) }
			desc = "Ask AI..."
		case "AgentSkills":
			fn = func() { askAiSkill(sr, &types.XY{Y: sr.termWin.Active.GetTerm().GetSize().Y - 1}) }
			desc = "Ask AI with Agent Skill..."

		case "SearchRegex":
			fn = func() { sr.termWin.Active.GetTerm().Search(types.SEARCH_REGEX) }
			desc = "Search terminal output for regex match..."
		case "SearchResults":
			fn = func() { sr.termWin.Active.GetTerm().Search(types.SEARCH_RESULTS) }
			desc = "View search results..."
		case "SearchClear":
			fn = func() { sr.termWin.Active.GetTerm().Search(types.SEARCH_CLEAR) }
			desc = "Clear search results"
		case "SearchAIPrompts":
			fn = func() { sr.termWin.Active.GetTerm().Search(types.SEARCH_AI_PROMPTS) }
			desc = "Search AI prompts..."
		case "SearchCommandLines":
			fn = func() { sr.termWin.Active.GetTerm().Search(types.SEARCH_CMD_LINES) }
			desc = "Search command line history..."
		case "OpenHistory":
			fn = func() { Open(sr, sr.termWin.Active) }
			desc = "Open history..."

		default:
			sr.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("unknown hotkey function: '%s'", hk.Function))
			continue
		}
		hotkeys.Add(hk.Prefix, hk.Hotkey, fn, desc)

	}
}
