package virtualterm

import (
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

func interactiveCell(screen types.Screen, pos *types.XY) *types.Cell {
	cell := screen[pos.Y].Cells[pos.X]
	if cell.Char == 0 && cell.Element == nil && pos.X > 0 {
		previous := screen[pos.Y].Cells[pos.X-1]
		if previous.HasTextElement() && previous.Sgr != nil && previous.Sgr.Bitwise.Is(types.SGR_WIDE_CHAR) {
			return previous
		}
	}
	return cell
}

func (term *Term) textElementHover(screen types.Screen, element types.Element) func() {
	return func() {
		if !config.Config.Window.HoverEffectHighlight {
			return
		}
		for y, row := range screen {
			for x := 0; x < len(row.Cells); {
				if row.Cells[x].Element != element {
					x++
					continue
				}
				start := x
				end := x + 1
				for x < len(row.Cells) && row.Cells[x].Element == element {
					width := 1
					if row.Cells[x].Sgr != nil && row.Cells[x].Sgr.Bitwise.Is(types.SGR_WIDE_CHAR) {
						width = 2
					}
					end = x + width
					x += width
				}
				term.renderer.DrawHighlightRect(term.tile, &types.XY{X: int32(start), Y: int32(y)},
					&types.XY{X: int32(min(end, len(row.Cells)) - start), Y: 1})
			}
		}
	}
}

func (term *Term) textElementRenderCell(cell *types.Cell) *types.Cell {
	if !cell.HasTextElement() || cell.Element != term._mouseIn || config.Config.Window.HoverEffectHighlight {
		return cell
	}
	underlined := *cell
	underlined.Sgr = cell.Sgr.Copy()
	underlined.Sgr.Bitwise.Set(types.SGR_UNDERLINE)
	return &underlined
}
