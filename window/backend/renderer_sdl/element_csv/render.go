package element_csv

import (
	"github.com/lmorg/mxtty/types"
)

func (el *ElementCsv) Draw(size *types.XY, pos *types.XY) {
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

	cell.Sgr.Bitwise.Unset(types.SGR_INVERT)
	cell.Sgr.Fg = types.SGR_COLOR_RED

	cell.Char = arrowGlyph[el.orderDesc]
	el.renderer.PrintCell(el.tile, cell, relPos)

	cell.Sgr.Fg = types.SGR_COLOR_FOREGROUND

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

	if el.highlight != nil {
		var start, end int32

		for i := range el.boundaries {
			if el.highlight.X-el.renderOffset < el.boundaries[i] {
				if i != 0 {
					start = el.boundaries[i-1] + pos.X
					end = int32(el.width[i]) + 2
				} else {
					end = int32(el.width[i]) + 2 + el.renderOffset
				}
				if start+end > el.size.X {
					end = el.size.X - start
				}
				break
			}
		}

		el.renderer.DrawHighlightRect(el.tile,
			&types.XY{X: start, Y: el.highlight.Y + pos.Y},
			&types.XY{X: end, Y: 1},
		)
	}
}
