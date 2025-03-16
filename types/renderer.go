package types

type MenuCallbackT func(int)

type MenuItem struct {
	Title string
	Fn    func()
	Icon  rune
}

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
	TriggerQuit()
	NewElement(Tile, ElementID) Element
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
