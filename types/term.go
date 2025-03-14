package types

import (
	"os"
)

type EventIgnoredCallback func()

type TermWindow struct {
	Tiles  []*Tile
	Active *Tile
}

type Tile struct {
	Top    int32
	Left   int32
	Right  int32
	Bottom int32
	Term   Term
	PaneId string
}

type Term interface {
	Start(Pty)
	GetSize() *XY
	Resize(*XY)
	Render() bool
	CopyRange(*XY, *XY) []byte
	CopyLines(int32, int32) []byte
	CopySquare(*XY, *XY) []byte
	Bg() *Colour
	Reply([]byte)
	MouseClick(*XY, MouseButtonT, uint8, ButtonStateT, EventIgnoredCallback)
	MouseWheel(*XY, *XY)
	MouseMotion(*XY, *XY, EventIgnoredCallback)
	MousePosition(*XY)
	ShowCursor(bool)
	HasFocus(bool)
	MakeVisible(bool)
	Search()
	ShowSearchResults()
	Match(*XY)
	FoldAtIndent(*XY) error
	Close()
}

type Pty interface {
	File() *os.File
	Read() (rune, error)
	Write([]byte) error
	Resize(*XY) error
	BufSize() int
	Close()
}
