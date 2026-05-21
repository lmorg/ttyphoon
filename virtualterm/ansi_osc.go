package virtualterm

import (
	"log"
	"strings"

	"github.com/lmorg/ttyphoon/codes"
)

/*
	Reference documentation used:
	- Wikipedia: https://en.wikipedia.org/wiki/ANSI_escape_code#OSC_(Operating_System_Command)_sequences
	- xterm: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands
*/

const (
	_TERMINATE_ST   = "\x1b\\"
	_TERMINATE_BELL = "\a"
)

func (term *Term) parseOscCodes() {
	var (
		r          rune
		err        error
		text       []rune
		terminator string
	)

	for {
		r, err = term.Pty.Read()
		if err != nil {
			return
		}
		text = append(text, r)
		switch r {

		case codes.AsciiEscape:
			r, err = term.Pty.Read()
			if err != nil {
				return
			}
			if r == '\\' { // ST (OSC terminator)
				terminator = _TERMINATE_ST
				goto parsed
			}
			text = append(text, r)
			continue

		case codes.AsciiCtrlG: // bell (xterm OSC terminator)
			terminator = _TERMINATE_BELL
			goto parsed

		}

	}

parsed:
	text = text[:len(text)-1]

	stack := strings.Split(string(text), ";")

	switch stack[0] {
	case "0":
		// Change icon and window title
		term.renderer.SetWindowTitle(stack[1])

	case "2":
		// Change window title
		term.renderer.SetWindowTitle(stack[1])

	case "4":
		// Change Color Number c to the color specified by spec.
		term.osc4ColorNumber(stack[1:], terminator)

	case "7":
		// Update path
		term.osc7UpdatePath(stack[1:])

	case "9":
		// Post a notification
		term.osc9PostNotification(stack[1:])

	case "10":
		// Change VT100 text foreground color to Pt.
		term.osc1xColorFgBG(10, stack[1:], terminator)

	case "11":
		// Change VT100 text background color to Pt.
		term.osc1xColorFgBG(11, stack[1:], terminator)

	case "1337":
		// iTerm2 proprietary escape codes
		term.osc1337iTerm2(stack[1:])

	default:
		log.Printf("WARNING: Unknown OSC code %s: %s", stack[0], string(text[:len(text)-1]))
	}
}
