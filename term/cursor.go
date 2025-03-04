package virtualterm

import (
	"time"

	"github.com/lmorg/mxtty/types"
)

func (term *Term) slowBlink() {
	for {
		select {
		case <-time.After(500 * time.Millisecond):
			if !term._hasFocus {
				continue
			}
			term._slowBlinkState = !term._slowBlinkState
			term.renderer.TriggerRedraw()

		case <-term._eventClose:
			term.Pty.Close()
			return

		case <-term._hasKeyPress:
			term._slowBlinkState = true
		}
	}
}

func (term *Term) ShowCursor(v bool) {
	term._hideCursor = !v
}

func (term *Term) _renderCursor() {
	if term._hideCursor || term._scrollOffset != 0 {
		return
	}

	if term._slowBlinkState {
		var w int32 = 1
		sgr := term.currentCell().Sgr
		if sgr != nil && sgr.Bitwise.Is(types.SGR_WIDE_CHAR) {
			w = 2
		}

		term.renderer.DrawHighlightRect(term.tile, term.curPos(), &types.XY{w, 1})
		term.renderer.DrawHighlightRect(term.tile, term.curPos(), &types.XY{w, 1})
	}
}
