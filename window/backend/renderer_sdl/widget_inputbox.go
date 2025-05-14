package rendersdl

import (
	"fmt"
	"math"

	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/cursor"
	"github.com/mattn/go-runewidth"
	"github.com/veandco/go-sdl2/sdl"
)

type inputBoxCallbackT func(string)

type inputBoxWidgetT struct {
	title      string
	callback   inputBoxCallbackT
	readline   *widgetReadlineT
	_maxChars  int32
}

const _INPUT_MAX_CHAR_WIDTH = 80

func (sr *sdlRender) DisplayInputBox(title string, defaultValue string, callback func(string)) {
	maxChars := min(sr.winCellSize.X-15, _INPUT_MAX_CHAR_WIDTH)

	if callback == nil {
		callback = func(string) {}
	}

	sr.inputBox = &inputBoxWidgetT{
		title: title,
		readline: sr.NewReadline(maxChars, defaultValue,
			fmt.Sprintf(`[Return] Ok  |  [Ctrl+c] Cancel  |  [Esc] Vim Mode  |  [Up] Default: "%s"`, defaultValue),
		),
		callback:  callback,
		_maxChars: maxChars,
	}

	sr.termWin.Active.GetTerm().ShowCursor(false)
	cursor.Arrow()

	sr.inputBox.readline.Readline(sr, func(s string, e error) {
		sr.closeInputBox()
		if e == nil {
			callback(s)
		}
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

/*func (inputBox *inputBoxWidgetT) eventMouseWheel(sr *sdlRender, evt *sdl.MouseWheelEvent) {
	// do nothing
}*/

func (inputBox *inputBoxWidgetT) eventMouseMotion(sr *sdlRender, evt *sdl.MouseMotionEvent) {
	// do nothing
}

func (sr *sdlRender) renderInputBox(windowRect *sdl.Rect) {
	surface, err := sdl.CreateRGBSurfaceWithFormat(0, windowRect.W, windowRect.H, 32, uint32(sdl.PIXELFORMAT_RGBA32))
	if err != nil {
		panic(err) //TODO: don't panic!
	}
	defer surface.Free()

	value := sr.inputBox.readline.Value()
	nLines := max(1, int32(math.Ceil(float64(runewidth.StringWidth(value))/float64(sr.inputBox._maxChars))))

	/*
		FRAME
	*/

	textHeight := sr.glyphSize.Y
	height := textHeight + (_WIDGET_OUTER_MARGIN * 3) + (sr.glyphSize.Y * nLines)
	width := sr.glyphSize.X*sr.inputBox._maxChars + sr.notifyIconSize.X + (_WIDGET_OUTER_MARGIN * 3)
	offsetH := (surface.H / 2) - (height / 2)
	offsetY := (surface.W - width) / 2

	// draw border
	_ = sr.renderer.SetDrawColor(_INPUT_BOARDER_COLOUR.Red, _INPUT_BOARDER_COLOUR.Green, _INPUT_BOARDER_COLOUR.Blue, _INPUT_ALPHA)
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
	_ = sr.renderer.SetDrawColor(_INPUT_BACKGROUND.Red, _INPUT_BACKGROUND.Green, _INPUT_BACKGROUND.Blue, _INPUT_ALPHA_BG)
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

	height = (sr.glyphSize.Y * nLines) + _WIDGET_OUTER_MARGIN
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
	if len(value) > 0 {
		pos := &types.XY{
			X: offsetY + _WIDGET_OUTER_MARGIN + sr.notifyIconSize.X + _WIDGET_INNER_MARGIN,
			Y: _WIDGET_INNER_MARGIN + offsetH,
		}

		var (
			lineSlice []rune
			lineWidth int
		)
		for _, r := range value {
			w := runewidth.RuneWidth(r)
			if lineWidth+w > int(sr.inputBox._maxChars) {
				sr.printString(string(lineSlice), types.SGR_DEFAULT, pos)
				pos.Y += sr.glyphSize.Y
				lineSlice = []rune{r}
				lineWidth = w
			} else {
				lineSlice = append(lineSlice, r)
				lineWidth += runewidth.RuneWidth(r)
			}
		}
		sr.printString(string(lineSlice), types.SGR_DEFAULT, pos)
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

	if sr.GetBlinkState() {
		curPos := sr.inputBox.readline.CursorPosition()
		rect = sdl.Rect{
			X: offsetY + _WIDGET_OUTER_MARGIN + sr.notifyIconSize.X + _WIDGET_INNER_MARGIN + curPos.X,
			Y: _WIDGET_INNER_MARGIN + offsetH + curPos.Y,
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
