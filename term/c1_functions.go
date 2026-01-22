package virtualterm

import "github.com/lmorg/ttyphoon/types"

func (term *Term) c1DecalnTestAlignment() {
	term._curPos = types.XY{} // top left
	for i := int32(0); i < term.size.X*term.size.Y; i++ {
		term.writeCell('E', nil)
	}
	term._curPos = types.XY{} // top left
}
