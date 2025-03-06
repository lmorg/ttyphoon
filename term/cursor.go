package virtualterm

import (
	"github.com/lmorg/mxtty/types"
)

func (term *Term) ShowCursor(v bool) {
	term._hideCursor = !v
}

func (term *Term) _renderCursor() {
	if term._hideCursor || term._scrollOffset != 0 {
		return
	}

	if !term._hasFocus || term.renderer.GetBlinkState() {
		var w int32 = 1
		sgr := term.currentCell().Sgr
		if sgr != nil && sgr.Bitwise.Is(types.SGR_WIDE_CHAR) {
			w = 2
		}

		// draw twice to make it _pop_
		term.renderer.DrawHighlightRect(term.tile, term.curPos(), &types.XY{X: w, Y: 1})
		term.renderer.DrawHighlightRect(term.tile, term.curPos(), &types.XY{X: w, Y: 1})
	}
}
