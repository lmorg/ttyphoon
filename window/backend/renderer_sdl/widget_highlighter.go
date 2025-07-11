package rendersdl

import (
	"fmt"

	"github.com/lmorg/mxtty/ai"
	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/cursor"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	highlightBorder = &types.Colour{0x31, 0x6d, 0xb0, 0xff}
	highlightFill   = &types.Colour{0x1c, 0x3e, 0x64, 0xff}
)

type _highlightMode uint8

const (
	_HIGHLIGHT_MODE_PNG _highlightMode = 0 + iota
	_HIGHLIGHT_MODE_SQUARE
	_HIGHLIGHT_MODE_FULL_LINES
	_HIGHLIGHT_MODE_LINE_RANGE
	_HIGHLIGHT_MODE_AI
)

type highlightWidgetT struct {
	button types.MouseButtonT
	rect   *sdl.Rect
	mode   _highlightMode
}

func (hl *highlightWidgetT) eventTextInput(sr *sdlRender, evt *sdl.TextInputEvent) {
	// do nothing
}

func (hl *highlightWidgetT) eventKeyPress(sr *sdlRender, evt *sdl.KeyboardEvent) {
	if evt.Keysym.Sym == sdl.K_ESCAPE {
		sr.highlighter = nil
		cursor.Arrow()
		return
	}

	hl.modifier(evt.Keysym.Mod)
}

func (hl *highlightWidgetT) modifier(mod sdl.Keymod) {
	switch {
	case mod&sdl.KMOD_CTRL != 0:
		fallthrough
	case mod&sdl.KMOD_LCTRL != 0:
		fallthrough
	case mod&sdl.KMOD_RCTRL != 0:
		hl.setMode(_HIGHLIGHT_MODE_SQUARE)

	case mod&sdl.KMOD_SHIFT != 0:
		fallthrough
	case mod&sdl.KMOD_LSHIFT != 0:
		fallthrough
	case mod&sdl.KMOD_RSHIFT != 0:
		hl.setMode(_HIGHLIGHT_MODE_LINE_RANGE)

	case mod&sdl.KMOD_ALT != 0:
		fallthrough
	case mod&sdl.KMOD_LALT != 0:
		fallthrough
	case mod&sdl.KMOD_RALT != 0:
		hl.setMode(_HIGHLIGHT_MODE_FULL_LINES)

	case mod&sdl.KMOD_GUI != 0:
		fallthrough
	case mod&sdl.KMOD_LGUI != 0:
		fallthrough
	case mod&sdl.KMOD_RGUI != 0:
		hl.setMode(_HIGHLIGHT_MODE_PNG)
	}
}

func (hl *highlightWidgetT) setMode(mode _highlightMode) {
	hl.mode = mode
	switch mode {
	case _HIGHLIGHT_MODE_LINE_RANGE:
		cursor.Ibeam()
	default:
		cursor.Arrow()
	}
}

func (hl *highlightWidgetT) eventMouseButton(sr *sdlRender, evt *sdl.MouseButtonEvent) {
	if evt.State == sdl.RELEASED {
		sr.StatusBarText("")
		sr.termWin.Active.GetTerm().MouseClick(nil, 0, 0, types.BUTTON_RELEASED, func() {})
	}

	hl.button = 0
	cursor.Arrow()

	switch hl.mode {
	case _HIGHLIGHT_MODE_PNG:
		normaliseRect(hl.rect)
		if hl.rect.W <= sr.glyphSize.X && hl.rect.H <= sr.glyphSize.Y {
			sr.clipboardPaste()
		}
		// clipboard copy will happen automatically on next redraw
		sr.TriggerRedraw()

	case _HIGHLIGHT_MODE_FULL_LINES:
		normaliseRect(hl.rect)
		rect := sr.rectPxToActiveTileCells(sr.termWin.Active, hl.rect)
		lines := sr.termWin.Active.GetTerm().CopyLines(rect.Y, rect.H)
		sr.highlighter = nil
		l := copyTextToClipboard(lines)
		if l > 0 {
			sr.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("%d lines have been copied to clipboard", l))
		}

	case _HIGHLIGHT_MODE_SQUARE:
		normaliseRect(hl.rect)
		rect := sr.rectPxToActiveTileCells(sr.termWin.Active, hl.rect)
		lines := sr.termWin.Active.GetTerm().CopySquare(&types.XY{X: rect.X, Y: rect.Y}, &types.XY{X: rect.W, Y: rect.H})
		sr.highlighter = nil
		l := copyTextToClipboard(lines)
		if l > 0 {
			sr.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("%dx%d grid has been copied to clipboard", rect.W-rect.X+1, l)) //rect.H-rect.Y+1))
		}

	case _HIGHLIGHT_MODE_LINE_RANGE, _HIGHLIGHT_MODE_AI:
		rect := sr.rectPxToActiveTileCells(sr.termWin.Active, hl.rect)
		pos := sr.convertPxToCellXYTile(sr.termWin.Active, evt.X, evt.Y)
		term := sr.termWin.Active.GetTerm()
		lines := term.CopyRange(&types.XY{X: rect.X, Y: rect.Y}, &types.XY{X: rect.W, Y: rect.H})
		sr.highlighter = nil
		//if rect.X-rect.W < 2 && rect.X-rect.W > -2 && rect.Y-rect.H == 0 {
		if rect.X-rect.W < 2 && rect.X-rect.W > -2 && rect.Y-rect.H < 2 && rect.Y-rect.H > -2 {
			term.MouseClick(pos, types.MouseButtonT(evt.Button), evt.Clicks, types.BUTTON_RELEASED, func() {})
			return
		}
		switch hl.mode {
		case _HIGHLIGHT_MODE_LINE_RANGE:
			l := copyTextToClipboard(lines)
			if l > 0 {
				sr.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("%d lines have been copied to clipboard", l))
			}
		case _HIGHLIGHT_MODE_AI:
			meta := agent.Get(sr.termWin.Active.Id())
			meta.Term = term
			meta.Renderer = sr
			meta.CmdLine = term.CmdLine(pos)
			meta.Pwd = term.Pwd(pos)
			meta.OutputBlock = string(lines)
			//meta.InsertRowPos = term.ConvertRelativeToAbsoluteY(pos)
			term.GetRowId(term.GetCursorPosition().Y)
			ai.Explain(meta, true)
		default:
			panic(fmt.Sprintf("TODO: unmet conditional '%d'", hl.mode))
		}

	default:
		panic(fmt.Sprintf("TODO: unmet conditional '%d'", hl.mode))
	}
}

func (hl *highlightWidgetT) eventMouseWheel(sr *sdlRender, evt *sdl.MouseWheelEvent) {
	// do nothing
}

func (hl *highlightWidgetT) eventMouseMotion(sr *sdlRender, evt *sdl.MouseMotionEvent) {
	hl.rect.W += evt.XRel
	hl.rect.H += evt.YRel
	sr.TriggerRedraw()
}

func (sr *sdlRender) selectionHighlighter() {
	if sr.highlighter == nil {
		return
	}

	var alphaBorder, alphaFill uint8
	var rect *sdl.Rect

	switch sr.highlighter.mode {
	case _HIGHLIGHT_MODE_PNG:
		alphaBorder, alphaFill = 190, 64
		rect = &sdl.Rect{X: sr.highlighter.rect.X, Y: sr.highlighter.rect.Y, W: sr.highlighter.rect.W, H: sr.highlighter.rect.H}

	case _HIGHLIGHT_MODE_SQUARE:
		alphaBorder, alphaFill = 64, 0
		rect = &sdl.Rect{X: sr.highlighter.rect.X, Y: sr.highlighter.rect.Y, W: sr.highlighter.rect.W, H: sr.highlighter.rect.H}

	case _HIGHLIGHT_MODE_LINE_RANGE, _HIGHLIGHT_MODE_FULL_LINES, _HIGHLIGHT_MODE_AI:
		return

	default:
		panic(fmt.Sprintf("TODO: unmet conditional '%d'", sr.highlighter.mode))
	}

	sr._drawHighlightRect(rect, highlightBorder, highlightFill, alphaBorder, alphaFill)
}

func isCellHighlighted(sr *sdlRender, rect *sdl.Rect) bool {
	if sr.highlighter == nil || sr.highlighter.button == 0 {
		return false
	}

	runeCell := sr.rectPxToCells(rect)

	if sr.termWin != nil {
		if runeCell.X < sr.termWin.Active.Left() || runeCell.X > sr.termWin.Active.Right() ||
			runeCell.Y < sr.termWin.Active.Top() || runeCell.Y > sr.termWin.Active.Bottom() {
			return false
		}
	}

	hlRect := *sr.highlighter.rect
	if sr.highlighter.mode != _HIGHLIGHT_MODE_LINE_RANGE {
		normaliseRect(&hlRect)
	}
	hlCell := sr.rectPxToCells(&hlRect)

	switch sr.highlighter.mode {
	case _HIGHLIGHT_MODE_FULL_LINES:
		return runeCell.Y >= hlCell.Y && runeCell.Y <= hlCell.H

	case _HIGHLIGHT_MODE_LINE_RANGE, _HIGHLIGHT_MODE_AI:
		switch {
		case hlCell.H < hlCell.Y: // select up
			// start multiline
			return ((runeCell.X <= hlCell.X && runeCell.Y == hlCell.Y) ||
				// middle multiline
				(runeCell.Y < hlCell.Y && runeCell.Y > hlCell.H) ||
				// end multiline
				(runeCell.X >= hlCell.W && runeCell.Y == hlCell.H))

		case hlCell.Y == hlCell.H:
			// midline
			if hlCell.W < hlCell.X { //backwards
				return runeCell.X <= hlCell.X && runeCell.X >= hlCell.W && runeCell.Y == hlCell.Y
			} else { // forwards
				return runeCell.X >= hlCell.X && runeCell.X <= hlCell.W && runeCell.Y == hlCell.Y
			}

		default: // select down
			// start multiline
			return ((runeCell.X >= hlCell.X && runeCell.Y == hlCell.Y) ||
				// middle multiline
				(runeCell.Y > hlCell.Y && runeCell.Y < hlCell.H) ||
				// end multiline
				(runeCell.X <= hlCell.W && runeCell.Y == hlCell.H))
		}

	case _HIGHLIGHT_MODE_SQUARE:
		return runeCell.X >= hlCell.X && runeCell.X <= hlCell.W &&
			runeCell.Y >= hlCell.Y && runeCell.Y <= hlCell.H

	case _HIGHLIGHT_MODE_PNG:
		return false

	default:
		panic(fmt.Sprintf("TODO: unmet conditional '%d'", sr.highlighter.mode))
	}
}
