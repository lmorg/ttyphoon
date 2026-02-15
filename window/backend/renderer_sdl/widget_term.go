package rendersdl

import (
	"fmt"
	"os"
	"time"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/hotkeys"
	"github.com/lmorg/ttyphoon/types"
	"github.com/veandco/go-sdl2/sdl"
	"golang.design/x/clipboard"
)

type termWidgetT struct{}

func (tw *termWidgetT) eventTextInput(sr *sdlRender, evt *sdl.TextInputEvent) {
	sr.footerText = ""
	b := []byte(evt.GetText())

	if len(b) == 1 {
		if (b[0] >= '0' && b[0] <= '9') || // keypad numeric
			b[0] == '+' || b[0] == '-' || b[0] == '*' || b[0] == '/' || b[0] == '.' || // keypad syms
			b[0] == '`' { // ctrl+\ in tmux

			go func() {
				select {
				case ignore := <-sr.keyIgnore:
					if ignore {
						return
					}

					if !sr.hotkey(codes.KeyCode(b[0]), 0) {
						sr.termWin.Active.GetTerm().Reply(b)
					}

				case <-time.After(5 * time.Millisecond):
					if !sr.hotkey(codes.KeyCode(b[0]), 0) {
						sr.termWin.Active.GetTerm().Reply(b)
					}
				}

			}()
			return
		}
		if sr.hotkey(codes.KeyCode(b[0]), 0) {
			return
		}
	}

	sr.termWin.Active.GetTerm().Reply(b)
}

func (tw *termWidgetT) eventKeyPress(sr *sdlRender, evt *sdl.KeyboardEvent) {
	sr.highlightAi = false

	go func() {
		// basically this just tells tw.eventTextInput() to ignore input
		if (evt.Keysym.Sym >= sdl.K_KP_DIVIDE && evt.Keysym.Sym <= sdl.K_KP_PERIOD) ||
			(evt.Keysym.Sym == '\\' && (evt.Keysym.Mod == sdl.KMOD_LCTRL || evt.Keysym.Mod == sdl.KMOD_RCTRL)) {
			go func() {
				sr.keyIgnore <- true
			}()
		}

	}()

	sr.SetBlinkState(true)
	tw._eventKeyPress(sr, evt)
}

func (tw *termWidgetT) _eventKeyPress(sr *sdlRender, evt *sdl.KeyboardEvent) {
	sr.footerText = ""
	sr.keyModifier = evt.Keysym.Mod

	switch evt.Keysym.Sym {
	case sdl.K_LSHIFT, sdl.K_RSHIFT, sdl.K_LALT, sdl.K_RALT,
		sdl.K_LCTRL, sdl.K_RCTRL, sdl.K_LGUI, sdl.K_RGUI,
		sdl.K_CAPSLOCK, sdl.K_NUMLOCKCLEAR, sdl.K_SCROLLLOCK, sdl.K_SPACE:
		// modifier keys pressed on their own shouldn't trigger anything
		return
	}

	mod := keyEventModToCodesModifier(evt.Keysym.Mod)

	if (evt.Keysym.Sym > ' ' && evt.Keysym.Sym < 127) &&
		(mod == 0 || mod == codes.MOD_SHIFT) {
		// lets let eventTextInput() handle this so we don't need to think about
		// keyboard layouts and shift chars like whether shift+'2' == '@' or '"'
		return
	}

	if evt.Keysym.Sym == sdl.K_APPLICATION {
		tw._eventMouseButtonRightClick(sr, &types.XY{Y: sr.termWin.Active.GetTerm().GetSize().Y - 1}, false)
		return
	}

	keyCode := sr.keyCodeLookup(evt.Keysym.Sym)

	if sr.hotkey(keyCode, mod) {
		return
	}

	b := codes.GetAnsiEscSeq(sr.keyboardMode.Get(), keyCode, mod)
	if len(b) > 0 {
		sr.termWin.Active.GetTerm().Reply(b)
	}
}

func (sr *sdlRender) visualEditor() {
	sr.DisplayInputBox("Visual editor", "", func(s string) {
		if s != "" {
			sr.termWin.Active.GetTerm().Reply([]byte(s))
		}
	}, nil)
}

func (sr *sdlRender) hotkey(keyCode codes.KeyCode, mod codes.Modifier) bool {
	fn := hotkeys.KeyPress(keyCode, mod)
	if fn == nil {
		return false
	}

	fn()
	return true
}

func (tw *termWidgetT) eventMouseButton(sr *sdlRender, evt *sdl.MouseButtonEvent) {
	if evt.Button != sdl.Button(types.MOUSE_BUTTON_LEFT) {
		sr.highlightAi = false
	}

	if config.Config.Tmux.Enabled && sr.windowTabs != nil &&
		(evt.Y-_PANE_TOP_MARGIN)/sr.glyphSize.Y == sr.winCellSize.Y+sr.footer-1 {
		tw._eventMouseButtonFooter(sr, evt)
		return
	}

	tile := sr.getTileFromPxOrActive(evt.X, evt.Y)
	sr.termWin.Active.GetTerm().HasFocus(false)
	tile.GetTerm().HasFocus(true)
	sr.termWin.Active = tile
	sr.cacheBgTexture.Destroy(sr)
	if sr.tmux != nil {
		go func() {
			err := sr.tmux.SelectPane(tile.Id())
			if err != nil {
				sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			}
		}()
	}

	posCell := sr.convertPxToCellXYNegXTile(tile, evt.X, evt.Y)

	button := types.MouseButtonT(evt.Button)
	state := types.ButtonStateT(evt.State)

	if posCell.X == -1 {
		sr.termWin.Active.GetTerm().MouseClick(posCell, button, evt.Clicks, state, func() {})
		return
	}

	switch types.MouseButtonT(evt.Button) {
	case types.MOUSE_BUTTON_LEFT:
		sr.termWin.Active.GetTerm().MouseClick(posCell, button, evt.Clicks, state, func() {
			if evt.State == sdl.PRESSED {
				highlighterStart(sr, button, evt.X, evt.Y)
				if sr.highlightAi {
					sr.highlighter.setMode(_HIGHLIGHT_MODE_AI)
				} else {
					sr.highlighter.setMode(_HIGHLIGHT_MODE_LINE_RANGE)
				}
			}
		})

	case types.MOUSE_BUTTON_MIDDLE:
		//if evt.State == sdl.PRESSED {
		sr.termWin.Active.GetTerm().MouseClick(posCell, button, evt.Clicks, state, sr.clipboardPaste)
		//}

	case types.MOUSE_BUTTON_RIGHT:
		sr.contextMenu = sr.NewContextMenu() // empty the context menu
		sr.termWin.Active.GetTerm().MouseClick(posCell, button, evt.Clicks, state, func() {
			if evt.State == sdl.RELEASED {
				tw._eventMouseButtonRightClick(sr, sr.convertPxToCellXYTile(sr.termWin.Active, evt.X, evt.Y), true)
			}
		})

	case types.MOUSE_BUTTON_X1:
		sr.termWin.Active.GetTerm().MouseClick(posCell, button, evt.Clicks, state, func() {})
	}
}

func (tw *termWidgetT) _eventMouseButtonRightClick(sr *sdlRender, pos *types.XY, underCursor bool) {
	term := sr.termWin.Active.GetTerm()

	menu := sr.NewContextMenu()
	menu.Append(types.MenuItem{
		Title: fmt.Sprintf("Paste from clipboard [%s+v]", types.KEY_STR_META),
		Fn:    sr.clipboardPaste,
		Icon:  0xf0ea,
	})

	if sr.contextMenu != nil && len(sr.contextMenu.MenuItems()) > 0 {
		//menu = append(menu, types.MenuItem{Title: MENU_SEPARATOR})
		menu.Append(sr.contextMenu.MenuItems()...)
	}

	menu.Append([]types.MenuItem{
		{
			Title: types.MENU_SEPARATOR,
		},
		/*{
			Title: "Fold on indentation",
			Fn: func() {
				err := sr.termWin.Active.Term.FoldAtIndent(posCell)
				if err != nil {
					sr.DisplayNotification(types.NOTIFY_WARN, err.Error())
				}
			},
		},*/
		{
			Title: fmt.Sprintf("Highlight and explain (%s)", agent.Get(sr.termWin.Active.Id()).ServiceName()),
			Fn:    func() { sr.highlightAi = true },
			Icon:  0xf72b,
		},
		{
			Title: fmt.Sprintf("Ask AI (%s)", agent.Get(sr.termWin.Active.Id()).ServiceName()),
			Fn:    func() { askAi(sr, pos) },
			Icon:  0xe05d,
		},
		{
			Title: types.MENU_SEPARATOR,
		},
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
			Fn:    sr.writeToTemp,
			Icon:  0xf0c7,
		},
		{
			Title: types.MENU_SEPARATOR,
		},
	}...)

	if sr.tmux != nil {
		menu.Append(types.MenuItem{
			Title: "List tmux hotkeys...",
			Fn:    sr.tmux.ListKeyBindings,
			Icon:  0xf11c,
		})
	}

	menu.Append(types.MenuItem{
		Title: "Settings...",
		Fn:    sr.UpdateConfig,
		Icon:  0xf013,
	})

	menuFn := sr.DisplayMenuUnderCursor
	if !underCursor {
		menuFn = sr.displayMenuWithIcons
	}

	menuFn("Select an action", menu.Options(), menu.Icons(), menu.Highlight, menu.Callback, menu.Cancel)
}

var _highlighterStartFooterText = fmt.Sprintf(
	"Copy to clipboard: [%s] Square region  |  [%s] Text region  |  [%s] Entire line(s)  |  [%s] PNG",
	types.KEY_STR_CTRL, types.KEY_STR_SHIFT, types.KEY_STR_ALT, types.KEY_STR_META,
)

func highlighterStart(sr *sdlRender, button types.MouseButtonT, x, y int32) {
	sr.footerText = _highlighterStartFooterText

	sr.highlighter = &highlightWidgetT{
		button: button,
		rect:   &sdl.Rect{X: x, Y: y},
	}
	if sr.keyModifier != 0 {
		sr.highlighter.modifier(sr.keyModifier)
	}
	sr.keyModifier = 0
}

func (tw *termWidgetT) eventMouseWheel(sr *sdlRender, evt *sdl.MouseWheelEvent) {
	mouseX, mouseY, _ := sdl.GetMouseState()
	tile := sr.getTileFromPxOrActive(mouseX, mouseY)
	pos := sr.convertPxToCellXYTile(tile, mouseX, mouseY)

	if evt.Direction == sdl.MOUSEWHEEL_FLIPPED {
		tile.GetTerm().MouseWheel(pos, &types.XY{X: evt.X, Y: -evt.Y})
	} else {
		tile.GetTerm().MouseWheel(pos, &types.XY{X: evt.X, Y: evt.Y})
	}
}

func (tw *termWidgetT) eventMouseMotion(sr *sdlRender, evt *sdl.MouseMotionEvent) {
	sr.TriggerLazyRedraw()

	if config.Config.Tmux.Enabled && sr.windowTabs != nil {

		if (evt.Y-_PANE_TOP_MARGIN)/sr.glyphSize.Y == sr.winCellSize.Y+sr.footer-1 {
			tw._eventMouseMotionFooter(sr, evt)
			return
		}

		sr.windowTabs.mouseOver = -1
		sr.footerText = ""
	}

	tile := sr.getTileFromPxOrActive(evt.X, evt.Y)
	pos := sr.convertPxToCellXYNegXTile(tile, evt.X, evt.Y)

	var callback = sr._termMouseMotionCallback
	if evt.State != 0 {
		callback = func() {
			switch evt.State {
			case sdl.ButtonLMask:
				highlighterStart(sr, types.MOUSE_BUTTON_LEFT, pos.X-evt.XRel, pos.Y-evt.YRel)
				sr.highlighter.setMode(_HIGHLIGHT_MODE_LINE_RANGE)
			}
		}
	}

	tile.GetTerm().MouseMotion(
		pos,
		&types.XY{
			X: evt.XRel / sr.glyphSize.X,
			Y: evt.YRel / sr.glyphSize.Y,
		},
		callback,
	)
}

func (sr *sdlRender) _termMouseMotionCallback() {
	sr.footerText = "[Left Click] Copy  |  [Right Click] Menu  |  [Wheel] Scrollback buffer"
}

func (sr *sdlRender) selectWindow(winIndex int) {
	if winIndex < 0 || winIndex >= len(*sr.windowTabs.tabs) {
		return
	}

	winId := (*sr.windowTabs.tabs)[winIndex].Id()
	err := sr.tmux.SelectAndResizeWindow(winId, sr.winCellSize)
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
	go sr.RefreshWindowList()
	sr.TriggerRedraw()
}

func (sr *sdlRender) RefreshWindowList() {
	if sr.tmux == nil {
		return
	}

	sr.limiter.Lock()

	sr.windowTabs = nil
	sr.termWin = sr.tmux.GetTermTiles()
	sr.cacheBgTexture.Destroy(sr)

	sr.limiter.Unlock()
}

func (sr *sdlRender) writeToTemp() {
	file, err := os.CreateTemp("", "*.txt")
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	b := sr.termWin.Active.GetTerm().GetTermContents()
	_, err = file.Write(b)
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	err = file.Close()
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	clipboard.Write(clipboard.FmtText, []byte(file.Name()))

	msg := fmt.Sprintf("Content written to disk & path copied to clipboard:\n%s", file.Name())
	sr.DisplayNotification(types.NOTIFY_INFO, msg)
}
