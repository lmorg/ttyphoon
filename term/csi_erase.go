package virtualterm

import (
	"fmt"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
)

/*
	ERASE DISPLAY
*/

func (term *Term) csiEraseDisplayAfter() {
	debug.Log(term.curPos())

	_, bottom := term.getScrollingRegionExcOrigin()

	curPosY := term.curPos().Y
	if curPosY > bottom {
		debug.Log(fmt.Sprintf("curPos().Y[%d]>bottom[%d]", curPosY, bottom))
		return
	}
	

	for y := curPosY; /*+ 1*/ y <= bottom; y++ {
		term.deallocateRows((*term.screen)[y])
		(*term.screen)[y] = term.makeRow()
	}
	term.csiEraseLineAfter()
}

func (term *Term) csiEraseDisplayBefore() {
	debug.Log(term.curPos())

	top, _ := term.getScrollingRegionExcOrigin()
	for y := term.curPos().Y - 1; y >= top; y-- {
		term.deallocateRows((*term.screen)[y])
		(*term.screen)[y] = term.makeRow()
	}
	term.csiEraseLineBefore()
}

func (term *Term) csiEraseDisplay() {
	debug.Log(term.curPos())

	y, bottom := term.getScrollingRegionExcOrigin()
	for ; y <= bottom; y++ {
		term.deallocateRows((*term.screen)[y])
		(*term.screen)[y] = term.makeRow()
	}
}

func (term *Term) eraseScrollBack() {
	debug.Log(term.curPos())

	term._scrollOffset = 0

	term.deallocateRows(term._scrollBuf...)
	term._scrollBuf = types.Screen{}
}

/*
	ERASE LINE
*/

func (term *Term) csiEraseLineAfter() {
	debug.Log(term.curPos())

	pos := term.curPos()
	n := term.size.X - pos.X

	term.deallocateCells((*term.screen)[pos.Y].Cells[pos.X:])

	clear := term.makeCells(n)
	copy((*term.screen)[pos.Y].Cells[pos.X:], clear)
}

func (term *Term) csiEraseLineBefore() {
	debug.Log(term.curPos())

	pos := term.curPos()
	n := pos.X + 1

	term.deallocateCells((*term.screen)[pos.Y].Cells)

	clear := term.makeCells(n)
	copy((*term.screen)[pos.Y].Cells, clear)
}

func (term *Term) csiEraseLine() {
	debug.Log(term.curPos())

	term.deallocateRows((*term.screen)[term.curPos().Y])
	(*term.screen)[term.curPos().Y] = term.makeRow()
}

/*
	ERASE CHARACTERS
*/

func (term *Term) csiEraseCharacters(n int32) {
	debug.Log(n)

	if n < 1 {
		n = 1
	}

	pos := term.curPos()
	clear := term.makeCells(n)

	term.deallocateCells((*term.screen)[pos.Y].Cells[pos.X:])
	copy((*term.screen)[pos.Y].Cells[pos.X:], clear)
}

/*
	DELETE
*/

func (term *Term) csiDeleteCharacters(n int32) {
	debug.Log(n)

	if n < 1 {
		n = 1
	}

	pos := term.curPos()

	if n+pos.X > term.size.X {
		n = term.size.X - pos.X
	}

	if n < 1 {
		return
	}

	term.deallocateCells((*term.screen)[pos.Y].Cells[pos.X:])
	copy((*term.screen)[pos.Y].Cells[pos.X:], (*term.screen)[pos.Y].Cells[pos.X+n:])
	blank := term.makeCells(n)
	copy((*term.screen)[pos.Y].Cells[term.size.X-n:], blank)
}

// TODO: test
func (term *Term) csiDeleteLines(n int32) {
	debug.Log(n)

	if n < 1 {
		n = 1
	}

	_, bottom := term.getScrollingRegionExcOrigin()

	term._scrollUp(term.curPos().Y, bottom, n)
}
