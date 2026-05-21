package virtualterm

import (
	"log"

	"github.com/lmorg/ttyphoon/debug"
)

/*
	Reference documentation used:
	- xterm: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Functions-using-CSI-_-ordered-by-the-final-character_s_
*/

func lookupTertiaryCsi(_ *Term, code []rune) {
	param := string(code[:len(code)-1])
	r := code[len(code)-1]

	debug.Log(param)

	switch r {
	case 'B':
		switch param {
		case "1":
			log.Printf("DEBUG: BEGIN 1")

		default:
			log.Printf("Tertiary CSI parameter not implemented in %s: %v [param: %s]", string(r), string(code), param)
		}

	case 'E':
		switch param {
		case "1":
			log.Printf("DEBUG: END 1")
		default:
			log.Printf("Tertiary CSI parameter not implemented in %s: %v [param: %s]", string(r), string(code), param)
		}

	default:
		log.Printf("Tertiary CSI code not implemented: %s (%s)", string(r), string(code))
	}
}
