//go:build ignore
// +build ignore

package virtualterm

const _DEBUG_CHAR = '·'

var _DEBUG_SGR = &sgr{
	fg: SGR_COLOUR_BLACK_BRIGHT,
	bg: SGR_DEFAULT.bg,
}

func _debug_Cell() cell {
	return cell{
		char: _DEBUG_CHAR,
		sgr:  _DEBUG_SGR,
	}
}

func (term *Term) _debug_FillRowWithDots() []cell {
	row := make([]cell, term.size.X)
	for i := range row {
		row[i] = _debug_Cell()
	}

	return row
}
