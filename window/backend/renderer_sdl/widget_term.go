package rendersdl

import (
	"fmt"
	"time"

	"github.com/lmorg/mxtty/codes"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/integrations"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
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
					sr.termWin.Active.Term.Reply(b)

				case <-time.After(5 * time.Millisecond):
					sr.termWin.Active.Term.Reply(b)
				}

			}()
			return
		}
	}

	sr.termWin.Active.Term.Reply(b)
}

func (tw *termWidgetT) eventKeyPress(sr *sdlRender, evt *sdl.KeyboardEvent) {
	go func() {
		// basically this just tells tw.eventTextInput() to ignore input
		if (evt.Keysym.Sym >= sdl.K_KP_DIVIDE && evt.Keysym.Sym <= sdl.K_KP_PERIOD) ||
			(evt.Keysym.Sym == '\\' && (evt.Keysym.Mod == sdl.KMOD_LCTRL || evt.Keysym.Mod == sdl.KMOD_RCTRL)) {
			go func() {
				sr.keyIgnore <- true
			}()
		}

	}()
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

	switch {
	case evt.Keysym.Sym == sdl.K_F3 && mod == 0:
		sr.termWin.Active.Term.Search()
		return
	case evt.Keysym.Sym == 'v' && mod == codes.MOD_META:
		sr.clipboardPasteText()
		return
	}

	keyCode := sr.keyCodeLookup(evt.Keysym.Sym)
	b := codes.GetAnsiEscSeq(sr.keyboardMode.Get(), keyCode, mod)
	if len(b) > 0 {
		sr.termWin.Active.Term.Reply(b)
	}
}

// SDL doesn't name these, so lets name it ourselves for convenience
const (
	_MOUSE_BUTTON_LEFT = 1 + iota
	_MOUSE_BUTTON_MIDDLE
	_MOUSE_BUTTON_RIGHT
	_MOUSE_BUTTON_X1
	_MOUSE_BUTTON_X2
)

func (tw *termWidgetT) eventMouseButton(sr *sdlRender, evt *sdl.MouseButtonEvent) {
	//tile := sr.termWin.Active
	//if evt.State == sdl.PRESSED {
	tile := sr.getTileFromPxOrActive(evt.X, evt.Y)
	sr.termWin.Active.Term.HasFocus(false)
	tile.Term.HasFocus(true)
	sr.termWin.Active = tile
	//}

	posCell := sr.convertPxToCellXYNegXTile(tile, evt.X, evt.Y)

	if config.Config.Tmux.Enabled && sr.windowTabs != nil &&
		(evt.Y-_PANE_TOP_MARGIN)/sr.glyphSize.Y == sr.winCellSize.Y+sr.footer-1 {
		// window tab bar
		if evt.State == sdl.PRESSED {
			return
		}

		x := ((evt.X - _PANE_LEFT_MARGIN) / sr.glyphSize.X) - sr.windowTabs.offset.X
		for i := range sr.windowTabs.boundaries {
			if x < sr.windowTabs.boundaries[i] {
				switch evt.Clicks {
				case 1:
					if i == 0 {
						return
					}
					sr.selectWindow(i - 1)

				default: // 2 or more
					if i == 0 {
						return
					}
					sr.DisplayInputBox("Please enter a new name for this window", sr.windowTabs.windows[i-1].Name, func(name string) {
						err := sr.windowTabs.windows[i-1].Rename(name)
						if err != nil {
							sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
						}
					})
				}
				return
			}
		}
		if evt.Clicks == 2 {
			sr.tmux.NewWindow()
		}
		return
	}

	state := evt.State == sdl.PRESSED

	//if evt.X <= _PANE_LEFT_MARGIN {
	//	posCell.X = -1
	if posCell.X == -1 {
		sr.termWin.Active.Term.MouseClick(posCell, uint8(evt.Button), evt.Clicks, state, func() {})
		return
	}

	switch evt.Button {
	case _MOUSE_BUTTON_LEFT:
		sr.termWin.Active.Term.MouseClick(posCell, uint8(evt.Button), evt.Clicks, state, func() {
			if evt.State == sdl.PRESSED {
				highlighterStart(sr, uint8(evt.Button), evt.X, evt.Y)
				sr.highlighter.setMode(_HIGHLIGHT_MODE_LINE_RANGE)
			}
		})

	case _MOUSE_BUTTON_MIDDLE:
		if evt.State == sdl.PRESSED {
			sr.termWin.Active.Term.MouseClick(posCell, uint8(evt.Button), evt.Clicks, state, sr.clipboardPasteText)
		}

	case _MOUSE_BUTTON_RIGHT:
		sr.contextMenu = make(contextMenuT, 0) // empty the context menu
		sr.termWin.Active.Term.MouseClick(posCell, uint8(evt.Button), evt.Clicks, state, func() {
			if evt.State == sdl.RELEASED {
				tw._eventMouseButtonRightClick(sr, posCell)
			}
		})

	case _MOUSE_BUTTON_X1:
		sr.termWin.Active.Term.MouseClick(posCell, uint8(evt.Button), evt.Clicks, state, func() {})
	}
}

func (tw *termWidgetT) _eventMouseButtonRightClick(sr *sdlRender, posCell *types.XY) {
	menu := contextMenuT{
		{
			Title: fmt.Sprintf("Paste text from clipboard [%s+v]", types.KEY_STR_META),
			Fn:    sr.clipboardPasteText,
		},
		{
			Title: MENU_SEPARATOR,
		},
		{
			Title: "Fold on indentation",
			Fn: func() {
				err := sr.termWin.Active.Term.FoldAtIndent(posCell)
				if err != nil {
					sr.DisplayNotification(types.NOTIFY_WARN, err.Error())
				}
			},
		},
		//{
		//	Title: "Match bracket",
		//	Fn:    func() { sr.term.Match(posCell) },
		//},
		{
			Title: "Search text [F3]",
			Fn:    sr.termWin.Active.Term.Search,
		},
	}

	if sr.tmux != nil {
		menu = append(menu,
			types.MenuItem{
				Title: MENU_SEPARATOR,
			},
			types.MenuItem{
				Title: "List tmux hotkeys",
				Fn:    sr.tmux.ListKeyBindings,
			},
		)
	}

	if len(sr.contextMenu) > 0 {
		menu = append(menu, types.MenuItem{Title: MENU_SEPARATOR})
		menu = append(menu, sr.contextMenu...)
	}

	menu = append(menu,
		types.MenuItem{
			Title: MENU_SEPARATOR,
		},
		types.MenuItem{
			Title: "Bash integration (pasted into shell)",
			Fn:    func() { sr.termWin.Active.Term.Reply(integrations.Get("shell.bash")) },
		},
		types.MenuItem{
			Title: "Zsh integration (pasted into shell)",
			Fn:    func() { sr.termWin.Active.Term.Reply(integrations.Get("shell.zsh")) },
		},
	)

	sr.DisplayMenuUnderCursor("Select an action", menu.Options(), nil, menu.Callback, nil)
}

var _highlighterStartFooterText = fmt.Sprintf(
	"Copy to clipboard: [%s] Square region  |  [%s] Text region  |  [%s] Entire line(s)  |  [%s] PNG",
	types.KEY_STR_CTRL, types.KEY_STR_SHIFT, types.KEY_STR_ALT, types.KEY_STR_META,
)

func highlighterStart(sr *sdlRender, button uint8, x, y int32) {
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
		tile.Term.MouseWheel(pos, &types.XY{X: evt.X, Y: -evt.Y})
	} else {
		tile.Term.MouseWheel(pos, &types.XY{X: evt.X, Y: evt.Y})
	}
}

func (tw *termWidgetT) eventMouseMotion(sr *sdlRender, evt *sdl.MouseMotionEvent) {
	if config.Config.Tmux.Enabled && sr.windowTabs != nil {

		if (evt.Y-_PANE_TOP_MARGIN)/sr.glyphSize.Y == sr.winCellSize.Y+sr.footer-1 {
			x := ((evt.X - _PANE_LEFT_MARGIN) / sr.glyphSize.X) - sr.windowTabs.offset.X
			for i := range sr.windowTabs.boundaries {
				if x >= 0 && x < sr.windowTabs.boundaries[i] {
					sr.windowTabs.mouseOver = i - 1
					sr.footerText = fmt.Sprintf("[Click]  Switch to window '%s' (%s)", sr.windowTabs.windows[i-1].Name, sr.windowTabs.windows[i-1].Id)
					return
				}
			}
			sr.footerText = "[2x Click]  Start new window"
			sr.windowTabs.mouseOver = -1
			return
		}

		sr.windowTabs.mouseOver = -1
		sr.footerText = ""
	}

	/*pos, ok := sr.convertPxToCellXYNegX(evt.X, evt.Y)
	if !ok {
		return
	}*/

	tile := sr.getTileFromPxOrActive(evt.X, evt.Y)
	pos := sr.convertPxToCellXYNegXTile(tile, evt.X, evt.Y)

	var callback = sr._termMouseMotionCallback
	if evt.State != 0 {
		callback = func() {
			switch evt.State {
			case _MOUSE_BUTTON_LEFT:
				highlighterStart(sr, uint8(evt.State), pos.X-evt.XRel, pos.Y-evt.YRel)
				sr.highlighter.setMode(_HIGHLIGHT_MODE_LINE_RANGE)
			}
		}
	}

	//sr.termWin.Active.Term.MouseMotion(
	tile.Term.MouseMotion(
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
	if winIndex < 0 || winIndex >= len(sr.windowTabs.windows) {
		return
	}

	winId := sr.windowTabs.windows[winIndex].Id
	err := sr.tmux.SelectWindow(winId)
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
}

func (sr *sdlRender) RefreshWindowList() {
	if sr.tmux == nil {
		return
	}

	sr.limiter.Lock()

	sr.windowTabs = nil
	sr.termWin = sr.tmux.ActiveWindow()

	sr.limiter.Unlock()
}
