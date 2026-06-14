package types

import (
	"context"
)

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

type InputBoxCallbackT func(*InputBoxCallbackResultT)

type InputBoxCallbackResultT struct {
	Value     string         `json:"value"`
	Variables map[string]any `json:"variables"`
}

func (ibcr InputBoxCallbackResultT) String() string {
	return ibcr.Value
}

type InputBoxWT struct {
	Options    InputBoxWTOptions
	OkFunc     InputBoxCallbackT
	CancelFunc InputBoxCallbackT
}

type InputBoxWTOptions struct {
	Title       string                `json:"title"`
	Prefill     string                `json:"prefill"`
	Placeholder string                `json:"placeholder"`
	History     []string              `json:"history"`
	Multiline   bool                  `json:"multiline"`
	Variables   []InputBoxWTVariables `json:"variables"`
}

type InputBoxWTVariables struct {
	Name        string   `json:"name" yaml:"name"`
	Label       string   `json:"label" yaml:"label"`
	Description string   `json:"description" yaml:"description"`
	Default     string   `json:"default" yaml:"default"`
	Options     []string `json:"options" yaml:"options"`
	Type        string   `json:"type" yaml:"type"`
}

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
	NewElement(Tile, ElementID, ...any) Element
	DisplayNotification(NotificationType, string)
	DisplaySticky(NotificationType, string, func()) Notification
	DisplayInputBox(string, string, InputBoxCallbackT, InputBoxCallbackT)
	DisplayInputBoxW(*InputBoxWT)
	DisplayMenu(title string, items []string, highlight MenuCallbackT, ok MenuCallbackT, cancel MenuCallbackT)
	NewContextMenu() ContextMenu
	AddToContextMenu(...MenuItem)
	DisplayMarkdownModel(string)
	ResizeWindow(*XY)
	SetKeyboardFnMode(KeyboardMode)
	GetKeyboardModifier() int
	RefreshNotes()
	NotesEditFile(filename string)
	NotesCreateAndOpen(filename, contents string)
	DisplayImageFullscreen(dataURL string, sourceWidth, sourceHeight int32)
	ActiveTile() Tile
	GetWindowContext() context.Context
	AskAi()
	Close()
}

type Image interface {
	Size() *XY
	Asset() any
	Draw(tile Tile, size *XY, pos *XY)
	Close()
}
