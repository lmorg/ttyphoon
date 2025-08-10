package types

type Element interface {
	Generate(*ApcSlice) error
	Write(rune) error
	Rune(*XY) rune
	Size() *XY
	Draw(*XY)
	MouseClick(*XY, MouseButtonT, uint8, ButtonStateT, EventIgnoredCallback)
	MouseWheel(*XY, *XY, EventIgnoredCallback)
	MouseMotion(*XY, *XY, EventIgnoredCallback)
	MouseHover(curPosTile *XY, curPosElement *XY) func()
	MouseOut()
}

type ElementID int

const (
	ELEMENT_ID_IMAGE ElementID = iota
	ELEMENT_ID_SIXEL
	ELEMENT_ID_CSV
	ELEMENT_ID_MARKDOWN_TABLE
	ELEMENT_ID_HYPERLINK
	ELEMENT_ID_CODEBLOCK
)
