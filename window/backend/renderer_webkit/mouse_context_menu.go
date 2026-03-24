package rendererwebkit

import (
	"fmt"
	"log"
	"os"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/types"
	"golang.design/x/clipboard"
)

func init() {
	err := clipboard.Init()
	if err != nil {
		log.Println(err)
	}
}

func (wr *webkitRender) showRightClickContextMenu(_ *types.XY, _ bool) {
	if wr.termWin == nil || wr.termWin.Active == nil {
		return
	}

	term := wr.termWin.Active.GetTerm()
	if term == nil {
		return
	}

	menu := wr.NewContextMenu()
	menu.Append([]types.MenuItem{
		{
			Title: fmt.Sprintf("Paste from clipboard [%s+v]", types.KEY_STR_META),
			Fn:    wr.clipboardPaste,
			Icon:  0xf0ea,
		},
	}...)

	if wr.contextMenu != nil && len(wr.contextMenu.MenuItems()) > 0 {
		menu.Append(wr.contextMenu.MenuItems()...)
	}

	menu.Append([]types.MenuItem{
		{Title: types.MENU_SEPARATOR},
		{
			Title: fmt.Sprintf("Ask AI (%s)", agent.Get(wr.termWin.Active.Id()).ServiceName()),
			Fn:    wr.askAi,
			Icon:  0xe05d,
		},
		{Title: types.MENU_SEPARATOR},
		{
			Title: "Find text",
			Fn:    func() { term.Search(types.SEARCH_REGEX) },
			Icon:  0xf002,
		},
		{
			Title: "List command line history",
			Fn:    func() { term.Search(types.SEARCH_CMD_LINES) },
			Icon:  0xf0ae,
		},
		{
			Title: "Jump to AI prompts",
			Fn:    func() { term.Search(types.SEARCH_AI_PROMPTS) },
			Icon:  0xf0ca,
		},
		{
			Title: "Write all output to temp file",
			Fn:    wr.writeToTemp,
			Icon:  0xf0c7,
		},
		{Title: types.MENU_SEPARATOR},
	}...)

	if wr.tmux != nil {
		menu.Append(types.MenuItem{
			Title: "List tmux hotkeys...",
			Fn:    wr.tmux.ListKeyBindings,
			Icon:  0xf11c,
		})
	}

	menu.DisplayMenu("Select an action")
	//wr.contextMenu = wr.NewContextMenu()
}

func (wr *webkitRender) askAi() {
	if wr.termWin == nil || wr.termWin.Active == nil {
		return
	}

	agt := agent.Get(wr.termWin.Active.Id())
	agt.Meta = &agent.Meta{}

	wr.DisplayInputBox(
		fmt.Sprintf("What would you like to ask %s?", agt.ServiceName()),
		"",
		func(prompt string) {
			if prompt == "" {
				return
			}
			ai.AskAI(agt, prompt)
		},
		nil,
	)
}

func (wr *webkitRender) clipboardPaste() {
	term := wr.activeTerm()
	if term == nil {
		return
	}

	b := clipboard.Read(clipboard.FmtText)
	if len(b) != 0 {
		term.Reply(b)
		wr.TriggerRedraw()
		return
	}

	b = clipboard.Read(clipboard.FmtImage)
	if len(b) != 0 {
		f, err := os.CreateTemp("", "*.png")
		if err != nil {
			wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}

		if _, err = f.Write(b); err != nil {
			wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}

		if err = f.Close(); err != nil {
			wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}

		term.Reply([]byte(f.Name()))
		wr.TriggerRedraw()
		return
	}

	wr.DisplayNotification(types.NOTIFY_WARN, "Clipboard does not contain text to paste")
}

func (wr *webkitRender) writeToTemp() {
	if wr.termWin == nil || wr.termWin.Active == nil || wr.termWin.Active.GetTerm() == nil {
		return
	}

	file, err := os.CreateTemp("", "*.txt")
	if err != nil {
		wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	b := wr.termWin.Active.GetTerm().GetTermContents()
	if _, err = file.Write(b); err != nil {
		wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	if err = file.Close(); err != nil {
		wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	clipboard.Write(clipboard.FmtText, []byte(file.Name()))
	wr.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("Content written to disk & path copied to clipboard:\n%s", file.Name()))
}
