package types

import "context"

type MenuCallbackT func(int)

type MenuItem struct {
	Title     string
	Fn        func()
	Highlight func() func()
	Icon      rune
	// WebkitContextHighlightPersistent is set by the webkit renderer for
	// AddToContextMenu() items so Highlight()'s returned function is treated as a
	// per-frame draw callback while the item is highlighted.
	WebkitContextHighlightPersistent bool
}

type ContextMenu interface {
	Append(...MenuItem)
	DisplayMenu(title string, showNextToMouseCursor ...bool)
	Options() []string
	Icons() []rune
	Highlight(int)
	Callback(int)
	Cancel(int)
	MenuItems() []MenuItem
}

const MENU_SEPARATOR = "-"

type InputBoxCallbackT func(string)

type Renderer interface {
	Start(*AppWindowTerms, any, context.Context)
	ShowAndFocusWindow()
	GetWindowSizeCells() *XY
	GetGlyphSize() *XY
	GetBlinkState() bool
	SetBlinkState(bool)
	PrintCell(Tile, *Cell, *XY)
	PrintRow(Tile, []*Cell, *XY)
	DrawFrame(tile Tile)
	DrawGaugeH(tile Tile, topLeft *XY, width int32, value, max int, c *Colour)
	DrawGaugeV(tile Tile, topLeft *XY, height int32, value, max int, c *Colour)
	DrawTable(Tile, *XY, int32, []int32)
	DrawHighlightRect(Tile, *XY, *XY)
	DrawRectWithColour(Tile, *XY, *XY, *Colour, bool)
	DrawRectWithColourAndBorder(Tile, *XY, *XY, *Colour, bool, bool)
	DrawOutputBlockChrome(Tile, int32, int32, *Colour, bool)
	GetWindowTitle() string
	SetWindowTitle(string)
	StatusBarText(string)
	RefreshWindowList()
	Bell()
	TriggerRedraw()
	TriggerLazyRedraw()
	TriggerDeallocation(func())
	TriggerQuit()
	NewElement(Tile, ElementID) Element
	DisplayNotification(NotificationType, string)
	DisplaySticky(NotificationType, string, func()) Notification
	DisplayInputBox(string, string, InputBoxCallbackT, InputBoxCallbackT)
	DisplayMenu(title string, items []string, highlight MenuCallbackT, ok MenuCallbackT, cancel MenuCallbackT)
	NewContextMenu() ContextMenu
	AddToContextMenu(...MenuItem)
	GetWindowMeta() any
	ResizeWindow(*XY)
	SetKeyboardFnMode(KeyboardMode)
	GetKeyboardModifier() int
	NotesCreateAndOpen(filename, contents string)
	EmitAIResponseChunk(chunk string)
	DisplayImageFullscreen(dataURL string, sourceWidth, sourceHeight int32)
	ActiveTile() Tile
	GetContext() context.Context
	Close()
}

type Image interface {
	Size() *XY
	Asset() any
	Draw(tile Tile, size *XY, pos *XY)
	Close()
}
