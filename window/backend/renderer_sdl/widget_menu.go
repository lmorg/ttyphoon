package rendersdl

import (
	"strings"

	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/utils/runewidth"
	"github.com/lmorg/mxtty/window/backend/cursor"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/layer"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	_INPUT_ALPHA    = 255
	_INPUT_ALPHA_BG = 255
)

type menuWidgetT struct {
	title             string
	options           []string
	incIcons          bool
	icons             []rune
	highlightIndex    int
	highlightCallback types.MenuCallbackT
	selectCallback    types.MenuCallbackT
	cancelCallback    types.MenuCallbackT
	mouseRect         sdl.Rect
	pos               *types.XY
	maxLen            int32
	readline          *widgetReadlineT
	hidden            []bool
	_hoverFn          func()
}

const (
	_MENU_HIGHLIGHT_HIDDEN = -2
	_MENU_HIGHLIGHT_INIT   = -1
)

type contextMenuT struct {
	items    []types.MenuItem
	renderer *sdlRender
}

func (renderer *sdlRender) NewContextMenu() types.ContextMenu {
	return &contextMenuT{renderer: renderer}
}

func (cm *contextMenuT) Options() []string {
	slice := make([]string, len(cm.items))
	for i := range cm.items {
		slice[i] = cm.items[i].Title
	}
	return slice
}

func (cm *contextMenuT) Icons() []rune {
	slice := make([]rune, len(cm.items))
	for i := range cm.items {
		slice[i] = cm.items[i].Icon
	}
	return slice
}

func (cm *contextMenuT) Highlight(i int) {
	if i < 0 || i > len(cm.items) {
		cm.renderer.menu._hoverFn = nil
		return
	}

	if cm.items[i].Highlight == nil {
		cm.renderer.menu._hoverFn = nil
		return
	}

	cm.renderer.menu._hoverFn = cm.items[i].Highlight()
}

func (cm *contextMenuT) Callback(i int) {
	cm._clearHoverFn()

	if i < 0 || i > len(cm.items) {
		return
	}

	cm.items[i].Fn()
}

func (cm *contextMenuT) Cancel(i int) {
	cm._clearHoverFn()
}

func (cm *contextMenuT) _clearHoverFn() {
	if cm.renderer.menu != nil {
		cm.renderer.menu._hoverFn = nil
	}
}

func (cm *contextMenuT) Append(menuItems ...types.MenuItem) {
	cm.items = append(cm.items, menuItems...)
}

func (cm *contextMenuT) DisplayMenu(title string) {
	cm.renderer.DisplayMenuUnderCursor(title, cm.Options(), cm.Icons(), cm.Highlight, cm.Callback, cm.Cancel)
}

func (cm *contextMenuT) MenuItems() []types.MenuItem { return cm.items }

func (sr *sdlRender) AddToContextMenu(menuItems ...types.MenuItem) {
	sr.contextMenu.Append(menuItems...)
}

func (sr *sdlRender) DisplayMenuUnderCursor(title string, options []string, icons []rune, highlightCallback, selectCallback, cancelCallback types.MenuCallbackT) {
	if len(options) == 0 {
		sr.DisplayNotification(types.NOTIFY_WARN, "Nothing to show in menu")
		return
	}

	sr.displayMenuWithIcons(title, options, icons, highlightCallback, selectCallback, cancelCallback)

	x, y, _ := sdl.GetMouseState()
	sr.menu.pos = &types.XY{X: x, Y: y}
}

func (sr *sdlRender) displayMenuWithIcons(title string, options []string, icons []rune, highlightCallback, selectCallback, cancelCallback types.MenuCallbackT) {
	sr.displayMenu(title, options, icons, highlightCallback, selectCallback, cancelCallback)
	sr.menu.icons = icons
}

func (sr *sdlRender) DisplayMenu(title string, options []string, highlightCallback, selectCallback, cancelCallback types.MenuCallbackT) {
	sr.displayMenu(title, options, nil, highlightCallback, selectCallback, cancelCallback)
}

func (sr *sdlRender) displayMenu(title string, options []string, icons []rune, highlightCallback, selectCallback, cancelCallback types.MenuCallbackT) {
	if len(options) == 0 {
		sr.DisplayNotification(types.NOTIFY_WARN, "Nothing to show in menu")
		return
	}

	if highlightCallback == nil {
		highlightCallback = func(_ int) {}
	}
	if selectCallback == nil {
		selectCallback = func(_ int) {}
	}
	if cancelCallback == nil {
		cancelCallback = func(_ int) {}
	}

	opts := make([]string, len(options))
	copy(opts, options)

	//sr.footerText = "[Up/Down] Highlight  |  [Return] Choose  |  [Esc] Cancel"
	sr.menu = &menuWidgetT{
		title:             title,
		options:           opts,
		incIcons:          len(icons) != 0,
		icons:             icons,
		hidden:            make([]bool, len(options)),
		highlightCallback: highlightCallback,
		selectCallback:    selectCallback,
		cancelCallback:    cancelCallback,
		highlightIndex:    _MENU_HIGHLIGHT_INIT,
	}

	var crop = int(sr.winCellSize.X - 10)
	if sr.menu.incIcons {
		crop -= 3
	}

	for i := range sr.menu.options {
		sr.menu.options[i] = runewidth.Truncate(sr.menu.options[i], int(crop), "…")
		if len(sr.menu.options[i]) > int(sr.menu.maxLen) {
			sr.menu.maxLen = int32(runewidth.StringWidth(sr.menu.options[i]))
		}
	}

	sr.menu.readline = sr.NewReadline(sr.menu.maxLen, "", "[Up/Down] Highlight  |  [Return] Choose  |  [Ctrl+c] Cancel  |  [Esc] Vim Mode)")

	sr.menu.readline.Hook = func() {
		sr.menu.updateHidden()
		sr.menu.updateHighlight(0)
	}

	sr.menu.readline.Readline(sr, func(s string, e error) {
		i := sr.menu.highlightIndex
		sr.closeMenu()
		if e != nil {
			cancelCallback(i)
		} else {
			selectCallback(i)
		}
	})

	sr.termWin.Active.GetTerm().ShowCursor(false)
	cursor.Arrow()
}

func (sr *sdlRender) closeMenu() {
	sr.footerText = ""
	sr.termWin.Active.GetTerm().ShowCursor(true)
	cursor.Arrow()
	sr.menu = nil
}

func (menu *menuWidgetT) updateHidden() {
	filter := menu.readline.Value()
	if filter == "" {
		for i := range menu.hidden {
			menu.hidden[i] = false
		}
		return
	}

	filter = strings.ToLower(filter)
	for i := range menu.options {
		menu.hidden[i] = !strings.Contains(strings.ToLower(menu.options[i]), filter)
	}
}

func (menu *menuWidgetT) updateHighlight(adjust int) {
	if adjust == 0 {
		//hl := menu.highlightIndex
		if menu.highlightIndex < 0 {
			menu.highlightIndex = 0
		}
		if menu.hidden[menu.highlightIndex] {
			adjust = 1
		}
	}

	var attempts int
	for {
		attempts++
		menu.highlightIndex += adjust

		if menu.highlightIndex >= len(menu.options) {
			menu.highlightIndex = 0
		} else if menu.highlightIndex < 0 {
			menu.highlightIndex = len(menu.options) - 1
		}

		if attempts > len(menu.options) {
			menu.highlightIndex = _MENU_HIGHLIGHT_HIDDEN
			return
		}

		if menu.hidden[menu.highlightIndex] {
			continue
		}

		if menu.options[menu.highlightIndex] != types.MENU_SEPARATOR {
			break
		}
	}

	menu.highlightCallback(menu.highlightIndex)
}

func (menu *menuWidgetT) eventTextInput(sr *sdlRender, evt *sdl.TextInputEvent) {
	//menu.filter += evt.GetText()
	menu.readline.eventTextInput(sr, evt)
}

func (menu *menuWidgetT) eventKeyPress(sr *sdlRender, evt *sdl.KeyboardEvent) {
	//mod := keyEventModToCodesModifier(evt.Keysym.Mod)

	switch evt.Keysym.Sym {
	case sdl.K_RETURN, sdl.K_RETURN2, sdl.K_KP_ENTER:
		if menu.highlightIndex < 0 {
			return
		}
		sr.closeMenu()
		menu.selectCallback(menu.highlightIndex)
		return

	case sdl.K_UP:
		menu.updateHighlight(-1)
		return

	case sdl.K_DOWN:
		menu.updateHighlight(1)
		return
	}

	menu.readline.eventKeyPress(sr, evt)
}

func (menu *menuWidgetT) eventMouseButton(sr *sdlRender, evt *sdl.MouseButtonEvent) {
	if evt.State != sdl.RELEASED {
		return
	}
	if evt.Button != 1 {
		sr.closeMenu()
		menu.cancelCallback(menu.highlightIndex)
		return
	}

	i := menu._mouseHover(evt.X, evt.Y, sr.glyphSize)
	menu._mouseMotion(evt.X, evt.Y, sr)
	if i == -1 {
		sr.closeMenu()
		menu.cancelCallback(menu.highlightIndex)
		return
	}

	sr.closeMenu()
	menu.selectCallback(menu.highlightIndex)
}

func (menu *menuWidgetT) eventMouseWheel(sr *sdlRender, evt *sdl.MouseWheelEvent) {
	// do nothing
}

func (menu *menuWidgetT) eventMouseMotion(sr *sdlRender, evt *sdl.MouseMotionEvent) {
	sr.TriggerRedraw()
	menu._mouseMotion(evt.X, evt.Y, sr)
}

func (menu *menuWidgetT) _mouseMotion(x, y int32, sr *sdlRender) {
	sr.TriggerRedraw()
	i := menu._mouseHover(x, y, sr.glyphSize)
	if i == -1 {
		cursor.Arrow()
		return
	}

	if menu.hidden[i] {
		cursor.Arrow()
		return
	}

	cursor.Hand()
	menu.highlightIndex = i
	menu.highlightCallback(menu.highlightIndex)
}

func (menu *menuWidgetT) _mouseHover(x, y int32, glyphSize *types.XY) int {
	if x < menu.mouseRect.X || x > menu.mouseRect.X+menu.mouseRect.W {
		return -1
	}
	if y < menu.mouseRect.Y || y > menu.mouseRect.Y+menu.mouseRect.H {
		return -1
	}

	rel := y - menu.mouseRect.Y
	i := int(rel / glyphSize.Y)

	if i >= len(menu.options) {
		return -1
	}

	if menu.options[i] == types.MENU_SEPARATOR {
		menu.highlightIndex = _MENU_HIGHLIGHT_HIDDEN
		return -1
	}

	return i
}

func (sr *sdlRender) renderMenu(windowRect *sdl.Rect) {
	if sr.menu.highlightIndex < 0 {
		sr.menu._hoverFn = nil
	}

	if sr.menu._hoverFn != nil {
		sr.menu._hoverFn()
	}

	if sr.menu.highlightIndex == _MENU_HIGHLIGHT_INIT {
		sr.menu.highlightIndex = 0
		sr.menu.highlightCallback(0)
	}

	texture := sr.createRendererTexture()
	if texture == nil {
		return
	}

	var optionOffset int32
	if sr.menu.incIcons {
		optionOffset = 3 * sr.glyphSize.X
	}

	/*
		FRAME
	*/

	iconByGlyphs := (sr.notifyIconSize.X / sr.glyphSize.X) + 1
	maxLen := sr.menu.maxLen
	if int32(len(sr.menu.title))+iconByGlyphs > maxLen {
		maxLen = (int32(len(sr.menu.title)) + iconByGlyphs)
	}
	height := (sr.glyphSize.Y * int32(len(sr.menu.options))) + (_WIDGET_OUTER_MARGIN * 2) + sr.notifyIconSize.Y
	width := (sr.glyphSize.X * maxLen) + (_WIDGET_OUTER_MARGIN * 3) + optionOffset

	var x, y int32
	if sr.menu.pos != nil {
		x = sr.menu.pos.X
		y = sr.menu.pos.Y

		winX, winY, err := sr.renderer.GetOutputSize()
		if err == nil {
			if sr.menu.pos.X+width > winX {
				x = winX - width
			}

			fullHeight := height + (_WIDGET_INNER_MARGIN * 4) + sr.glyphSize.Y
			if sr.menu.pos.Y+fullHeight > winY {
				y = winY - fullHeight
			}
		}
	} else {
		x = (windowRect.W - width) / 2
		y = (windowRect.H - height) / 2
	}

	menuRect := sdl.Rect{
		X: x,
		Y: y,
		W: width,
		H: height,
	}

	// draw border
	_ = sr.renderer.SetDrawColor(notifyBorderColour[types.NOTIFY_QUESTION].Red, notifyBorderColour[types.NOTIFY_QUESTION].Green, notifyBorderColour[types.NOTIFY_QUESTION].Blue, notifyBorderColour[types.NOTIFY_QUESTION].Alpha)
	rect := sdl.Rect{
		X: menuRect.X - 1,
		Y: menuRect.Y - 1,
		W: menuRect.W + 2,
		H: menuRect.H + 2,
	}
	_ = sr.renderer.DrawRect(&rect)
	_ = sr.renderer.DrawRect(&menuRect)

	// fill background
	rect = sdl.Rect{
		X: menuRect.X + 1,
		Y: menuRect.Y + 1,
		W: menuRect.W - 2,
		H: menuRect.H - 2,
	}
	_ = sr.renderer.SetDrawColor(types.SGR_COLOR_BACKGROUND.Red, types.SGR_COLOR_BACKGROUND.Green, types.SGR_COLOR_BACKGROUND.Blue, 255)
	_ = sr.renderer.FillRect(&rect)
	_ = sr.renderer.SetDrawColor(notifyColour[types.NOTIFY_QUESTION].Red, notifyColour[types.NOTIFY_QUESTION].Green, notifyColour[types.NOTIFY_QUESTION].Blue, notifyColour[types.NOTIFY_QUESTION].Alpha)
	_ = sr.renderer.FillRect(&rect)

	/*
		TITLE
	*/

	pos := &types.XY{
		X: menuRect.X + _WIDGET_OUTER_MARGIN + sr.notifyIconSize.X,
		Y: menuRect.Y + _WIDGET_OUTER_MARGIN,
	}
	sr.printString(sr.menu.title, notifyColourSgr[types.NOTIFY_QUESTION], pos)

	// draw border
	offset := sr.notifyIconSize.Y
	width = menuRect.W - _WIDGET_OUTER_MARGIN - _WIDGET_OUTER_MARGIN
	_ = sr.renderer.SetDrawColor(types.SGR_COLOR_FOREGROUND.Red, types.SGR_COLOR_FOREGROUND.Green, types.SGR_COLOR_FOREGROUND.Blue, _INPUT_ALPHA)
	rect = sdl.Rect{
		X: menuRect.X + _WIDGET_OUTER_MARGIN - 1,
		Y: menuRect.Y + offset - 1,
		W: width + 2, // menuRect.W - _WIDGET_OUTER_MARGIN - _WIDGET_OUTER_MARGIN + 2,
		H: menuRect.H - offset - _WIDGET_OUTER_MARGIN + 2,
	}
	sr.renderer.DrawRect(&rect)

	rect = sdl.Rect{
		X: menuRect.X + _WIDGET_OUTER_MARGIN,
		Y: menuRect.Y + offset,
		W: width, //menuRect.W - _WIDGET_OUTER_MARGIN - _WIDGET_OUTER_MARGIN,
		H: menuRect.H - offset - _WIDGET_OUTER_MARGIN,
	}
	sr.renderer.DrawRect(&rect)

	// fill background
	sr.renderer.SetDrawColor(types.SGR_COLOR_BACKGROUND.Red, types.SGR_COLOR_BACKGROUND.Green, types.SGR_COLOR_BACKGROUND.Blue, _INPUT_ALPHA_BG)
	rect = sdl.Rect{
		X: menuRect.X + _WIDGET_OUTER_MARGIN + 1,
		Y: menuRect.Y + offset + 1,
		W: width - 2, //menuRect.W - _WIDGET_OUTER_MARGIN - _WIDGET_OUTER_MARGIN - 2,
		H: menuRect.H - offset - _WIDGET_OUTER_MARGIN - 2,
	}
	sr.renderer.FillRect(&rect)

	/*
		MOUSE INTERACTIVE ZONE
	*/

	sr.menu.mouseRect = sdl.Rect{
		X: menuRect.X + _WIDGET_OUTER_MARGIN,
		Y: menuRect.Y + offset + _WIDGET_OUTER_MARGIN,
		W: width,
		H: menuRect.H - offset - _WIDGET_OUTER_MARGIN,
	}

	/*
		OPTIONS
	*/

	filter := sr.menu.readline.Value()
	curPos := sr.menu.readline.CursorPosition()

	offset += _WIDGET_INNER_MARGIN
	for i := range sr.menu.options {
		if sr.menu.options[i] == types.MENU_SEPARATOR {
			if filter != "" {
				sr.menu.hidden[i] = true
				continue
			}
			sr.menu.hidden[i] = false

			// draw horizontal separator
			sr.renderer.SetDrawColor(types.SGR_COLOR_FOREGROUND.Red, types.SGR_COLOR_FOREGROUND.Green, types.SGR_COLOR_FOREGROUND.Blue, 96)
			rect = sdl.Rect{
				X: menuRect.X + _WIDGET_OUTER_MARGIN + _WIDGET_OUTER_MARGIN,
				Y: menuRect.Y + offset + 2 + (sr.glyphSize.Y * int32(i)) + ((sr.glyphSize.Y / 2) - 4),
				W: width - _WIDGET_OUTER_MARGIN - _WIDGET_OUTER_MARGIN,
				H: 4,
			}
			_ = sr.renderer.DrawRect(&rect)
			continue
		}

		if sr.menu.hidden[i] {
			continue
		}

		if sr.menu.incIcons && sr.menu.icons[i] != 0 {
			rectIcon := sdl.Rect{
				X: menuRect.X + _WIDGET_OUTER_MARGIN + (_WIDGET_INNER_MARGIN * 2),
				Y: menuRect.Y + offset + (sr.glyphSize.Y * int32(i)) + 1,
				W: sr.glyphSize.X*2 + dropShadowOffset,
				H: sr.glyphSize.Y + dropShadowOffset, // * 2,
			}
			sr.printCellRect(sr.menu.icons[i], &types.Sgr{Fg: types.SGR_COLOR_FOREGROUND, Bg: types.SGR_COLOR_BACKGROUND, Bitwise: types.SGR_WIDE_CHAR | types.SGR_SPECIAL_FONT_AWESOME}, &rectIcon)
		}

		pos := &types.XY{
			X: menuRect.X + _WIDGET_OUTER_MARGIN + _WIDGET_INNER_MARGIN + optionOffset,
			Y: menuRect.Y + offset + (sr.glyphSize.Y * int32(i)),
		}
		sr.printString(sr.menu.options[i], types.SGR_DEFAULT, pos)

	}

	if surface, ok := sr.notifyIcon[types.NOTIFY_QUESTION].Asset().(*sdl.Surface); ok {
		srcRect := &sdl.Rect{
			X: 0,
			Y: 0,
			W: surface.W,
			H: surface.H,
		}

		dstRect := &sdl.Rect{
			X: menuRect.X,
			Y: menuRect.Y,
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

	sr.AddToOverlayStack(&layer.RenderStackT{texture, windowRect, windowRect, false})
	sr.restoreRendererTexture()

	if filter != "" {
		surface, err := sdl.CreateRGBSurfaceWithFormat(0, windowRect.W, windowRect.H, 32, uint32(sdl.PIXELFORMAT_RGBA32))
		if err != nil {
			panic(err) //TODO: don't panic!
		}
		defer surface.Free()

		sr.menu._renderInputBox(filter, curPos.X, sr, surface, windowRect, &sdl.Rect{
			X: sr.menu.mouseRect.X,
			Y: sr.menu.mouseRect.Y + sr.menu.mouseRect.H + _WIDGET_OUTER_MARGIN,
			W: sr.menu.mouseRect.W,
			H: sr.glyphSize.Y + _WIDGET_OUTER_MARGIN,
		})
	}

	if sr.menu.highlightIndex == _MENU_HIGHLIGHT_HIDDEN {
		return
	}

	rect = sdl.Rect{
		X: menuRect.X + _WIDGET_OUTER_MARGIN + _WIDGET_INNER_MARGIN,
		Y: menuRect.Y + offset + (sr.glyphSize.Y * int32(sr.menu.highlightIndex)),
		W: width - _WIDGET_OUTER_MARGIN,
		H: sr.glyphSize.Y,
	}
	sr._drawHighlightRect(&rect, types.COLOR_SELECTION, types.COLOR_SELECTION, highlightAlphaBorder, highlightAlphaBorder-20)
}

func (menu *menuWidgetT) _renderInputBox(filter string, curPos int32, sr *sdlRender, surface *sdl.Surface, windowRect, rect *sdl.Rect) {
	texture := sr.createRendererTexture()
	if texture == nil {
		return
	}

	// draw border
	_ = sr.renderer.SetDrawColor(types.SGR_COLOR_FOREGROUND.Red, types.SGR_COLOR_FOREGROUND.Green, types.SGR_COLOR_FOREGROUND.Blue, _INPUT_ALPHA)
	borderRect := sdl.Rect{
		X: rect.X - 1,
		Y: rect.Y - 1,
		W: rect.W + 2,
		H: rect.H + 2,
	}
	sr.renderer.DrawRect(&borderRect)
	borderRect = sdl.Rect{
		X: rect.X,
		Y: rect.Y,
		W: rect.W,
		H: rect.H,
	}
	sr.renderer.DrawRect(&borderRect)

	// fill background
	sr.renderer.SetDrawColor(types.SGR_COLOR_BACKGROUND.Red, types.SGR_COLOR_BACKGROUND.Green, types.SGR_COLOR_BACKGROUND.Blue, _INPUT_ALPHA_BG)
	borderRect = sdl.Rect{
		X: rect.X + 1,
		Y: rect.Y + 1,
		W: rect.W - 2,
		H: rect.H - 2,
	}
	sr.renderer.FillRect(&borderRect)

	// value

	textPos := &types.XY{
		X: rect.X + _WIDGET_INNER_MARGIN,
		Y: rect.Y + _WIDGET_INNER_MARGIN,
	}

	sr.printString(filter, types.SGR_DEFAULT, textPos)

	sr._renderNotificationSurface(surface, rect)
	//width = int32(runewidth.StringWidth(sr.menu.filter)) * (sr.glyphSize.X + dropShadowOffset)

	sr.AddToOverlayStack(&layer.RenderStackT{texture, windowRect, windowRect, false})
	sr.restoreRendererTexture()

	if sr.GetBlinkState() {
		cursorRect := sdl.Rect{
			X: textPos.X + curPos,
			Y: textPos.Y,
			W: sr.glyphSize.X,
			H: sr.glyphSize.Y,
		}
		sr._drawHighlightRect(&cursorRect, highlightBorder, highlightFill, 255, 200)
	}
}
