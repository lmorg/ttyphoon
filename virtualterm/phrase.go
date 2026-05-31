package virtualterm

import (
	"github.com/lmorg/ttyphoon/types"
)

func (term *Term) phraseSetToRowPos(flags linefeedF) {
	/*if term.IsAltBuf() {
		return
	}*/

	if flags.Is(_LINEFEED_LINE_OVERFLOWED) {
		(*term.screen)[term.curPos().Y].RowMeta.Set(types.META_ROW_FROM_LINE_OVERFLOW)
	} else {
		(*term.screen)[term.curPos().Y].RowMeta.Unset(types.META_ROW_FROM_LINE_OVERFLOW)
	}

	(*term.screen)[term.curPos().Y].Source = term._rowSource
	(*term.screen)[term.curPos().Y].Block = term._blockMeta
}
