package virtualterm

import (
	"github.com/lmorg/ttyphoon/types"
)

func (term *Term) convertRelPosToAbsPos(pos *types.XY) *types.XY {
	return &types.XY{
		X: pos.X,
		Y: int32(len(term._scrollBuf)) - int32(term._scrollOffset) + pos.Y,
	}
}
