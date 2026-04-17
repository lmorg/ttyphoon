package virtualterm

import (
	"log"
	"strconv"
	"strings"

	"github.com/lmorg/ttyphoon/codes"
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
	case 'c':
		// Device Attributes response (DA), e.g. "CSI ? 65;... c".
		// This is typically terminal output and does not change local parser state.
		// Accept and ignore it so we do not leak warnings when this response is
		// looped back through intermediate PTY layers.
		//log.Println("INFO: Device Attributes response (DA) (lookupPrivateCsi() c) swallowed")

	case 'h':
		// DEC Private Mode Set (DECSET).
		for _, p := range strings.Split(param, ";") {
			i, err := strconv.Atoi(p)
			if err != nil {
				log.Printf("Private CSI parameter not valid in %s: %v [param: %s]", string(r), string(code), p)
				continue
			}
			switch i {
			case 1:
				// Application Cursor Keys (DECCKM), VT100.
				term.renderer.SetKeyboardFnMode(types.KeysApplication)

			//case 2:
			// Designate USASCII for character sets G0-G3 (DECANM), VT100, and set VT100 mode.

			case 3:
				// 132 Column Mode (DECCOLM), VT100.
				term.resize132()

			case 4:
				// Smooth (Slow) Scroll (DECSCLM), VT100.
				term.setSmoothScroll()

			case 5:
				// Reverse Video (DECSCNM), VT100.

			case 6:
				// Origin Mode (DECOM), VT100.
				term._originMode = true

			case 7:
				// Auto-Wrap Mode (DECAWM), VT100.
				term.csiNoAutoLineWrap(false)

			//case 8:
			// Auto-Repeat Keys (DECARM), VT100.

			case 9:
				// Send Mouse X & Y on button press
				// This is the X10 xterm mouse protocol.
				// See https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
				term._mouseTracking = codes.MouseTrackingX10

			//case 10:
			// Show toolbar (rxvt).

			//case 14:
			// Enable XOR of blinking cursor control sequence and menu.

			case 12, 25:
				// 12: Start blinking cursor (AT&T 610).
				// 25: Show cursor (DECTCEM), VT220.
				term.csiCursorShow()

			case 47, 1047:
				// alt screen buffer
				term.csiScreenBufferAlternative()

			case 1048:
				term.csiCursorPosSave()

			case 1049:
				term.csiCursorPosSave()
				term.csiScreenBufferAlternative()

			case 1000:
				// Send Mouse X & Y on button press and release
				// This is the X11 xterm mouse protocol.
				// See https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
				term._mouseTracking = codes.MouseTrackingX10

				//case "1001:
				// Use Hilite Mouse Tracking

			case 1002:
				// Use Cell Motion Mouse Tracking.
				// See https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Button-event-tracking
				term._mouseTracking = codes.MouseTrackingButtonEvent

			case 1003:
				// Use All Motion Mouse Tracking, xterm.
				// See https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Any-event-tracking
				term._mouseTracking = codes.MouseTrackingAnyEvent

			case 1004:
				// Send FocusIn/FocusOut events, xterm.
				// See https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-FocusIn_FocusOut#

			case 1005:
				// Enable UTF-8 Mouse Mode, xterm.
				term._mouseEncoding = codes.MouseEncodingUTF8

			case 1006:
				// Enable SGR Mouse Mode, xterm.
				term._mouseEncoding = codes.MouseEncodingSGR

			case 1015:
				// Enable urxvt Mouse Mode.
				term._mouseEncoding = codes.MouseEncodingURXVT

			case 2004:
				// Set bracketed paste mode
				log.Printf("TODO: Set bracketed paste mode")

			default:
				log.Printf("Private CSI parameter not implemented in %s: %v [param: %s]", string(r), string(code), p)
			}
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
		for _, p := range strings.Split(param, ";") {
			i, err := strconv.Atoi(p)
			if err != nil {
				log.Printf("Private CSI parameter not valid in %s: %v [param: %s]", string(r), string(code), p)
				continue
			}
			switch i {
			case 1:
				// Normal Cursor Keys (DECCKM), VT100.
				term.renderer.SetKeyboardFnMode(types.KeysNormal)

			case 2:
				// Designate VT52 mode (DECANM), VT100.
				term._vtMode = _VT52

			case 3:
				// 80 Column Mode (DECCOLM), VT100.
				term.resize80()

			case 4:
				// Jump (Fast) Scroll (DECSCLM), VT100.
				term.setJumpScroll()

			case 6:
				// Normal Cursor Mode (DECOM), VT100.
				term._originMode = false

			case 7:
				// No Auto-Wrap Mode (DECAWM), VT100.
				term.csiNoAutoLineWrap(true)

			case 9:
				// Don't send Mouse X & Y on button press, xterm.
				term._mouseTracking = codes.MouseTrackingOff

			case 12, 25:
				// Stop Blinking Cursor (att610) / Hide Cursor (DECTCEM)
				term.csiCursorHide()

			case 47, 1047:
				// normal screen buffer
				term.csiScreenBufferNormal()

			case 1048:
				term.csiCursorPosRestore()

			case 1049:
				term.csiScreenBufferNormal()
				term.csiCursorPosRestore()

			case 1000:
				// Don't send Mouse X & Y on button press and release.
				term._mouseTracking = codes.MouseTrackingOff

			case 1001:
				// Don't use Hilite Mouse Tracking, xterm.
				term._mouseTracking = codes.MouseTrackingOff

			case 1002:
				// Don't use Cell Motion Mouse Tracking, xterm.
				term._mouseTracking = codes.MouseTrackingOff

			case 1003:
				// Don't use All Motion Mouse Tracking, xterm.
				term._mouseTracking = codes.MouseTrackingOff

			//case 1004
			// Don't send FocusIn/FocusOut events, xterm.

			case 1005:
				// Disable UTF-8 Mouse Mode, xterm.
				term._mouseEncoding = codes.MouseEncodingDefault

			case 1006:
				// Disable SGR Mouse Mode, xterm.
				term._mouseEncoding = codes.MouseEncodingDefault

			case 1015:
				// Disable urxvt Mouse Mode.
				term._mouseEncoding = codes.MouseEncodingDefault

			case 2004:
				// Reset bracketed paste mode
				log.Printf("TODO: Reset bracketed paste mode")

			default:
				log.Printf("Private CSI parameter not implemented in %s: %v [param: %s]", string(r), string(code), p)
			}
		}

	default:
		log.Printf("Private CSI code not implemented: %s (%s)", string(r), string(code))
	}
}
