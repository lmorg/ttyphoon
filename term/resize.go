package virtualterm

import (
	"strings"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"golang.org/x/sys/unix"
)

func (term *Term) Resize(size *types.XY) {
	xDiff := int32(size.X - term.size.X)
	yDiff := int(size.Y - term.size.Y)

	term._mutex.Lock()

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
		for i := 0; i < yDiff; i++ {
			term._normBuf = append(term._normBuf, term.makeRow())
		}
		for i := 0; i < yDiff; i++ {
			term._altBuf = append(term._altBuf, term.makeRow())
		}

	case yDiff < 0:
		// shrink
		fromBottom := term._resizeFromBottom(-yDiff)
		//debug.Log(yDiff)
		//debug.Log(fromBottom)
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

	term._mutex.Unlock()
}

func (term *Term) _resizeFromBottom(max int) int {
	if len(term._scrollBuf) > 0 || term.IsAltBuf() {
		return 0
	}

	empty := strings.Repeat(" ", int(term.size.X))
	var i int
	for y := len(term._normBuf) - 1; i < max; y-- {
		if term._normBuf[y].String() != empty {
			//debug.Log(i)
			return i
		}
		i++
	}

	return i
}

func (term *Term) _resizeNestedScreenWidth(screen types.Screen, xDiff int32) {
	switch {
	case xDiff == 0:
		// nothing to do

	case xDiff > 0:
		// grow

		for y := range screen {
			screen[y].Cells = append(screen[y].Cells, term.makeCells(xDiff)...)
			term._resizeNestedScreenWidth(screen[y].Hidden, xDiff)
		}

	case xDiff < 0:
		// crop (this is lazy, really we should reflow)

		for y := range screen {
			screen[y].Cells = screen[y].Cells[:term.size.X+xDiff] // this is correct: + & - == -
			term._resizeNestedScreenWidth(screen[y].Hidden, xDiff)
		}
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

	if term.process != nil {
		err = term.process.Signal(unix.SIGWINCH)
		if err != nil {
			debug.Log(err)
		}
	}
}

func (term *Term) resize80() {
	term.setSize(&types.XY{X: 80, Y: 24})
}

func (term *Term) resize132() {
	term.setSize(&types.XY{X: 132, Y: 24})
}

func (term *Term) setSize(size *types.XY) {
	if !config.Config.Tmux.Enabled {
		term.reset(size)
		term.renderer.ResizeWindow(size)
	}
}
