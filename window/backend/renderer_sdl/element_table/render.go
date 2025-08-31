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

	el.renderScrollbars(pos.Y, types.SGR_COLOR_BACKGROUND)
}

func (el *ElementTable) renderScrollbars(posY int32, c *types.Colour) {
	el.renderScrollbarVertical(posY, c)
	el.renderScrollbarHorizontal(posY, c)
}

func (el *ElementTable) renderScrollbarHorizontal(posY int32, c *types.Colour) {
	termSize := el.tile.GetTerm().GetSize()
	topleft := &types.XY{X: 0, Y: posY + el.size.Y}
	if topleft.Y > termSize.Y {
		topleft.Y = termSize.Y
	}

	tableWidth := el.boundaries[len(el.boundaries)-1]

	if tableWidth < termSize.X {
		return
	}

	el.renderer.DrawGaugeH(el.tile, topleft, termSize.X-1, int(-el.renderOffset+termSize.X), int(tableWidth), c)
}

func (el *ElementTable) renderScrollbarVertical(posY int32, c *types.Colour) {
	termSize := el.tile.GetTerm().GetSize()
	topleft := &types.XY{X: termSize.X, Y: posY + 2}
	height := el.size.Y
	if topleft.Y+height > termSize.Y {
		height = termSize.Y - posY
	}

	if height >= el.lines {
		return
	}
	
	el.renderer.DrawGaugeV(el.tile, topleft, height-2, int(el.limitOffset+el.size.Y), int(el.lines), c)
}
