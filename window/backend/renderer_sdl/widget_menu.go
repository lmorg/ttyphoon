package rendersdl

import (
	"fmt"
	"log"
	"sync"

	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/find"
	"github.com/lmorg/ttyphoon/utils/runewidth"
	"github.com/lmorg/ttyphoon/window/backend/cursor"
	"github.com/lmorg/ttyphoon/window/backend/renderer_sdl/layer"
	"github.com/veandco/go-sdl2/sdl"
)

type menuItemRendererT struct {
	label         string
	icon          rune
	callbackIndex int
	hidden        bool
}

var _HIDDEN_PADDING_ITEM = &menuItemRendererT{hidden: true}

type menuWidgetT struct {
	title              string
	incIcons           bool
	_menuOptions       []menuItemRendererT
	menuItems          []*menuItemRendererT
	highlightIndex     int
	_highlightCallback types.MenuCallbackT
	_selectCallback    types.MenuCallbackT
	_cancelCallback    types.MenuCallbackT
	mouseRect          sdl.Rect
	pos                *types.XY
	maxLen             int32
	maxHeight          int
	visible            int
	visOffset          int
	readline           *widgetReadlineT
	filterErr          error
	_hoverFn           func()
	renderMutex        sync.Mutex
	opacity            byte
}

func (menu *menuWidgetT) highlightCallback(index int) {
	menu._highlightCallback(menu.menuItems[index+menu.visOffset].callbackIndex)
}
func (menu *menuWidgetT) selectCallback() {
	index := menu.highlightIndex + menu.visOffset
	if index < 0 || index >= len(menu.menuItems) {
		debug.Log(fmt.Sprintf("%d out of bounds for menuItems[%d] in selectCallback()", index, len(menu.menuItems)))
		return
	}
	menu._selectCallback(menu.menuItems[index].callbackIndex)
}
func (menu *menuWidgetT) cancelCallback() {
	index := menu.highlightIndex + menu.visOffset
	if index < 0 || index >= len(menu.menuItems) {
		debug.Log(fmt.Sprintf("%d out of bounds for menuItems[%d] in cancelCallback()", index, len(menu.menuItems)))
		return
	}
	menu._cancelCallback(menu.menuItems[index].callbackIndex)
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
	if sr.contextMenu == nil {
		log.Println("nil context menu")
		return
	}
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

	items := make([]menuItemRendererT, len(options))
	incIcons := len(icons) != 0

	sr.menu = &menuWidgetT{
		title:              title,
		_menuOptions:       items,
		incIcons:           incIcons,
		_highlightCallback: highlightCallback,
		_selectCallback:    selectCallback,
		_cancelCallback:    cancelCallback,
		highlightIndex:     _MENU_HIGHLIGHT_INIT,
		opacity:            _INPUT_ALPHA,
	}

	if !incIcons {
		icons = make([]rune, len(options))
	}

	var crop = int(sr.winCellSize.X - 10)
	if incIcons {
		crop -= 3
	}

	for i := range options {
		items[i].label = runewidth.Truncate(options[i], int(crop), "…")
		if len(items[i].label) > int(sr.menu.maxLen) {
			sr.menu.maxLen = int32(runewidth.StringWidth(items[i].label))
		}

		items[i].callbackIndex = i
		items[i].icon = icons[i]
	}

	sr.menu.maxHeight = min(int(sr.winCellSize.Y-10), len(options))
	sr.menu.showAll()

	sr.menu.readline = sr.NewReadline(sr.menu.maxLen, "", "", "[Up/Down] Highlight  |  [Return] Choose  |  [Ctrl+c] Cancel  |  [Esc] Vim Mode  | [F4] Translucent")

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

func (menu *menuWidgetT) showAll() {
	menu.visible = len(menu._menuOptions)
	menu.menuItems = make([]*menuItemRendererT, menu.visible)
	for i := range menu.visible {
		menu.menuItems[i] = &menu._menuOptions[i]
	}
}

func (menu *menuWidgetT) updateHidden() {
	menu.renderMutex.Lock()
	defer menu.renderMutex.Unlock()

	filter := menu.readline.Value()
	var ff find.FindT
	ff, menu.filterErr = find.New(filter)

	if filter == "" || menu.filterErr != nil {
		for i := range menu._menuOptions {
			menu._menuOptions[i].hidden = false
		}
		menu.showAll()
		return
	}

	menu.visible = 0
	var j int
	if len(menu._menuOptions) <= menu.maxHeight {
		j = menu.maxHeight
	} else {
		menu.menuItems = []*menuItemRendererT{}
	}

	for i := range menu._menuOptions {
		menu._menuOptions[i].hidden = !ff.MatchString(menu._menuOptions[i].label)

		if !menu._menuOptions[i].hidden {
			menu.visible++
			menu.menuItems = append(menu.menuItems, &menu._menuOptions[i])
			j++
		}
	}

	for ; j < menu.maxHeight; j++ {
		menu.menuItems = append(menu.menuItems, _HIDDEN_PADDING_ITEM)
	}
}

func (menu *menuWidgetT) updateHighlight(adjust int) {
	if adjust == 0 {
		if menu.highlightIndex < 0 {
			menu.highlightIndex = 0
		}
		if menu.menuItems[menu.highlightIndex].hidden {
			adjust = 1
		}
	}

	var attempts int
	for {
		attempts++
		menu.highlightIndex += adjust

		if menu.highlightIndex >= menu.maxHeight {
			menu.highlightIndex = 0
		} else if menu.highlightIndex < 0 {
			menu.highlightIndex = menu.maxHeight - 1
		}

		if attempts > menu.maxHeight {
			menu.highlightIndex = _MENU_HIGHLIGHT_HIDDEN
			return
		}

		if menu.menuItems[menu.highlightIndex].hidden {
			continue
		}

		if menu.menuItems[menu.highlightIndex].label != types.MENU_SEPARATOR {
			break
		}
	}

	menu.highlightCallback(menu.highlightIndex)
}

func (menu *menuWidgetT) eventTextInput(sr *sdlRender, evt *sdl.TextInputEvent) {
	menu.readline.eventTextInput(sr, evt)
}

func (menu *menuWidgetT) eventKeyPress(sr *sdlRender, evt *sdl.KeyboardEvent) {
	switch evt.Keysym.Sym {
	case sdl.K_RETURN, sdl.K_RETURN2, sdl.K_KP_ENTER:
		if menu.highlightIndex < 0 {
			return
		}
		sr.closeMenu()
		menu.selectCallback()
		return

	case sdl.K_UP:
		menu.updateHighlight(-1)
		return

	case sdl.K_DOWN, sdl.K_TAB:
		menu.updateHighlight(1)
		return

	case sdl.K_F4:
		if menu.opacity == _INPUT_ALPHA {
			menu.opacity = 32
		} else {
			menu.opacity = _INPUT_ALPHA
		}

	}

	menu.readline.eventKeyPress(sr, evt)
}

func (menu *menuWidgetT) eventMouseButton(sr *sdlRender, evt *sdl.MouseButtonEvent) {
	if evt.State != sdl.RELEASED {
		return
	}
	if evt.Button != 1 {
		sr.closeMenu()
		menu.cancelCallback()
		return
	}

	i := menu._mouseHover(evt.X, evt.Y, sr.glyphSize)
	menu._mouseMotion(evt.X, evt.Y, sr)
	if i == -1 {
		sr.closeMenu()
		menu.cancelCallback()
		return
	}

	sr.closeMenu()
	menu.selectCallback()
}

func (menu *menuWidgetT) eventMouseWheel(sr *sdlRender, evt *sdl.MouseWheelEvent) {
	if evt.MouseX < menu.mouseRect.X || evt.MouseX > menu.mouseRect.X+menu.mouseRect.W ||
		evt.MouseY < menu.mouseRect.Y || evt.MouseY > menu.mouseRect.Y+menu.mouseRect.H {
		sr.termWidget.eventMouseWheel(sr, evt)
		return
	}

	var offset int
	if evt.Direction == sdl.MOUSEWHEEL_FLIPPED {
		offset = int(-evt.Y)
	} else {
		offset = int(evt.Y)
	}

	menu.visOffset -= offset

	if menu.visOffset < 0 {
		menu.visOffset = 0
	}
	if menu.visOffset+menu.maxHeight >= menu.visible {
		menu.visOffset = menu.visible - menu.maxHeight
	}

	menu.updateHidden()
	menu.updateHighlight(0)
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

	if menu.menuItems[i].hidden {
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

	if i >= menu.maxHeight {
		return -1
	}

	if menu.menuItems[i].label == types.MENU_SEPARATOR {
		menu.highlightIndex = _MENU_HIGHLIGHT_HIDDEN
		return -1
	}

	return i
}

func (sr *sdlRender) renderMenu(windowRect *sdl.Rect) {
	sr.menu.renderMutex.Lock()
	defer sr.menu.renderMutex.Unlock()
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
	height := (sr.glyphSize.Y * int32(sr.menu.maxHeight)) + (_WIDGET_OUTER_MARGIN * 2) + sr.notifyIconSize.Y
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
	_ = sr.renderer.SetDrawColor(types.SGR_COLOR_BACKGROUND.Red, types.SGR_COLOR_BACKGROUND.Green, types.SGR_COLOR_BACKGROUND.Blue, sr.menu.opacity)
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
	_ = sr.renderer.SetDrawColor(types.SGR_COLOR_FOREGROUND.Red, types.SGR_COLOR_FOREGROUND.Green, types.SGR_COLOR_FOREGROUND.Blue, sr.menu.opacity)
	rect = sdl.Rect{
		X: menuRect.X + _WIDGET_OUTER_MARGIN - 1,
		Y: menuRect.Y + offset - 1,
		W: width + 2,
		H: menuRect.H - offset - _WIDGET_OUTER_MARGIN + 2,
	}
	sr.renderer.DrawRect(&rect)

	rect = sdl.Rect{
		X: menuRect.X + _WIDGET_OUTER_MARGIN,
		Y: menuRect.Y + offset,
		W: width,
		H: menuRect.H - offset - _WIDGET_OUTER_MARGIN,
	}
	sr.renderer.DrawRect(&rect)

	// fill background
	sr.renderer.SetDrawColor(types.SGR_COLOR_BACKGROUND.Red, types.SGR_COLOR_BACKGROUND.Green, types.SGR_COLOR_BACKGROUND.Blue, sr.menu.opacity)
	rect = sdl.Rect{
		X: menuRect.X + _WIDGET_OUTER_MARGIN + 1,
		Y: menuRect.Y + offset + 1,
		W: width - 2,
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
	i := -1
	for index, item := range sr.menu.menuItems {
		if index < sr.menu.visOffset {
			continue
		}
		if i == sr.menu.maxHeight {
			break
		}
		i++

		if item.label == types.MENU_SEPARATOR {
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

		if item.hidden {
			continue
		}

		if sr.menu.incIcons && item.icon != 0 &&
			(sr.menu.opacity == _INPUT_ALPHA || sr.menu.highlightIndex == i) {
			rectIcon := sdl.Rect{
				X: menuRect.X + _WIDGET_OUTER_MARGIN + (_WIDGET_INNER_MARGIN * 2),
				Y: menuRect.Y + offset + (sr.glyphSize.Y * int32(i)) + 1,
				W: sr.glyphSize.X*2 + dropShadowOffset,
				H: sr.glyphSize.Y + dropShadowOffset,
			}
			sr.printCellRect(item.icon, &types.Sgr{Fg: types.SGR_COLOR_FOREGROUND, Bg: types.SGR_COLOR_BACKGROUND, Bitwise: types.SGR_WIDE_CHAR | types.SGR_SPECIAL_FONT_AWESOME}, &rectIcon)
		}

		pos := &types.XY{
			X: menuRect.X + _WIDGET_OUTER_MARGIN + _WIDGET_INNER_MARGIN + optionOffset,
			Y: menuRect.Y + offset + (sr.glyphSize.Y * int32(i)),
		}
		if sr.menu.opacity == _INPUT_ALPHA || sr.menu.highlightIndex == i {
			sr.printString(item.label, types.SGR_DEFAULT, pos)
		}
	}

	if sr.menu.opacity == _INPUT_ALPHA {
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
	}

	sr.AddToOverlayStack(&layer.RenderStackT{texture, windowRect, windowRect, false})
	sr.restoreRendererTexture()

	if len(sr.menu._menuOptions) > sr.menu.maxHeight && sr.menu.opacity == _INPUT_ALPHA {
		sr.drawGaugeV(&sdl.Rect{
			X: menuRect.X + ((sr.menu.maxLen + 1) * sr.glyphSize.X) - 2,
			Y: menuRect.Y + (sr.glyphSize.X * 5),
			W: sr.glyphSize.X,
			H: int32(sr.menu.maxHeight) * sr.glyphSize.Y,
		}, min(sr.menu.maxHeight+sr.menu.visOffset, sr.menu.visible), sr.menu.visible, types.SGR_COLOR_GREEN)
	}

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

	if sr.menu.highlightIndex != _MENU_HIGHLIGHT_HIDDEN {
		rect = sdl.Rect{
			X: menuRect.X + _WIDGET_OUTER_MARGIN + _WIDGET_INNER_MARGIN,
			Y: menuRect.Y + offset + (sr.glyphSize.Y * int32(sr.menu.highlightIndex)),
			W: width - _WIDGET_OUTER_MARGIN,
			H: sr.glyphSize.Y,
		}
		sr._drawHighlightRect(&rect, types.COLOR_SELECTION, types.COLOR_SELECTION, highlightAlphaBorder, highlightAlphaBorder-20)
	}
}

func (menu *menuWidgetT) _renderInputBox(filter string, curPos int32, sr *sdlRender, surface *sdl.Surface, windowRect, rect *sdl.Rect) {
	texture := sr.createRendererTexture()
	if texture == nil {
		return
	}

	// draw border
	_ = sr.renderer.SetDrawColor(types.SGR_COLOR_FOREGROUND.Red, types.SGR_COLOR_FOREGROUND.Green, types.SGR_COLOR_FOREGROUND.Blue, sr.menu.opacity)
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
	if menu.filterErr == nil {
		sr.renderer.SetDrawColor(types.SGR_COLOR_BACKGROUND.Red, types.SGR_COLOR_BACKGROUND.Green, types.SGR_COLOR_BACKGROUND.Blue, sr.menu.opacity)
	} else {
		sr.renderer.SetDrawColor(types.SGR_COLOR_RED.Red, types.SGR_COLOR_RED.Green, types.SGR_COLOR_RED.Blue, sr.menu.opacity)
	}
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
