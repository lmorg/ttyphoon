package virtualterm

import (
	"github.com/lmorg/ttyphoon/types"
)

func (term *Term) ShowCursor(v bool) {
	term._hideCursor = !v
}

func (term *Term) _renderCursor() {
	if term._hideCursor || term._scrollOffset != 0 {
		return
	}

	var w int32 = 1
	sgr := term.currentCell().Sgr
	if sgr != nil && sgr.Bitwise.Is(types.SGR_WIDE_CHAR) {
		w = 2
	}

	// Focused cursor: draw twice (marker for animated cursor in WebKit).
	if term._isFocused {
		term.renderer.DrawRectWithColourAndBorder(term.tile, term.curPos(), &types.XY{X: w, Y: 1}, types.SGR_COLOR_FOREGROUND, false, true)
		term.renderer.DrawRectWithColourAndBorder(term.tile, term.curPos(), &types.XY{X: w, Y: 1}, types.SGR_COLOR_FOREGROUND, false, true)
		return
	}

	// Inactive cursor: single draw (marker for static hollow cursor in WebKit).
	term.renderer.DrawRectWithColourAndBorder(term.tile, term.curPos(), &types.XY{X: w, Y: 1}, types.SGR_COLOR_FOREGROUND, false, true)
}
