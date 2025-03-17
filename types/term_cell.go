package types

import (
	"strings"
)

type Cell struct {
	Char    rune
	Sgr     *Sgr
	Element Element
	Phrase  *[]rune
}

func (c *Cell) Clear() {
	c.Char = 0
	c.Sgr = &Sgr{}
	c.Element = nil
}

func (c *Cell) Rune() rune {
	switch {
	case c.Element != nil:
		return c.Element.Rune(c.GetElementXY())

	case c.Char == 0:
		return ' '

	default:
		return c.Char
	}
}

const (
	_CELL_ELEMENTXY_MASK    = (^int32(0)) << 16
	_CELL_ELEMENTXY_CEILING = int32(^uint16(0) >> 1)
)

func SetElementXY(xy *XY) rune {
	if xy.X > _CELL_ELEMENTXY_CEILING || xy.Y > _CELL_ELEMENTXY_CEILING {
		panic("TODO: proper error handling")
	}
	return (xy.X << 16) | xy.Y
}

func (c *Cell) GetElementXY() *XY {
	return &XY{
		X: c.Char >> 16,
		Y: c.Char &^ _CELL_ELEMENTXY_MASK,
	}
}

/*
	ROWS
*/

type Row struct {
	Id     uint64
	Cells  []*Cell
	Meta   RowMetaFlag
	Hidden Screen
	Phrase *[]rune
}

type RowMetaFlag uint16

// Flags
const (
	ROW_META_NONE          RowMetaFlag = 0
	ROW_OUTPUT_BLOCK_BEGIN RowMetaFlag = 1 << iota
	ROW_OUTPUT_BLOCK_END
	ROW_OUTPUT_BLOCK_ERROR
	ROW_META_COLLAPSED
)

func (f RowMetaFlag) Is(flag RowMetaFlag) bool {
	return f&flag != 0
}

func (f *RowMetaFlag) Set(flag RowMetaFlag) {
	*f |= flag
}

func (f *RowMetaFlag) Unset(flag RowMetaFlag) {
	*f &^= flag
}

func (r *Row) String() string {
	slice := make([]rune, len(r.Cells))

	for i, cell := range r.Cells {
		slice[i] = cell.Rune()
	}

	return string(slice)
}

/*
	SCREEN
*/

type Screen []*Row

func (screen *Screen) String() string {
	slice := make([]string, len(*screen))
	for i, row := range *screen {
		slice[i] = row.String()
	}

	return strings.Join(slice, "\n")
}
