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
	GetCursorPosition() *XY
	CopyRange(*XY, *XY) []byte
	CopyLines(int32, int32) []byte
	CopySquare(*XY, *XY) []byte
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
	SearchCmdLines()
	SearchAiPrompts()
	Match(*XY)
	InsertSubTerm(string, string, int32, RowMetaFlag) error
	ConvertRelativeToAbsoluteY(*XY) int32
	FoldAtIndent(*XY) error
	GetTermContents() []byte
	Host(*XY) string
	Pwd(*XY) string
	CmdLine(*XY) string
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
