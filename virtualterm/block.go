package virtualterm

import (
	"github.com/lmorg/ttyphoon/types"
)

func (term *Term) getBlockStartAndEndAbs(absPos int) [2]int {
	screen := append(term._scrollBuf, term._normBuf...)
	var begin, end int

	for begin = absPos; begin >= 0; begin-- {
		if screen[begin].RowMeta.Is(types.META_ROW_BEGIN_BLOCK) {
			break
		}
	}

	for end = absPos; end < len(screen); end++ {
		if screen[end].RowMeta.Is(types.META_ROW_END_BLOCK) {
			break
		}
	}

	return [2]int{begin, end}
}

func (term *Term) getBlockStartAndEndRel(absBlockPos [2]int) [2]int32 {
	return [2]int32{
		int32(absBlockPos[0] - len(term._scrollBuf) + term._scrollOffset),
		int32(absBlockPos[1] - absBlockPos[0] + 1),
	}
}

func _outputBlockChromeColour(meta types.BlockMetaFlag) *types.Colour {
	switch {
	case meta.Is(types.META_BLOCK_AI):
		return types.COLOR_AI
	case meta.Is(types.META_BLOCK_OK):
		return types.COLOR_OK
	case meta.Is(types.META_BLOCK_ERROR):
		return types.COLOR_ERROR
	default:
		return types.COLOR_FOLDED
	}
}
