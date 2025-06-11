package types

type MenuCallbackT func(int)

type MenuItem struct {
	Title     string
	Fn        func()
	Highlight func() func()
	Icon      rune
}

type ContextMenu interface {
	Append(...MenuItem)
	DisplayMenu(title string)
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
	Start(*AppWindowTerms, any)
	ShowAndFocusWindow()
	GetWindowSizeCells() *XY
	GetGlyphSize() *XY
	GetBlinkState() bool
	SetBlinkState(bool)
	PrintCell(Tile, *Cell, *XY)
	PrintRow(Tile, []*Cell, *XY)
	DrawScrollbar(Tile, int, int)
	DrawTable(Tile, *XY, int32, []int32)
	DrawHighlightRect(Tile, *XY, *XY)
	DrawRectWithColour(Tile, *XY, *XY, *Colour, bool)
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
	DisplayMenu(string, []string, MenuCallbackT, MenuCallbackT, MenuCallbackT)
	NewContextMenu() ContextMenu
	AddToContextMenu(...MenuItem)
	GetWindowMeta() any
	ResizeWindow(*XY)
	SetKeyboardFnMode(KeyboardMode)
	GetKeyboardModifier() int
	Close()
}

type Image interface {
	Size() *XY
	Asset() any
	Draw(size *XY, pos *XY)
	Close()
}
