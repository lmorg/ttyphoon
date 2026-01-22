package virtualterm

import (
	"log"

	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

/*
	Reference documentation used:
	- xterm: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Functions-using-CSI-_-ordered-by-the-final-character_s_
*/

func lookupPrivateCsi(term *Term, code []rune) {
	param := string(code[:len(code)-1])
	r := code[len(code)-1]

	debug.Log(param)

	switch r {
	case 'h':
		// DEC Private Mode Set (DECSET).
		switch param {
		case "1":
			// Application Cursor Keys (DECCKM), VT100.
			term.renderer.SetKeyboardFnMode(types.KeysApplication)

		case "3":
			// 132 Column Mode (DECCOLM), VT100.
			term.resize132()

		case "4":
			// Smooth (Slow) Scroll (DECSCLM), VT100.
			term.setSmoothScroll()

		case "6":
			// Origin Mode (DECOM), VT100.
			term._originMode = true

		case "7":
			// Auto-Wrap Mode (DECAWM), VT100.
			term.csiNoAutoLineWrap(false)

		case "12", "25":
			// Start Blinking Cursor (att610) / Show Cursor (DECTCEM)
			term.csiCursorShow()

		case "47", "1047":
			// alt screen buffer
			term.csiScreenBufferAlternative()

		case "1048":
			term.csiCursorPosSave()

		case "1049":
			term.csiCursorPosSave()
			term.csiScreenBufferAlternative()

		case "2004":
			// Set bracketed paste mode
			log.Printf("TODO: Set bracketed paste mode")

		default:
			log.Printf("Private CSI parameter not implemented in %s: %v [param: %s]", string(r), string(code), param)
		}

	case 'K':
		// Erase in Line (DECSEL), VT220. (selective)
		switch param {
		case "", "0":
			term.csiEraseLineAfter()
		case "1":
			term.csiEraseLineBefore()
		case "2":
			term.csiEraseLine()
		default:
			log.Printf("WARNING: Unknown Erase in Line (EL) sequence: %s", param)
		}

	case 'l':
		// DEC Private Mode Reset (DECRST).
		switch param {
		case "1":
			// Normal Cursor Keys (DECCKM), VT100.
			term.renderer.SetKeyboardFnMode(types.KeysApplication)

		case "2":
			// Designate VT52 mode (DECANM), VT100.
			term._vtMode = _VT52

		case "3":
			// 80 Column Mode (DECCOLM), VT100.
			term.resize80()

		case "4":
			// Jump (Fast) Scroll (DECSCLM), VT100.
			term.setJumpScroll()

		case "6":
			// Normal Cursor Mode (DECOM), VT100.
			term._originMode = false

		case "7":
			// No Auto-Wrap Mode (DECAWM), VT100.
			term.csiNoAutoLineWrap(true)

		case "12", "25":
			// Stop Blinking Cursor (att610) / Hide Cursor (DECTCEM)
			term.csiCursorHide()

		case "47", "1047":
			// normal screen buffer
			term.csiScreenBufferNormal()

		case "1048":
			term.csiCursorPosRestore()

		case "1049":
			term.csiScreenBufferNormal()
			term.csiCursorPosRestore()

		case "2004":
			// Reset bracketed paste mode
			log.Printf("TODO: Reset bracketed paste mode")

		default:
			log.Printf("Private CSI parameter not implemented in %s: %v [param: %s]", string(r), string(code), param)
		}

	default:
		log.Printf("Private CSI code not implemented: %s (%s)", string(r), string(code))
	}
}
