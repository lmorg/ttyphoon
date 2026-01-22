package virtualterm

import (
	"fmt"

	"github.com/lmorg/ttyphoon/types"
)

func (term *Term) Match(pos *types.XY) {
	if term.IsAltBuf() {
		term.renderer.DisplayNotification(types.NOTIFY_WARN, "Match is not supported in alt buffer")
		return
	}

	r := (*term.screen)[pos.Y].Cells[pos.X].Rune()
	switch r {
	default:
		term.renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Match cannot run match against the character '%s'", string(r)))
		return
	}
}
