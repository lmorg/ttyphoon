package virtualterm

import (
	"fmt"
	"log"

	"github.com/lmorg/mxtty/types"
)

/*
	Reference documentation used:
	- xterm: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-C1-lparen-8-Bit-rparen-Control-Characters
	- Wikipedia: https://en.wikipedia.org/wiki/C0_and_C1_control_codes
	- ChatGPT (when the documentation above was unclear)
*/

func (term *Term) parseC1Codes() {
	r, err := term.Pty.Read()
	if err != nil {
		return
	}

	switch r {
	case '[':
		// CSI code
		term.parseCsiCodes()

	case ']':
		// OSC code
		term.parseOscCodes()

	case 'P':
		// DCS code
		term.parseDcsCodes()

	case '^':
		// PM code
		term.parsePmCodes()

	case '_':
		// APC code
		term.parseApcCodes()

	case '#':
		// DEC codes
		r, err := term.Pty.Read()
		if err != nil {
			return
		}

		switch r {
		case '5':
			// DEC single-width line (DECSWL), VT100
			// While this technically isn't an SGR code, we'll treat it like it is one
			term.sgr.Bitwise.Unset(types.SGR_WIDE_CHAR) // TODO: this doesn't work
		case '6':
			// DEC double-width line (DECDWL), VT100
			// While this technically isn't an SGR code, we'll treat it like it is one
			term.sgr.Bitwise.Set(types.SGR_WIDE_CHAR) // TODO: this doesn't work
		case '8':
			// DEC Screen Alignment Test (DECALN)
			term.c1DecalnTestAlignment()

		default:
			log.Printf("TODO: Unhandled DEC C1 escape sequence: {ESC}#%s", string(r))
		}

	case ' ':
		// 7/8bit controls
		// ANSI conformance level
		param, _ := term.Pty.Read()
		log.Printf("DEBUG: Ignored '{ESC}%%%s' sequence", string(param))

	case '%':
		// @: Select default character set.  That is ISO 8859-1 (ISO 2022).
		// G: Select UTF-8 character set, ISO 2022.
		param, _ := term.Pty.Read() // Ignore these sequences. We always default to UTF-8
		log.Printf("DEBUG: Ignored '{ESC}%%%s' sequence, we always default to UTF-8", string(param))

	case '(':
		// Designate G0 Character Set (ISO 2022), VT100.
		term._charSetG[0], err = term.fetchCharacterSet()

	case ')':
		// Designate G1 Character Set (ISO 2022), VT100.
		term._charSetG[1], err = term.fetchCharacterSet()

	case '*':
		// Designate G2 Character Set (ISO 2022), VT220.
		term._charSetG[2], err = term.fetchCharacterSet()

	case '+':
		// Designate G3 Character Set (ISO 2022), VT220.
		term._charSetG[3], err = term.fetchCharacterSet()

	case '-':
		// Designate G1 Character Set, VT300.
		term._charSetG[1], err = term.fetchCharacterSet()

	case '.':
		// Designate G2 Character Set, VT300.
		term._charSetG[2], err = term.fetchCharacterSet()

	case '/':
		// Designate G3 Character Set, VT300.
		term._charSetG[3], err = term.fetchCharacterSet()

	case '=':
		// Application Keypad (DECPAM)
		// TODO: this probably shouldn't redefine every function key
		term.renderer.SetKeyboardFnMode(types.KeysApplication)

	case '>':
		// Normal Keypad (DECPNM)
		// TODO: this probably shouldn't redefine every function key
		term.renderer.SetKeyboardFnMode(types.KeysNormal)

	case 'c':
		// Full Reset (RIS)
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'k':
		// Tmux: allow-rename [on | off]
		// Allow programs in the pane to change the window name
		// using a terminal escape sequence (\ek...\e\\).
		term.tmuxRenameWindow()

	case 'l':
		// Memory Lock (per HP terminals). Locks memory above the cursor.
		//log.Printf("TODO: Unhandled C1 code: %s", string(r))
		term.renderer.DisplayNotification(types.NOTIFY_WARN, "Unsupported C0 code: Memory Lock (per HP terminals). Locks memory above the cursor")

	case 'm':
		// Memory Unlock (per HP terminals)
		//log.Printf("TODO: Unhandled C1 code: %s", string(r))
		term.renderer.DisplayNotification(types.NOTIFY_WARN, "Unsupported C0 code: Memory Unlock (per HP terminals)")

	case 'n':
		// Invoke the G2 Character Set as GL (LS2).
		term._activeCharSet = 2

	case 'o':
		// Invoke the G3 Character Set as GL (LS3).
		term._activeCharSet = 3

	case '|':
		// Invoke the G3 Character Set as GR (LS3R).
		term._activeCharSet = 3

	case '}':
		// Invoke the G2 Character Set as GR (LS2R).
		term._activeCharSet = 2

	case '~':
		// Invoke the G1 Character Set as GR (LS1R), VT100.
		term._activeCharSet = 1

	case '7':
		term.csiCursorPosSave()

	case '8':
		term.csiCursorPosRestore()

	case '@':
		// Padding Character
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'A':
		// High Octet Preset
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'B':
		// Break Permitted Here
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'C':
		// No Break Here
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'D':
		// Index (IND  is 0x84).
		term.lineFeed()

	case 'E':
		// Next Line (NEL  is 0x85).
		term.carriageReturn()
		term.lineFeed()

	case 'F', 'G':
		// Start of Selected Area
		// Start of Selected Area
		// Cursor to lower left corner of screen (if enabled by the 'hpLowerleftBugCompat' resource).
		//log.Printf("TODO: Unhandled C1 code: %s", string(r))
		term.renderer.DisplayNotification(types.NOTIFY_WARN, "Unsupported C0 code: Select Area (Cursor to lower left corner of screen - if enabled by the 'hpLowerleftBugCompat' resource")

	case 'H':
		// Tab Set (HTS  is 0x88).
		// Character Tabulation Set
		// Horizontal Tabulation Set
		term.c1AddTabStop()

	case 'I':
		// Character Tabulation With Justification
		// Horizontal Tabulation With Justification
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'J':
		// Line Tabulation Set
		// Vertical Tabulation Set
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'K':
		// Partial Line Forward
		// Partial Line Down
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'L':
		// Partial Line Backward
		// Partial Line Up
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'M':
		// Reverse Index (RI  is 0x8d).
		// Reverse Line Feed
		// Reverse Index
		term.reverseLineFeed()

	case 'N':
		// Single Shift Select of G2 Character Set (SS2  is 0x8e), VT220.
		// This affects next character only.
		charSet := term._activeCharSet
		term._activeCharSet = 2
		r, err := term.Pty.Read()
		if err != nil {
			return
		}
		term.readChar(r)
		term._activeCharSet = charSet

	case 'O':
		// Single Shift Select of G3 Character Set (SS3  is 0x8f), VT220.
		// This affects next character only.
		charSet := term._activeCharSet
		term._activeCharSet = 3
		r, err := term.Pty.Read()
		if err != nil {
			return
		}
		term.readChar(r)
		term._activeCharSet = charSet

	case 'Q':
		// Private Use 1
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'R':
		// Private Use 2
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'S':
		// Set Transmit State
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'T':
		// Cancel character
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'U':
		// Message Waiting
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'V':
		// Start of Protected Area
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'W':
		// End of Protected Area
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'X':
		// Start of String
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'Y':
		// Single Graphic Character Introducer
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case 'Z':
		// Single Character Introducer
		log.Printf("TODO: Unhandled C1 code: %s", string(r))

	case '\\':
		// String Terminator
		log.Printf("DEBUG: unexpected string terminator")

	/////

	default:
		term.renderer.DisplayNotification(types.NOTIFY_DEBUG,
			fmt.Sprintf("Unexpected rune after escape: %d", r))

		//term.writeCell(r)
	}
}
