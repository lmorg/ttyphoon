package virtualterm

import (
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
)

func (term *Term) Render() bool {
	if term.Pty.BufSize() > 0 && term._ssLargeBuf.Add(1) < 1000 {
		return false
	}
	term._ssLargeBuf.Store(0)

	term._mutex.Lock()

	screen := term.visibleScreen()

	if !config.Config.TypeFace.Ligatures || term._mouseButtonDown || term._searchHighlight {
		term._renderCells(screen)
	} else {
		term._renderLigs(screen)
	}

	if term._scrollOffset != 0 {
		term.renderer.DrawScrollbar(term.tile, len(term._scrollBuf)-term._scrollOffset, len(term._scrollBuf))
	}

	term._renderOutputBlockChrome(screen)

	term._renderCursor()

	term._mutex.Unlock()

	return true
}

func (term *Term) _renderCells(screen types.Screen) {
	pos := new(types.XY)
	elementStack := make(map[types.Element]bool) // no duplicates

	for ; pos.Y < term.size.Y; pos.Y++ {
		for pos.X = 0; pos.X < term.size.X; pos.X++ {
			switch {
			case screen[pos.Y].Cells[pos.X].Element != nil:
				_, ok := elementStack[screen[pos.Y].Cells[pos.X].Element]
				if !ok {
					elementStack[screen[pos.Y].Cells[pos.X].Element] = true
					offset := screen[pos.Y].Cells[pos.X].GetElementXY()
					screen[pos.Y].Cells[pos.X].Element.Draw(nil, &types.XY{X: pos.X - offset.X, Y: pos.Y - offset.Y})
				}

			case screen[pos.Y].Cells[pos.X].Char == 0:
				continue

			case screen[pos.Y].Cells[pos.X].Sgr == nil:
				continue

			default:
				if screen[pos.Y].Cells[pos.X].Sgr.Bitwise.Is(types.SGR_SLOW_BLINK) && !term.renderer.GetBlinkState() {
					continue // blink
				}
				term.renderer.PrintCell(term.tile, screen[pos.Y].Cells[pos.X], pos)
			}
		}
	}
}

func (term *Term) _renderLigs(screen types.Screen) {
	pos := new(types.XY)
	elementStack := make(map[types.Element]bool) // no duplicates
	row := make([]*types.Cell, term.size.X)

	for ; pos.Y < term.size.Y; pos.Y++ {
		for ; pos.X < term.size.X; pos.X++ {
			switch {
			case screen[pos.Y].Cells[pos.X].Element != nil:
				_, ok := elementStack[screen[pos.Y].Cells[pos.X].Element]
				if !ok {
					elementStack[screen[pos.Y].Cells[pos.X].Element] = true
					offset := screen[pos.Y].Cells[pos.X].GetElementXY()
					screen[pos.Y].Cells[pos.X].Element.Draw(nil, &types.XY{X: pos.X - offset.X, Y: pos.Y - offset.Y})
				}
				row[pos.X] = nil

			case screen[pos.Y].Cells[pos.X].Char == 0:
				row[pos.X] = nil

			case screen[pos.Y].Cells[pos.X].Sgr == nil:
				row[pos.X] = nil

			default:
				if screen[pos.Y].Cells[pos.X].Sgr.Bitwise.Is(types.SGR_SLOW_BLINK) && !term.renderer.GetBlinkState() {
					row[pos.X] = nil // blink
					continue
				}
				row[pos.X] = screen[pos.Y].Cells[pos.X]
			}
		}
		pos.X = 0
		term.renderer.PrintRow(term.tile, row, pos)
	}
}

/*func (term *Term) _renderOutputBlockChrome(screen types.Screen) {
	var (
		foundEnd bool
		i        int32
		meta     types.RowMetaFlag
	)

	term._cacheBlock = [][]int32{}

	for y := int32(len(screen)) - 1; y >= 0; y-- {
		i++
		if len(screen[y].Hidden) != 0 {
			term.renderer.DrawOutputBlockChrome(term.tile, y, 1, _outputBlockChromeColour(screen[y].Hidden[len(screen[y].Hidden)-1].Meta), true)
		}
		switch {
		case screen[y].Meta.Is(types.ROW_OUTPUT_BLOCK_END),
			screen[y].Meta.Is(types.ROW_OUTPUT_BLOCK_ERROR):
			i = 0
			meta = screen[y].Meta
			foundEnd = true

		case screen[y].Meta.Is(types.ROW_OUTPUT_BLOCK_BEGIN):
			if !foundEnd {
				_, row, err := term.outputBlocksFindStartEnd(int32(len(term._scrollBuf)-term._scrollOffset) + y)
				if err != nil {
					continue
				}
				i--
				meta = row[1].Meta
			}

			_renderOutputBlockChrome(term, y, i, meta)
			foundEnd = false
			i = 0
		}
	}

	if foundEnd {
		_renderOutputBlockChrome(term, 0, i, meta)
	}

	if len(term._cacheBlock) == 0 {
		_, row, err := term.outputBlocksFindStartEnd(int32(len(term._scrollBuf) - term._scrollOffset))
		if err != nil {
			return
		}

		meta = row[1].Meta
		_renderOutputBlockChrome(term, 0, int32(len(screen))-1, meta)
	}
}*/

func (term *Term) _renderOutputBlockChrome(screen types.Screen) {
	var begin, y int
	if screen[0].Block.Meta == types.META_BLOCK_NONE {
		begin = -1
	}
	for y = 0; y < len(screen); y++ {
		if len(screen[y].Hidden) != 0 {
			term.renderer.DrawOutputBlockChrome(term.tile, int32(y), 1, _outputBlockChromeColour(screen[y].Hidden[len(screen[y].Hidden)-1].Block.Meta), true)
		}
		if screen[y].RowMeta == types.META_ROW_BEGIN {
			begin = y
		}
		if screen[y].RowMeta.Is(types.META_ROW_END) {
			term.renderer.DrawOutputBlockChrome(term.tile, int32(begin), int32(y-begin), _outputBlockChromeColour(screen[y].Block.Meta), false)
			begin = -1
		}
	}
	if begin != -1 {
		if screen[y-1].Block == nil {
			panic("nil block")
		}
		term.renderer.DrawOutputBlockChrome(term.tile, int32(begin), int32(y-begin), _outputBlockChromeColour(screen[y-1].Block.Meta), false)
	}
}

/*func _renderOutputBlockChrome(term *Term, start, end int32, meta types.RowMetaFlag) {
	//end++
	//if start+end > term.size.Y {
	//	end = term.size.Y - start
	//}

	term.renderer.DrawOutputBlockChrome(term.tile, start, end, _outputBlockChromeColour(meta), false)
	term._cacheBlock = append(term._cacheBlock, []int32{start, end})
}*/

/*func _outputBlockChromeColour(meta types.RowMetaFlag) *types.Colour {
	switch {
	case meta.Is(types.ROW_OUTPUT_BLOCK_ERROR):
		return types.COLOR_ERROR
	case meta.Is(types.ROW_OUTPUT_BLOCK_AI):
		return types.COLOR_AI
	case meta.Is(types.ROW_OUTPUT_BLOCK_END):
		return types.COLOR_OK
	default:
		return types.COLOR_FOLDED
	}
}
*/
