package types

import (
	"os"
)

type EventIgnoredCallback func()

type TermWin struct {
	Terms  []Term
	Cords  [][2]*XY
	Active Term
}

type Term interface {
	Start(Pty)
	GetSize() *XY
	Resize(*XY)
	Render()
	CopyRange(*XY, *XY) []byte
	CopyLines(int32, int32) []byte
	CopySquare(*XY, *XY) []byte
	Bg() *Colour
	Reply([]byte)
	MouseClick(*XY, uint8, uint8, bool, EventIgnoredCallback)
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
	Close()
}
