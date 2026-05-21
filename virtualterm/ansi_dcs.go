package virtualterm

import (
	"log"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/types"
)

/*
	Reference documentation used:
	- xterm: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Device-Control-functions
*/

func (term *Term) parseDcsCodes() {
	var (
		r    rune
		err  error
		text []rune
	)

	r, err = term.Pty.Read()
	if err != nil {
		return
	}

	switch r {
	case 'q':
		escSeq := term._parseDcsCodes()
		escSeq = append([]rune{codes.AsciiEscape, 'P', 'q'}, escSeq...)
		escSeq = append(escSeq, codes.AsciiEscape, '\\')
		apc := types.NewApcSliceNoParse([]string{string(escSeq)})
		term.mxapcInsert(types.ELEMENT_ID_SIXEL, apc)

	default:
		log.Printf("WARNING: Unhandled DCS code %s", string(text))
	}

}

func (term *Term) _parseDcsCodes() []rune {
	var (
		r      rune
		err    error
		escSeq []rune
	)

	for {
		r, err = term.Pty.Read()
		if err != nil {
			return escSeq
		}

		switch r {
		case codes.AsciiEscape:
			r, err = term.Pty.Read()
			if err != nil {
				return escSeq
			}
			escSeq = append(escSeq, r)
			if r == '\\' { // ST (DCS terminator)
				return escSeq
			}
			continue

		case ' ', '\r', '\n':
			continue

		default:
			escSeq = append(escSeq, r)
		}
	}
}
