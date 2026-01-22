package virtualterm

import (
	"fmt"
	"log"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

/*
	Reference documentation used:
	- xterm: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Functions-using-CSI-_-ordered-by-the-final-character_s_
	- Wikipedia: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
	- HN discussion: https://news.ycombinator.com/item?id=38849690
	- ChatGPT (when the documentation above was unclear)
*/

func (term *Term) parseCsiCodes() {
	var (
		r       rune
		err     error
		stack   = []int32{0} // default value is 0
		n       = &stack[0]
		cache   []rune
		unknown bool
	)

	for {
		r, err = term.Pty.Read()
		if err != nil {
			return
		}
		cache = append(cache, r)
		if r >= '0' && '9' >= r {
			multiplyN(n, r)
			continue
		}

		if r < ' ' && r != codes.AsciiEscape {
			term.readChar(r)
			continue
		}

		if isCsiTerminator(r) {
			debug.Log(string(cache))
		}

		switch r {
		case '@':
			// Insert Ps (Blank) Character(s) (default = 1) (ICH)
			term.csiInsertCharacters(*n)

		case 'a':
			// Character Position Relative  [columns] (default = [row,col+1]) (HPR).
			term.csiMoveCursorForwards(*n)

		case 'A':
			// Cursor Up Ps Times (default = 1) (CUU).
			term.csiMoveCursorUpwards(*n)

		case 'b':
			// Repeat the preceding graphic character Ps times (REP).
			term.csiRepeatPreceding(*n)

		case 'B':
			// Cursor Down Ps Times (default = 1) (CUD).
			term.csiMoveCursorDownwards(*n)

		case 'c':
			// Send Device Attributes (Primary DA).
			// send reply: "\0x1B[?1;" + https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Functions-using-CSI-_-ordered-by-the-final-character_s_
			reply := append(codes.Csi, []byte("?65;1;6;15;17;22;28;29c")...)
			term.Reply(reply)

		case 'C':
			// Cursor Forward Ps Times (default = 1) (CUF).
			term.csiMoveCursorForwards(*n)

		case 'd':
			// Line Position Absolute  [row] (default = [1,column]) (VPA).
			term.moveCursorToRow(*n)

		case 'D':
			// Cursor Backward Ps Times (default = 1) (CUB).
			term.csiMoveCursorBackwards(*n)

		case 'e':
			// Line Position Relative  [rows] (default = [row+1,column]) (VPR).
			if *n < 0 {
				term.renderer.DisplayNotification(types.NOTIFY_DEBUG, fmt.Sprintf("VPR is negative value: %d", *n))
			}
			term.csiMoveCursorDownwards(*n)

		case 'E':
			// Cursor Next Line Ps Times (default = 1) (CNL).
			term.csiMoveCursorDownwards(*n)
			term._curPos.X = 0

		case 'f':
			// Horizontal and Vertical Position [row;column] (default = [1,1]) (HVP).
			switch len(stack) {
			case 0:
				term.moveCursorToPos(1, 1)
			case 1:
				term.moveCursorToPos(*n, 1)
			case 2:
				term.moveCursorToPos(stack[0], stack[1])
			default:
				term.moveCursorToPos(stack[0], stack[1])
				log.Printf("WARNING: more parameters than expected for %s: %v (%s)", string(r), stack, string(cache))
			}

		case 'F':
			// Cursor Preceding Line Ps Times (default = 1) (CPL).
			term.csiMoveCursorUpwards(*n)
			term._curPos.X = 0

		case 'g':
			// Tab Clear (TBC).  ECMA-48 defines additional codes, but the
			/*
				VT100 user manual notes that it ignores other codes.  DEC's
				later terminals (and xterm) do the same, for compatibility.
				Ps = 0  ⇒  Clear Current Column (default).
				Ps = 3  ⇒  Clear All.
			*/
			switch *n {
			case -1, 0:
				term.csiClearTabStop()
			case 1, 2:
				// Ignored by vt100 and xterm
			case 3:
				term.csiResetTabStops()
			default:
				log.Printf("WARNING: Unhandled parameter for %s: %v (%s)", string(r), stack, string(cache))
			}

		case 'G':
			// Cursor Character Absolute  [column] (default = [row,1]) (CHA).
			term.moveCursorToColumn(*n)

		case 'h':
			// Set Mode (SM).
			switch stack[0] {

			// Ps = 2  ⇒  Keyboard Action Mode (KAM).

			case 4:
				// Insert Mode (IRM).
				term.csiIrmInsertOrReplace(_STATE_IRM_INSERT)

				// Ps = 1 2  ⇒  Send/receive (SRM).
				// Ps = 2 0  ⇒  Automatic Newline (LNM).

			default:
				log.Printf("WARNING: Unknown Set Mode (SM) sequence: %s", string(cache))
			}

		case 'H':
			// Cursor Position [row;column] (default = [1,1]) (CUP).
			switch len(stack) {
			case 0:
				term.moveCursorToPos(1, 1)
			case 1:
				term.moveCursorToPos(*n, 1)
			case 2:
				term.moveCursorToPos(stack[0], stack[1])
			default:
				term.moveCursorToPos(stack[0], stack[1])
				log.Printf("WARNING: invalid parameters %s: %v (%s)", string(r), stack, string(cache))
			}

		//case 'i':
		// CSI Ps i  Media Copy (MC).
		/*
			Ps = 0  ⇒  Print screen (default).
			Ps = 4  ⇒  Turn off printer controller mode.
			Ps = 5  ⇒  Turn on printer controller mode.
			Ps = 1 0  ⇒  HTML screen dump, xterm.
			Ps = 1 1  ⇒  SVG screen dump, xterm.
		*/

		/*case 'I':
		// Cursor Forward Tabulation Ps tab stops (default = 1) (CHT).
		// TODO don't do this!!!
		if *n < 0 {
			term.printTab()
		} else {
			for i := int32(0); i < *n; i++ {
				term.printTab()
			}
		}*/

		case 'J':
			// Erase in Display (ED), VT100.
			switch *n {
			case -1, 0:
				term.csiEraseDisplayAfter()
			case 1:
				term.csiEraseDisplayBefore()
			case 2:
				term.csiEraseDisplay()
			case 3:
				term.csiEraseDisplay()
				term.eraseScrollBack()
			default:
				log.Printf("WARNING: Unknown Erase in Display (ED) sequence: %d", *n)
			}

		case 'K':
			// Erase in Line (EL), VT100.
			switch *n {
			case -1, 0:
				term.csiEraseLineAfter()
			case 1:
				term.csiEraseLineBefore()
			case 2:
				term.csiEraseLine()
			default:
				log.Printf("WARNING: Unknown Erase in Line (EL) sequence: %d", *n)
			}

		case 'l':
			// Reset Mode (RM).
			switch stack[0] {

			// Ps = 2  ⇒  Keyboard Action Mode (KAM).

			case 4:
				// Replace Mode (IRM).
				term.csiIrmInsertOrReplace(_STATE_IRM_REPLACE)

				// Ps = 1 2  ⇒  Send/receive (SRM).
				// Ps = 2 0  ⇒  Normal Linefeed (LNM).

			default:
				log.Printf("WARNING: Unknown Reset Mode (RM) sequence: %d", *n)
			}

		case 'L':
			// Insert Ps Line(s) (default = 1) (IL).
			term.csiInsertLines(*n)

		case 'm':
			// Character Attributes (SGR).
			lookupSgr(term.sgr, stack[0], stack)

		case 'M':
			// Delete Ps Line(s) (default = 1) (DL).
			term.csiDeleteLines(*n)

		case 'n':
			// Device Status Report (DSR).
			/*
				Ps = 5  ⇒  Status Report.
					Result ("OK") is CSI 0 n
				Ps = 6  ⇒  Report Cursor Position (CPR) [row;column].
					Result is CSI r ; c R
			*/
			switch *n {
			case 6:
				pos := term.curPos()
				term.csiCallback("%d;%dR", pos.Y+1, pos.X+1)
			default:
				log.Printf("WARNING: Unknown Device Status Report (DSR) sequence: %d", *n)
			}

		case 'P':
			// Delete Ps Character(s) (default = 1) (DCH).
			term.csiDeleteCharacters(*n)

		case 'q':
			// Load LEDs (DECLL), VT100.
			// Ignored by mxtty

		case 'r':
			// Set Scrolling Region [top;bottom] (default = full size of window) (DECSTBM), VT100.
			switch len(stack) {
			case 0, 1:
				term.unsetScrollingRegion()
			case 2:
				term.setScrollingRegion(stack)
			default:
				log.Printf("WARNING: Unexpected number of parameters in CSI r (%s): %v", string(cache), stack)
			}

		case 's':
			// Save cursor, available only when DECLRMM is disabled (SCOSC, also ANSI.SYS).
			/*
				TODO: this conditional could break the following sequence:
				CSI Pl ; Pr s
					Set left and right margins (DECSLRM), VT420 and up.  This is
					available only when DECLRMM is enabled.
			*/
			term.csiCursorPosSave()

		case 'S':
			// Scroll up Ps lines (default = 1) (SU), VT420, ECMA-48.
			term.csiScrollUp(*n)

		case 't':
			// Window manipulation (XTWINOPS)
			var p2 int32
			if len(stack) > 1 {
				p2 = stack[1]
			}
			switch stack[0] {
			case 22:
				switch p2 {
				case 0, 2:
					term.csiWindowTitleStackSaveTo()
				default:
					log.Printf("WARNING: Unknown Window manipulation (XTWINOPS) sequence %d: %v (%s)", *n, stack, string(cache))
				}
			case 23:
				switch p2 {
				case 0, 2:
					term.csiWindowTitleStackRestoreFrom()
				default:
					log.Printf("WARNING: Unknown Window manipulation (XTWINOPS) sequence %d: %v (%s)", *n, stack, string(cache))
				}
			default:
				log.Printf("WARNING: Unknown Window manipulation (XTWINOPS) sequence %d: %v (%s)", *n, stack, string(cache))
			}

		case 'T':
			// Scroll down Ps lines (default = 1) (SD), VT420.
			term.csiScrollDown(*n)

		case 'u':
			// Restore cursor (SCORC, also ANSI.SYS).
			term.csiCursorPosRestore()

		case 'X':
			// Erase Ps Character(s) (default = 1) (ECH).
			term.csiEraseCharacters(*n)

		//case 'Z':
		// Cursor Backward Tabulation Ps tab stops (default = 1) (CBT).

		case '^':
			// Scroll down Ps lines (default = 1) (SD), ECMA-48.
			// This was a publication error in the original ECMA-48 5th edition (1991) corrected in 2003.
			term.csiScrollDown(*n)

		case '`':
			// Character Position Absolute  [column] (default = [row,1]) (HPA).
			term.moveCursorToColumn(*n)

		//case '!':
		// CSI ! p: Soft terminal reset (DECSTR), VT220 and up.

		case '?': // private codes
			code, err := term.parseCsiExtendedCodes()
			if err != nil {
				return
			}
			lookupPrivateCsi(term, code)
			return

		case '>': // secondary codes
			code, err := term.parseCsiExtendedCodes()
			if err != nil {
				return
			}
			log.Printf("TODO: Secondary CSI code ignored: '%s%s'", string(cache), string(code))
			return

		case '=': // tertiary codes
			code, err := term.parseCsiExtendedCodes()
			if err != nil {
				return
			}
			lookupTertiaryCsi(term, code)
			return

		case ':', ';':
			stack = append(stack, -1)
			n = &stack[len(stack)-1]

		default:
			unknown = true
			if !isCsiTerminator(r) {
				code, err := term.parseCsiExtendedCodes()
				if err != nil {
					return
				}
				log.Printf("WARNING: Unknown extended CSI code %s: %v [string: %s]", string(r), append(cache, code...), string(cache)+string(code))
				return
			}
		}

		if isCsiTerminator(r) {
			if unknown {
				log.Printf("WARNING: Unknown CSI code %s: %v [string: %s]", string(r), cache, string(cache))
			}
			return
		}
	}
}

func (term *Term) parseCsiExtendedCodes() ([]rune, error) {
	var (
		r    rune
		err  error
		code []rune
	)

	for {
		r, err = term.Pty.Read()
		if err != nil {
			return nil, err
		}
		code = append(code, r)
		if isCsiTerminator(r) {
			return code, nil
		}
	}
}

func isCsiTerminator(r rune) bool {
	return r >= 0x40 && r <= 0x7E
}

func multiplyN(n *int32, r rune) {
	if *n < 0 {
		*n = 0
	}

	*n = (*n * 10) + (r - 48)
}
