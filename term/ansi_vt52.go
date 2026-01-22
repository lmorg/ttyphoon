package virtualterm

import (
	"log"

	"github.com/lmorg/ttyphoon/codes"
)

/*
	Reference documentation used:
	- xterm: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-VT52-Mode

	Parameters for cursor movement are at the end of the ESC Y  escape
	sequence.  Each ordinate is encoded in a single character as value+32.
	For example, !  is 1.  The screen coordinate system is 0-based.

*/

func (term *Term) parseVt52Codes() {
	r, err := term.Pty.Read()
	if err != nil {
		return
	}

	switch r {
	case '<':
		// Exit VT52 mode (Enter VT100 mode).
		term._vtMode = _VT100

	case '=':
		// Enter alternate keypad mode.
		log.Printf("TODO: VT52 code not implemented: %s", string(r))

	case '>':
		// Exit alternate keypad mode.
		log.Printf("TODO: VT52 code not implemented: %s", string(r))

	case 'A':
		// Cursor up.
		term.csiMoveCursorUpwards(1)

	case 'B':
		// Cursor down.
		term.csiMoveCursorDownwards(1)

	case 'C':
		// Cursor right.
		term.csiMoveCursorForwards(1)

	case 'D':
		// Cursor left.
		term.csiMoveCursorBackwards(1)

	case 'F':
		// Enter graphics mode.
		log.Printf("TODO: VT52 code not implemented: %s", string(r))

	case 'G':
		// Exit graphics mode.
		log.Printf("TODO: VT52 code not implemented: %s", string(r))

	case 'H':
		// Move the cursor to the home position.
		term.moveCursorToPos(1, 1)

	case 'I':
		// Reverse line feed.
		term.reverseLineFeed()

	case 'J':
		// Erase from the cursor to the end of the screen.
		term.csiEraseDisplayAfter()

	case 'K':
		// Erase from the cursor to the end of the line.
		term.csiEraseLineAfter()

	case 'Y':
		// ESC Y Ps Ps
		// Move the cursor to given row and column.
		row, err := term.Pty.Read()
		if err != nil {
			return
		}

		col, err := term.Pty.Read()
		if err != nil {
			return
		}

		term.moveCursorToPos(col-32, row-32)

	case 'Z':
		// Identify.
		// ⇒  ESC  /  Z  ("I am a VT52.").
		term.Reply([]byte{codes.AsciiEscape, '/', 'Z'})

	default:
		log.Printf("WARNING: VT52 code not recognized: %s", string(r))
	}
}
