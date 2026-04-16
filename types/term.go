package types

import "time"

type EventIgnoredCallback func()

type SearchMode int

const (
	SEARCH_REGEX SearchMode = iota
	SEARCH_RESULTS
	SEARCH_CLEAR
	SEARCH_CMD_LINES
	SEARCH_AI_PROMPTS
)

type Term interface {
	Start(Pty)
	GetSize() *XY
	GetSgr() *Sgr
	GetCellSgr(*XY) *Sgr
	Resize(*XY)
	Render() bool
	Tile() Tile
	GetCursorPosition() *XY
	CopyRange(*XY, *XY) []byte
	CopyLines(int32, int32) []byte
	CopySquare(*XY, *XY) []byte
	Reply([]byte)
	MouseClick(*XY, MouseButtonT, uint8, ButtonStateT, EventIgnoredCallback)
	MouseWheel(*XY, *XY)
	MouseMotion(*XY, *XY, EventIgnoredCallback)
	IsMouseCaptureEnabled() bool
	MouseHover(*XY)
	ShowCursor(bool)
	SetFocus(bool)
	IsFocused() bool
	MakeVisible(bool)
	Search(SearchMode)
	Match(*XY)
	GetRowId(int32) uint64
	InsertSubTerm(string, string, uint64, BlockMetaFlag, *AiMetaT, time.Time) error
	ConvertRelativeToAbsoluteY(*XY) int32
	FoldAtIndent(*XY) error
	GetTermContents() []byte
	Host(*XY) string
	Pwd(*XY) string
	CmdLine(*XY) string
	Close()
}
