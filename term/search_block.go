package virtualterm

import (
	"github.com/lmorg/mxtty/types"
)

func (term *Term) convertRelPosToAbsPos(pos *types.XY) *types.XY {
	return &types.XY{
		X: pos.X,
		Y: int32(len(term._scrollBuf)) - int32(term._scrollOffset) + pos.Y,
	}
}

/*
func (term *Term) outputBlocksFindStartEnd(absolutePosition int32) (loc [2]int32, row [2]*types.Row, err error) {
	if term.IsAltBuf() {
		return loc, row, errors.New("this is not supported in alt buffer")
	}

	tmpBuf := append(term._scrollBuf, term._normBuf...)

	for i := absolutePosition; i >= 0; i-- {
		if tmpBuf[i].Meta.Is(types.ROW_OUTPUT_BLOCK_BEGIN) {
			loc[0] = i
			row[0] = tmpBuf[i]
			goto findEnd
		}
	}

	return loc, row, errors.New("cannot find start of output block")

findEnd:

	for i := absolutePosition; int(i) < len(tmpBuf); i++ {
		if tmpBuf[i].Meta.Is(types.ROW_OUTPUT_BLOCK_END) || tmpBuf[i].Meta.Is(types.ROW_OUTPUT_BLOCK_ERROR) {
			loc[1] = i
			row[1] = tmpBuf[i]
			goto fin
		}
	}

	return loc, row, errors.New("cannot find end of output block")

fin:

	return loc, row, nil

}

// TODO
func outputBlockFindMatchingRuneAfter(screen types.Screen, absPos *types.XY, r rune) {
	for i := absPos.Y; int(i) < len(screen); i++ {

	}
}
*/
