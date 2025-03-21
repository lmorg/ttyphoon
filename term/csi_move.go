package virtualterm

import (
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
)

// basic TTY operations

func (term *Term) carriageReturn() {
	term._curPos.X = 0
}

func (term *Term) lineFeed() {
	//debug.Log(term.curPos.Y)
	//term.phraseIsClickable((*term.screen)[term._curPos.Y])

	if term.csiMoveCursorDownwardsExcOrigin(1) != 0 {
		term.appendScrollBuf(1)
		term.csiScrollUp(1)
		term.csiMoveCursorDownwardsExcOrigin(1)
	}

	term.phraseSetToRowPos()
	go term.lfRedraw()
}

func (term *Term) setJumpScroll() {
	i := int32(config.Config.Terminal.JumpScrollLineCount)
	if i < 0 {
		i = term.size.Y
	}
	term._ssFrequency = i
}

func (term *Term) setSmoothScroll() {
	term._ssFrequency = 0
}

func (term *Term) reverseLineFeed() {
	debug.Log(term._curPos.Y)

	if term.csiMoveCursorUpwardsExcOrigin(1) != 0 {
		term.csiScrollDown(1)
		term.csiMoveCursorUpwardsExcOrigin(1)
	}

	term.phraseSetToRowPos()
}

/*
	csiMoveCursor[...] functions DOESN'T affect other contents in the grid
*/

// csiMoveCursorBackwards: 0 should default to 1.
// Returns how many additional cells were requested before hitting the edge of
// the screen.
func (term *Term) csiMoveCursorBackwards(i int32) (overflow int32) {
	debug.Log(i)

	if i < 1 {
		i = 1
	}

	pos := term.curPos()
	term._curPos.X = pos.X - i
	if term._curPos.X < 0 {
		overflow = -term._curPos.X
		term._curPos.X = 0
	}

	return
}

// csiMoveCursorForwards: 0 should default to 1.
// Returns how many additional cells were requested before hitting the edge of
// the screen.
func (term *Term) csiMoveCursorForwards(i int32) (overflow int32) {
	debug.Log(i)

	if i < 1 {
		i = 1
	}

	pos := term.curPos()
	term._curPos.X = pos.X + i
	if term._curPos.X >= term.size.X {
		overflow = term._curPos.X - (term.size.X - 1)
		term._curPos.X = term.size.X - 1
	}

	return
}

// csiMoveCursorUpwards: 0 should default to 1.
// Returns how many additional cells were requested before hitting the edge of
// the screen.
func (term *Term) csiMoveCursorUpwards(i int32) (overflow int32) {
	debug.Log(i)

	if i < 1 {
		i = 1
	}

	pos := term.curPos()
	top, _ := term.getScrollingRegionIncOrigin()

	term._curPos.Y = pos.Y - i
	if term._curPos.Y < top {
		overflow = term._curPos.Y - top
		term._curPos.Y = top
	}

	term.phraseSetToRowPos()

	return
}

func (term *Term) csiMoveCursorUpwardsExcOrigin(i int32) (overflow int32) {
	debug.Log(i)

	if i < 1 {
		i = 1
	}

	pos := term.curPos()
	top, _ := term.getScrollingRegionExcOrigin()

	term._curPos.Y = pos.Y - i
	if term._curPos.Y < top {
		overflow = term._curPos.Y - top
		term._curPos.Y = top
	}

	term.phraseSetToRowPos()

	return
}

// csiMoveCursorDownwards: 0 should default to 1.
// Returns how many additional cells were requested before hitting the edge of
// the screen.
func (term *Term) csiMoveCursorDownwards(i int32) (overflow int32) {
	debug.Log(i)

	if i < 1 {
		i = 1
	}

	pos := term.curPos()
	term._curPos.Y = pos.Y + i

	_, bottom := term.getScrollingRegionIncOrigin()

	if term._curPos.Y > bottom {
		overflow = term._curPos.Y - bottom
		term._curPos.Y = bottom
	}

	term.phraseSetToRowPos()

	return
}

func (term *Term) csiMoveCursorDownwardsExcOrigin(i int32) (overflow int32) {
	debug.Log(i)

	if i < 1 {
		i = 1
	}

	pos := term.curPos()
	term._curPos.Y = pos.Y + i

	_, bottom := term.getScrollingRegionExcOrigin()

	if term._curPos.Y > bottom {
		overflow = term._curPos.Y - bottom
		term._curPos.Y = bottom
	}

	term.phraseSetToRowPos()

	return
}

func (term *Term) moveCursorToColumn(col int32) {
	debug.Log(col)

	switch {
	case col < 1:
		term._curPos.X = 0
	case col > term.size.X:
		term._curPos.X = term.size.X - 1
	default:
		term._curPos.X = col - 1
	}
}

func (term *Term) moveCursorToRow(row int32) {
	debug.Log(row)

	top, bottom := term.getScrollingRegionIncOrigin()

	row += top

	switch {
	case row < top:
		term._curPos.Y = top
	case row > bottom:
		term._curPos.Y = bottom
	default:
		term._curPos.Y = row - 1
	}

	term.phraseSetToRowPos()
}

// csiMoveCursorToPos: 0 values should default to current cursor position.
func (term *Term) moveCursorToPos(row, col int32) {
	debug.Log([]int32{row, col})
	term.moveCursorToRow(row)
	term.moveCursorToColumn(col)

	//term.phraseSetToRowPos() // already included in `moveCursorToRow()`
}

/*
	SCROLLING
*/

// csiSetScrollingRegion values should be offset by 1 (as seen in the ANSI
// escape codes). eg the top left corder would be `[]int32{1, 1}`.
func (term *Term) setScrollingRegion(region []int32) {
	debug.Log(region)

	switch {
	case region[0] < 0:
		region[0] = 0
	case region[0] > term.size.Y:
		region[0] = term.size.Y - 1
	}

	switch {
	case region[1] < region[0]:
		region[1] = region[0]
	case region[1] > term.size.Y:
		region[1] = term.size.Y
	}

	term._scrollRegion = &scrollRegionT{
		Top:    region[0] - 1,
		Bottom: region[1] - 1,
	}

	pos := term.curPos()
	if pos.Y <= region[0] {
		term._curPos.Y = term._scrollRegion.Top
	}
	if pos.Y >= region[1] {
		term._curPos.Y = term._scrollRegion.Bottom
	}
}

func (term *Term) unsetScrollingRegion() {
	debug.Log(nil)
	term._scrollRegion = nil
}

func (term *Term) getScrollingRegionIncOrigin() (top int32, bottom int32) {
	debug.Log(term._scrollRegion)

	if term._scrollRegion == nil || !term._originMode {
		top = 0
		bottom = term.size.Y - 1
	} else {
		top = term._scrollRegion.Top
		bottom = term._scrollRegion.Bottom
	}

	return
}

func (term *Term) getScrollingRegionExcOrigin() (top int32, bottom int32) {
	debug.Log(term._scrollRegion)

	if term._scrollRegion == nil {
		top = 0
		bottom = term.size.Y - 1
	} else {
		top = term._scrollRegion.Top
		bottom = term._scrollRegion.Bottom
	}

	// this shouldn't be needed, but it at least helps with some rogue panics
	if int(bottom) > len(*term.screen) {
		bottom = int32(len(*term.screen) - 1)
	}

	return
}

// csiScrollUp: 0 should default to 1.
func (term *Term) csiScrollUp(n int32) {
	debug.Log(n)

	if n < 1 {
		n = 1
	}

	top, bottom := term.getScrollingRegionExcOrigin()

	term._scrollUp(top, bottom, n)
}

// _scrollDown does not take into account term size nor scrolling region. Any
// error handling should be done by the calling function.
func (term *Term) _scrollUp(top, bottom, shift int32) {
	for i := top; i <= bottom; i++ {
		if i+shift <= bottom {
			(*term.screen)[i] = (*term.screen)[i+shift]
		} else {
			(*term.screen)[i] = term.makeRow()
		}
	}
}

// csiScrollDown: 0 should default to 1.
func (term *Term) csiScrollDown(n int32) {
	debug.Log(n)

	if n < 1 {
		n = 1
	}

	top, bottom := term.getScrollingRegionExcOrigin()

	term._scrollDown(top, bottom, n)
}

// _scrollDown does not take into account term size nor scrolling region. Any
// error handling should be done by the calling function.
func (term *Term) _scrollDown(top, bottom, shift int32) {
	debug.Log(map[string]int32{"top": top, "bottom": bottom, "shift": shift})

	screen := term.makeScreen()

	if shift > bottom-top {
		shift = bottom - top
	}

	copy(screen[top+shift:], (*term.screen)[top:bottom+1])
	copy((*term.screen)[top:], screen[top:bottom+1])
}

/*
	INSERT
*/

// csiInsertCharacters: 0 should default to 1.
func (term *Term) csiInsertCharacters(n int32) {
	debug.Log(n)

	if n < 1 {
		n = 1
	}

	insert := term.makeCells(n)
	cells := make([]*types.Cell, term.size.X)

	pos := term.curPos()
	copy(cells, (*term.screen)[pos.Y].Cells[:pos.X])
	copy(cells[pos.X:], insert)
	copy(cells[pos.X+n:], (*term.screen)[pos.Y].Cells[pos.X:])

	(*term.screen)[pos.Y].Cells = cells
}

// csiInsertLines: 0 should default to 1.
func (term *Term) csiInsertLines(n int32) {
	debug.Log(n)
	// TODO: test
	if n < 1 {
		n = 1
	}

	_, bottom := term.getScrollingRegionExcOrigin()

	top := term.curPos().Y
	if top > bottom {
		top = bottom
	}
	term._scrollDown(top, bottom, n)
}
