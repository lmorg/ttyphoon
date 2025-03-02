package types

type MenuCallbackT func(int)

type MenuItem struct {
	Title string
	Fn    func()
}

type Renderer interface {
	Start(*TermWindow, any)
	ShowAndFocusWindow()
	GetTermSize(TileId) *XY
	GetWindowSizeCells() *XY
	GetGlyphSize() *XY
	PrintCell(TileId, *Cell, *XY)
	PrintRow(TileId, []*Cell, *XY)
	DrawScrollbar(TileId, int, int)
	DrawTable(TileId, *XY, int32, []int32)
	DrawHighlightRect(TileId, *XY, *XY)
	DrawRectWithColour(TileId, *XY, *XY, *Colour, bool)
	DrawOutputBlockChrome(TileId, int32, int32, *Colour, bool)
	GetWindowTitle() string
	SetWindowTitle(string)
	StatusBarText(string)
	RefreshWindowList()
	Bell()
	TriggerRedraw()
	TriggerQuit()
	NewElement(TileId, ElementID) Element
	DisplayNotification(NotificationType, string)
	DisplaySticky(NotificationType, string) Notification
	DisplayInputBox(string, string, func(string))
	AddToContextMenu(...MenuItem)
	DisplayMenu(string, []string, MenuCallbackT, MenuCallbackT, MenuCallbackT)
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
