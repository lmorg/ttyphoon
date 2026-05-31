package virtualterm

import (
	"log"

	"github.com/lmorg/ttyphoon/codes"
)

/*
	Reference documentation used:
	- https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
	- ChatGPT (when the documentation above was unclear)
*/

func (term *Term) parsePmCodes() {
	var (
		r    rune
		err  error
		text []rune
	)

	for {
		r, err = term.Pty.Read()
		if err != nil {
			return
		}

		text = append(text, r)
		if r == codes.AsciiEscape {
			r, err = term.Pty.Read()
			if err != nil {
				return
			}
			if r == '\\' { // ST (PM terminator)
				break
			}
			text = append(text, r)
			continue
		}
	}

	log.Printf("DEBUG: Ignored PM code %s", string(text[:len(text)-2]))
}
