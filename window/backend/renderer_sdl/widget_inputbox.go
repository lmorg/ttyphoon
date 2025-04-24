package rendersdl

import (
	"fmt"

	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/cursor"
	"github.com/veandco/go-sdl2/sdl"
)

type inputBoxCallbackT func(string)

type inputBoxWidgetT struct {
	title    string
	callback inputBoxCallbackT
	readline *widgetReadlineT
}

func (sr *sdlRender) DisplayInputBox(title string, defaultValue string, callback func(string)) {
	sr.inputBox = &inputBoxWidgetT{
		title: title,
		readline: sr.NewReadline(
			defaultValue,
			fmt.Sprintf(`[Return] Ok  |  [Ctrl+c] Cancel  |  [Esc] Vim Mode  |  [Up] Default: "%s"`, defaultValue),
		),
		callback: callback,
	}

	sr.termWin.Active.GetTerm().ShowCursor(false)
	cursor.Arrow()

	sr.inputBox.readline.Readline(sr, func(s string, e error) {
		sr.closeInputBox()
		callback(s)
	})
}

func (sr *sdlRender) closeInputBox() {
	sr.footerText = ""
	sr.inputBox = nil
	sr.termWin.Active.GetTerm().ShowCursor(true)
}

func (inputBox *inputBoxWidgetT) eventTextInput(sr *sdlRender, evt *sdl.TextInputEvent) {
	inputBox.readline.eventTextInput(sr, evt)
}

func (inputBox *inputBoxWidgetT) eventKeyPress(sr *sdlRender, evt *sdl.KeyboardEvent) {
	inputBox.readline.eventKeyPress(sr, evt)
}

func (inputBox *inputBoxWidgetT) eventMouseButton(sr *sdlRender, evt *sdl.MouseButtonEvent) {
	if evt.State == sdl.PRESSED {
		sr.closeInputBox()
		sr.termWidget.eventMouseButton(sr, evt)
	}
}

func (inputBox *inputBoxWidgetT) eventMouseWheel(sr *sdlRender, evt *sdl.MouseWheelEvent) {
	// do nothing
}

func (inputBox *inputBoxWidgetT) eventMouseMotion(sr *sdlRender, evt *sdl.MouseMotionEvent) {
	// do nothing
}

const _INPUTBOX_MAX_CHARS = int32(75)

func (sr *sdlRender) renderInputBox(windowRect *sdl.Rect) {
	surface, err := sdl.CreateRGBSurfaceWithFormat(0, windowRect.W, windowRect.H, 32, uint32(sdl.PIXELFORMAT_RGBA32))
	if err != nil {
		panic(err) //TODO: don't panic!
	}
	defer surface.Free()

	/*
		FRAME
	*/

	textHeight := sr.glyphSize.Y
	height := textHeight + (_WIDGET_OUTER_MARGIN * 3) + sr.glyphSize.Y
	maxLen := int32(len(sr.inputBox.title))
	if maxLen < _INPUTBOX_MAX_CHARS {
		maxLen = _INPUTBOX_MAX_CHARS
	}
	width := sr.glyphSize.X*maxLen + sr.notifyIconSize.X + _WIDGET_OUTER_MARGIN
	offsetH := (surface.H / 2) - (height / 2)
	offsetY := (surface.W - width) / 2

	// draw border
	_ = sr.renderer.SetDrawColor(types.SGR_COLOR_BLACK.Red, types.SGR_COLOR_BLACK.Green, types.SGR_COLOR_BLACK.Blue, _INPUT_ALPHA)
	rect := sdl.Rect{
		X: offsetY - 1,
		Y: offsetH - 1,
		W: width + 2,
		H: height + 2,
	}
	sr.renderer.DrawRect(&rect)
	rect = sdl.Rect{
		X: offsetY,
		Y: offsetH,
		W: width,
		H: height,
	}
	sr.renderer.DrawRect(&rect)

	// fill background
	_ = sr.renderer.SetDrawColor(types.COLOR_WIDGET_INPUT.Red, types.COLOR_WIDGET_INPUT.Green, types.COLOR_WIDGET_INPUT.Blue, _INPUT_ALPHA)
	rect = sdl.Rect{
		X: offsetY + 1,
		Y: 1 + offsetH,
		W: width - 2,
		H: height - 2,
	}
	sr.renderer.FillRect(&rect)

	/*
		TEXT FIELD
	*/

	sr.printString(sr.inputBox.title, types.SGR_HEADING, &types.XY{
		X: offsetY + _WIDGET_OUTER_MARGIN + sr.notifyIconSize.X,
		Y: _WIDGET_INNER_MARGIN + offsetH,
	})

	height = sr.glyphSize.Y + _WIDGET_OUTER_MARGIN
	offsetH += textHeight + _WIDGET_OUTER_MARGIN

	// draw border
	_ = sr.renderer.SetDrawColor(types.SGR_COLOR_FOREGROUND.Red, types.SGR_COLOR_FOREGROUND.Green, types.SGR_COLOR_FOREGROUND.Blue, _INPUT_ALPHA)
	rect = sdl.Rect{
		X: offsetY + sr.notifyIconSize.X + _WIDGET_OUTER_MARGIN - 1,
		Y: offsetH - 1,
		W: width - sr.notifyIconSize.X - _WIDGET_OUTER_MARGIN - _WIDGET_OUTER_MARGIN + 2,
		H: height + 2,
	}
	sr.renderer.DrawRect(&rect)
	rect = sdl.Rect{
		X: offsetY + sr.notifyIconSize.X + _WIDGET_OUTER_MARGIN,
		Y: offsetH,
		W: width - sr.notifyIconSize.X - _WIDGET_OUTER_MARGIN - _WIDGET_OUTER_MARGIN,
		H: height,
	}
	sr.renderer.DrawRect(&rect)

	// fill background
	sr.renderer.SetDrawColor(types.SGR_COLOR_BACKGROUND.Red, types.SGR_COLOR_BACKGROUND.Green, types.SGR_COLOR_BACKGROUND.Blue, _INPUT_ALPHA_BG)
	rect = sdl.Rect{
		X: offsetY + sr.notifyIconSize.X + _WIDGET_OUTER_MARGIN + 1,
		Y: 1 + offsetH,
		W: width - sr.notifyIconSize.X - _WIDGET_OUTER_MARGIN - _WIDGET_OUTER_MARGIN - 2,
		H: height - 2,
	}
	sr.renderer.FillRect(&rect)

	// value
	value := sr.inputBox.readline.Value()
	if len(value) > 0 {
		pos := &types.XY{
			X: offsetY + _WIDGET_OUTER_MARGIN + sr.notifyIconSize.X + _WIDGET_INNER_MARGIN,
			Y: _WIDGET_INNER_MARGIN + offsetH,
		}

		sr.printString(value, types.SGR_DEFAULT, pos)
	}

	if surface, ok := sr.notifyIcon[types.NOTIFY_QUESTION].Asset().(*sdl.Surface); ok {
		srcRect := &sdl.Rect{
			X: 0,
			Y: 0,
			W: width,
			H: surface.H,
		}

		dstRect := &sdl.Rect{
			X: offsetY + (_WIDGET_OUTER_MARGIN / 2),
			Y: offsetH + textHeight - sr.notifyIconSize.Y,
			W: sr.notifyIconSize.X,
			H: sr.notifyIconSize.X,
		}

		texture, err := sr.renderer.CreateTextureFromSurface(surface)
		if err != nil {
			panic(err) // TODO: don't panic!
		}
		defer texture.Destroy()

		err = sr.renderer.Copy(texture, srcRect, dstRect)
		if err != nil {
			panic(err) // TODO: don't panic!
		}
	}

	//textWidth := int32(runewidth.StringWidth(sr.inputBox.value)) * (sr.glyphSize.X + dropShadowOffset)
	if sr.GetBlinkState() {
		rect = sdl.Rect{
			X: offsetY + _WIDGET_OUTER_MARGIN + sr.notifyIconSize.X + _WIDGET_INNER_MARGIN + sr.inputBox.readline.CursorPosition(),
			Y: _WIDGET_INNER_MARGIN + offsetH,
			W: sr.glyphSize.X,
			H: sr.glyphSize.Y,
		}
		sr._drawHighlightRect(&rect, highlightBorder, highlightFill, 255, 200)
	}

	texture, err := sr.renderer.CreateTextureFromSurface(surface)
	if err != nil {
		panic(err) // TODO: better error handling please!
	}
	defer texture.Destroy()

	err = sr.renderer.Copy(texture, windowRect, windowRect)
	if err != nil {
		panic(err) // TODO: better error handling please!
	}
}
