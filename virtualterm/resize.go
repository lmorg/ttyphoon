package virtualterm

import (
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

func sanitiseResizeSize(size *types.XY) *types.XY {
	if size == nil {
		return &types.XY{X: 1, Y: 1}
	}

	if size.X < 1 {
		size.X = 1
	}
	if size.Y < 1 {
		size.Y = 1
	}

	return size
}

func (term *Term) Resize(size *types.XY) {
	term._mutex.Lock()
	defer term._mutex.Unlock()

	if term.size == nil {
		term.size = &types.XY{X: 1, Y: 1}
	}
	size = sanitiseResizeSize(size)

	xDiff := int32(size.X - term.size.X)
	yDiff := int(size.Y - term.size.Y)

	debug.Log(term.size)
	debug.Log(size)

	term._resizeNestedScreenWidth(term._scrollBuf, xDiff)
	term._resizeNestedScreenWidth(term._normBuf, xDiff)
	term._resizeNestedScreenWidth(term._altBuf, xDiff)

	// This needs to be after xDiff but before yDiff!
	term.size = size

	switch {
	case yDiff == 0:
		// nothing to do

	case yDiff > 0:
		// grow
		term._resizeFromTop(yDiff)

	case yDiff < 0:
		// shrink
		fromBottom := term._resizeFromBottom(-yDiff)
		if fromBottom > 0 {
			term._normBuf = term._normBuf[:len(term._normBuf)-fromBottom]
			term._altBuf = term._altBuf[:len(term._altBuf)-fromBottom]
			yDiff += fromBottom
		}
		term.appendScrollBuf(-yDiff)
		term._normBuf = term._normBuf[-yDiff:]
		term._altBuf = term._altBuf[-yDiff:]
	}

	term.resizePty()
}

func (term *Term) _resizeFromTop(max int) {
	for i := 0; i < max; i++ {
		term._altBuf = append(term._altBuf, term.makeRow())
	}

	if len(term._scrollBuf) > max {
		newScreen := make([]*types.Row, term.size.Y)
		copy(newScreen, term._scrollBuf[len(term._scrollBuf)-max:])
		copy(newScreen[max:], term._normBuf)
		term._normBuf = newScreen

		term._scrollBuf = term._scrollBuf[:len(term._scrollBuf)-max]

		if !term.IsAltBuf() {
			term._curPos.Y += int32(max)
		}
		return
	}

	l := len(term._scrollBuf)
	offset := max - l
	term._normBuf = append(term._scrollBuf, term._normBuf...)
	term._scrollBuf = types.Screen{}
	for i := 0; i < offset; i++ {
		term._normBuf = append(term._normBuf, term.makeRow())
	}

	if !term.IsAltBuf() {
		term._curPos.Y += int32(l)
	}
}

func (term *Term) _resizeFromBottom(max int) int {
	if len(term._scrollBuf) > 0 || term.IsAltBuf() {
		return 0
	}
	if max > len(term._normBuf) {
		max = len(term._normBuf)
	}

	var i int
	for y := len(term._normBuf) - 1; i < max && y >= 0; y-- {
		if !rowIsBlank(term._normBuf[y]) {
			//debug.Log(i)
			return i
		}
		i++
	}

	return i
}

func rowIsBlank(row *types.Row) bool {
	for i := range row.Cells {
		if row.Cells[i].Rune() != ' ' {
			return false
		}
	}

	return true
}

func (term *Term) _resizeNestedScreenWidth(screen types.Screen, xDiff int32) {
	if xDiff == 0 {
		return
	}

	newWidth := term.size.X + xDiff // this is correct: + & - == -
	if newWidth < 1 {
		newWidth = 1
	}

	term._reflowScreenWidth(screen, newWidth)
}

func (term *Term) _reflowScreenWidth(screen types.Screen, newWidth int32) {
	if len(screen) == 0 {
		return
	}

	type line struct {
		cells  []*types.Cell
		meta   types.RowMetaFlag
		source *types.RowSource
		block  *types.BlockMeta
	}

	type rowState struct {
		cells  []*types.Cell
		meta   types.RowMetaFlag
		source *types.RowSource
		block  *types.BlockMeta
	}

	trimTrailingSpaces := func(cells []*types.Cell) []*types.Cell {
		for i := len(cells) - 1; i >= 0; i-- {
			if cells[i].Rune() != ' ' {
				return cells[:i+1]
			}
		}

		return cells[:0]
	}

	lines := make([]line, 0, len(screen))
	for i := 0; i < len(screen); {
		start := i
		for i+1 < len(screen) && screen[i+1].RowMeta.Is(types.META_ROW_FROM_LINE_OVERFLOW) {
			i++
		}
		end := i

		cells := make([]*types.Cell, 0, (end-start+1)*int(term.size.X))
		for y := start; y <= end; y++ {
			cells = append(cells, screen[y].Cells...)
		}
		cells = trimTrailingSpaces(cells)

		meta := screen[start].RowMeta
		meta.Unset(types.META_ROW_FROM_LINE_OVERFLOW)

		lines = append(lines, line{
			cells:  cells,
			meta:   meta,
			source: screen[start].Source,
			block:  screen[start].Block,
		})

		i++
	}

	states := make([]rowState, 0, len(screen))
	for i := range lines {
		if len(lines[i].cells) == 0 {
			states = append(states, rowState{
				cells:  term.makeCells(newWidth),
				meta:   lines[i].meta,
				source: lines[i].source,
				block:  lines[i].block,
			})
			continue
		}

		for offset := 0; offset < len(lines[i].cells); offset += int(newWidth) {
			end := offset + int(newWidth)
			if end > len(lines[i].cells) {
				end = len(lines[i].cells)
			}

			cells := make([]*types.Cell, 0, newWidth)
			cells = append(cells, lines[i].cells[offset:end]...)
			if int32(len(cells)) < newWidth {
				cells = append(cells, term.makeCells(newWidth-int32(len(cells)))...)
			}

			meta := lines[i].meta
			if offset > 0 {
				meta.Set(types.META_ROW_FROM_LINE_OVERFLOW)
			}

			states = append(states, rowState{
				cells:  cells,
				meta:   meta,
				source: lines[i].source,
				block:  lines[i].block,
			})
		}
	}

	if len(states) > len(screen) {
		states = states[:len(screen)]
	} else if len(states) < len(screen) {
		pad := make([]rowState, len(screen)-len(states))
		for i := range pad {
			srcRow := screen[len(states)+i]
			pad[i].cells = term.makeCells(newWidth)
			pad[i].meta = srcRow.RowMeta
			pad[i].source = srcRow.Source
			pad[i].block = srcRow.Block
		}
		states = append(states, pad...)
	}

	for y := range screen {
		screen[y].Cells = states[y].cells
		screen[y].RowMeta = states[y].meta
		screen[y].Source = states[y].source
		screen[y].Block = states[y].block

		term._reflowScreenWidth(screen[y].Hidden, newWidth)
	}
}

func (term *Term) resizePty() {
	if term.Pty == nil {
		debug.Log("cannot resize pt: term.Pty == nil")
		return
	}

	err := term.Pty.Resize(term.size)
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
}

func (term *Term) resize80() {
	term.setSize(&types.XY{X: 80, Y: 24})
}

func (term *Term) resize132() {
	term.setSize(&types.XY{X: 132, Y: 24})
}

func (term *Term) setSize(size *types.XY) {
	term.reset(size)
	if !config.Config.Tmux.Enabled {
		term.renderer.ResizeWindow(size)
	}
}
