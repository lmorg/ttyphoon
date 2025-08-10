package element_table

import (
	"github.com/lmorg/mxtty/types"
)

func (el *ElementTable) Draw(pos *types.XY) {
	pos.X += el.renderOffset

	cell := &types.Cell{Sgr: &types.Sgr{}}
	cell.Sgr.Reset()
	relPos := &types.XY{X: pos.X, Y: pos.Y}

	cell.Sgr.Bitwise.Set(types.SGR_INVERT)
	for i := range el.top {
		cell.Char = el.top[i]
		el.renderer.PrintCell(el.tile, cell, relPos)
		relPos.X++
	}

	switch el.orderByIndex {
	case 0:
		goto skipOrderGlyph

	case 1:
		relPos.X = pos.X + 0

	default:
		relPos.X = pos.X + el.boundaries[el.orderByIndex-2]
	}

	cell.Sgr.Bg = types.SGR_COLOR_RED
	cell.Sgr.Bitwise.Set(types.SGR_BOLD)

	cell.Char = arrowGlyph[el.orderDesc]
	el.renderer.PrintCell(el.tile, cell, relPos)

	cell.Sgr.Bitwise.Unset(types.SGR_BOLD)
	cell.Sgr.Bg = types.SGR_COLOR_BACKGROUND

skipOrderGlyph:

	relPos.Y++
	cell.Sgr.Bitwise.Unset(types.SGR_INVERT)

	for y := int32(0); y < el.size.Y-1 && int(y) < len(el.table); y++ {
		relPos.X = 0
		for x := -el.renderOffset; x+el.renderOffset < el.size.X && int(x) < len(el.table[y]); x++ {
			cell.Char = el.table[y][x]
			el.renderer.PrintCell(el.tile, cell, relPos)
			relPos.X++
		}
		relPos.Y++
	}

	el.renderer.DrawTable(el.tile, pos, int32(len(el.table)), el.boundaries)
}
