package types

import (
	"os"
)

type EventIgnoredCallback func()

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
	GetTermContents() []byte
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
