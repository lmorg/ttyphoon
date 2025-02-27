package virtualterm

import (
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
)

func (term *Term) Render() {
	if !term.visible {
		return
	}

	term._mutex.Lock()

	screen := term.visibleScreen()

	if !config.Config.TypeFace.Ligatures || term._mouseButtonDown || term._searchHighlight {
		term._renderCells(screen)
	} else {
		term._renderLigs(screen)
	}

	term._renderOutputBlockChrome(screen)

	term._renderCursor()

	term._mutex.Unlock()
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
					offset := getElementXY(screen[pos.Y].Cells[pos.X].Char)
					screen[pos.Y].Cells[pos.X].Element.Draw(nil, &types.XY{X: pos.X - offset.X, Y: pos.Y - offset.Y})
				}

			case screen[pos.Y].Cells[pos.X].Char == 0:
				continue

			case screen[pos.Y].Cells[pos.X].Sgr == nil:
				continue

			default:
				if screen[pos.Y].Cells[pos.X].Sgr.Bitwise.Is(types.SGR_SLOW_BLINK) && !term._slowBlinkState {
					continue // blink
				}
				term.renderer.PrintCell(term.tileId, screen[pos.Y].Cells[pos.X], pos)
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
					offset := getElementXY(screen[pos.Y].Cells[pos.X].Char)
					screen[pos.Y].Cells[pos.X].Element.Draw(nil, &types.XY{X: pos.X - offset.X, Y: pos.Y - offset.Y})
				}
				row[pos.X] = nil

			case screen[pos.Y].Cells[pos.X].Char == 0:
				row[pos.X] = nil

			case screen[pos.Y].Cells[pos.X].Sgr == nil:
				row[pos.X] = nil

			default:
				if screen[pos.Y].Cells[pos.X].Sgr.Bitwise.Is(types.SGR_SLOW_BLINK) && !term._slowBlinkState {
					row[pos.X] = nil // blink
					continue
				}
				row[pos.X] = screen[pos.Y].Cells[pos.X]
			}
		}
		pos.X = 0
		term.renderer.PrintRow(term.tileId, row, pos)
	}
}

func (term *Term) _renderOutputBlockChrome(screen types.Screen) {
	var (
		foundEnd   bool
		i          int32
		errorBlock bool
	)

	term._cacheBlock = [][]int32{}

	for y := int32(len(screen)) - 1; y >= 0; y-- {
		i++
		if len(screen[y].Hidden) != 0 {
			term.renderer.DrawOutputBlockChrome(term.tileId, y, 1, types.COLOUR_FOLDED, true)
		}
		if screen[y].Meta.Is(types.ROW_OUTPUT_BLOCK_END) {
			i = 0
			errorBlock = false
			foundEnd = true
		}
		if screen[y].Meta.Is(types.ROW_OUTPUT_BLOCK_ERROR) {
			i = 0
			errorBlock = true
			foundEnd = true
		}

		if screen[y].Meta.Is(types.ROW_OUTPUT_BLOCK_BEGIN) {
			if !foundEnd {
				_, row, err := term.outputBlocksFindStartEnd(int32(len(term._scrollBuf)-term._scrollOffset) + y)
				if err != nil {
					continue
				}
				i--
				errorBlock = row[1].Meta.Is(types.ROW_OUTPUT_BLOCK_ERROR)
			}

			_renderOutputBlockChrome(term, y, i, errorBlock)
			foundEnd = false
			i = 0
		}
	}

	if foundEnd {
		_renderOutputBlockChrome(term, 0, i, errorBlock)
	}

	if len(term._cacheBlock) == 0 {
		_, row, err := term.outputBlocksFindStartEnd(int32(len(term._scrollBuf) - term._scrollOffset))
		if err != nil {
			return
		}

		errorBlock = row[1].Meta.Is(types.ROW_OUTPUT_BLOCK_ERROR)
		_renderOutputBlockChrome(term, 0, int32(len(screen))-1, errorBlock)
	}
}

func _renderOutputBlockChrome(term *Term, start, end int32, errorBlock bool) {
	end++
	if errorBlock {
		term.renderer.DrawOutputBlockChrome(term.tileId, start, end, types.COLOUR_ERROR, false)
	} else {
		term.renderer.DrawOutputBlockChrome(term.tileId, start, end, types.COLOUR_OK, false)
	}
	term._cacheBlock = append(term._cacheBlock, []int32{start, end})
}
