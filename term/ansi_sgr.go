package virtualterm

import (
	"log"

	"github.com/lmorg/ttyphoon/types"
)

/*
	Reference documentation used:
	- https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
	- https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
*/

func lookupSgr(sgr *types.Sgr, n int32, stack []int32) {
	for _, i := range stack {
		switch i {
		case -1, 0: // Normal (default), VT100
			sgr.Reset()

		case 1: // Bold, VT100
			sgr.Bitwise.Set(types.SGR_BOLD)

		case 2: // Faint, decreased intensity, ECMA-48 2nd
			sgr.Bitwise.Set(types.SGR_FAINT)

		case 3: // Italicized, ECMA-48 2nd
			sgr.Bitwise.Set(types.SGR_ITALIC)

		case 4: // Underlined, VT100
			sgr.Bitwise.Set(types.SGR_UNDERLINE)

		case 5, // Blink, VT100
			6: // (fast blink)
			sgr.Bitwise.Set(types.SGR_SLOW_BLINK)

		case 7: // Inverse, VT100
			sgr.Bitwise.Set(types.SGR_INVERT)

		//case 8: // Invisible, i.e., hidden, ECMA-48 2nd, VT300

		//case 9: // Crossed-out characters, ECMA-48 3rd

		//case 21: // Doubly-underlined, ECMA-48 3rd

		case 22: // no bold
			sgr.Bitwise.Unset(types.SGR_BOLD)

		case 23: // no italic
			sgr.Bitwise.Unset(types.SGR_ITALIC)

		case 24: // no underline
			sgr.Bitwise.Unset(types.SGR_UNDERLINE)

		case 25: // no blink
			sgr.Bitwise.Unset(types.SGR_SLOW_BLINK)

		case 27: // no invert
			sgr.Bitwise.Unset(types.SGR_INVERT)

		//case 28: // Visible, i.e., not hidden, ECMA-48 3rd, VT300

		//case 29: // Not crossed-out, ECMA-48 3rd

		//
		// 3-bit foreground colour:
		//

		case 30: // fg black
			sgr.Fg = types.SGR_COLOR_BLACK

		case 31: // fg red
			sgr.Fg = types.SGR_COLOR_RED

		case 32: // fg green
			sgr.Fg = types.SGR_COLOR_GREEN

		case 33: // fg yellow
			sgr.Fg = types.SGR_COLOR_YELLOW

		case 34: // fg blue
			sgr.Fg = types.SGR_COLOR_BLUE

		case 35: // fg magenta
			sgr.Fg = types.SGR_COLOR_MAGENTA

		case 36: // fg cyan
			sgr.Fg = types.SGR_COLOR_CYAN

		case 37: // fg white
			sgr.Fg = types.SGR_COLOR_WHITE

		case 38:
			colour := _sgrEnhancedColour(n, stack)
			if colour != nil {
				sgr.Fg = colour
			}
			return

		case 39: // fg default
			sgr.Fg = types.SGR_COLOR_FOREGROUND

		//
		// 3-bit background colour:
		//

		case 40: // bg black
			sgr.Bg = types.SGR_COLOR_BLACK

		case 41: // bg rede
			sgr.Bg = types.SGR_COLOR_RED

		case 42: // bg green
			sgr.Bg = types.SGR_COLOR_GREEN

		case 43: // bg yellow
			sgr.Bg = types.SGR_COLOR_YELLOW

		case 44: // bg blue
			sgr.Bg = types.SGR_COLOR_BLUE

		case 45: // bg magenta
			sgr.Bg = types.SGR_COLOR_MAGENTA

		case 46: // bg cyan
			sgr.Bg = types.SGR_COLOR_CYAN

		case 47: // bg white
			sgr.Bg = types.SGR_COLOR_WHITE

		case 48:
			colour := _sgrEnhancedColour(n, stack)
			if colour != nil {
				sgr.Bg = colour
			}
			return

		case 49: // bg default
			sgr.Bg = types.SGR_COLOR_BACKGROUND

		//
		// 4-bit foreground colour:
		//

		case 90: // fg black
			sgr.Fg = types.SGR_COLOR_BLACK_BRIGHT

		case 91: // fg red
			sgr.Fg = types.SGR_COLOR_RED_BRIGHT

		case 92: // fg green
			sgr.Fg = types.SGR_COLOR_GREEN_BRIGHT

		case 93: // fg yellow
			sgr.Fg = types.SGR_COLOR_YELLOW_BRIGHT

		case 94: // fg blue
			sgr.Fg = types.SGR_COLOR_BLUE_BRIGHT

		case 95: // fg magenta
			sgr.Fg = types.SGR_COLOR_MAGENTA_BRIGHT

		case 96: // fg cyan
			sgr.Fg = types.SGR_COLOR_CYAN_BRIGHT

		case 97: // fg white
			sgr.Fg = types.SGR_COLOR_WHITE_BRIGHT

		//
		// 4-bit background colour:
		//

		case 100: // bg black
			sgr.Bg = types.SGR_COLOR_BLACK_BRIGHT

		case 101: // bg red
			sgr.Bg = types.SGR_COLOR_RED_BRIGHT

		case 102: // bg green
			sgr.Bg = types.SGR_COLOR_GREEN_BRIGHT

		case 103: // bg yellow
			sgr.Bg = types.SGR_COLOR_YELLOW_BRIGHT

		case 104: // bg blue
			sgr.Bg = types.SGR_COLOR_BLUE_BRIGHT

		case 105: // bg magenta
			sgr.Bg = types.SGR_COLOR_MAGENTA_BRIGHT

		case 106: // bg cyan
			sgr.Bg = types.SGR_COLOR_CYAN_BRIGHT

		case 107: // bg white
			sgr.Bg = types.SGR_COLOR_WHITE_BRIGHT

		default:
			log.Printf("WARNING: Unknown SGR code: %d", n)
		}
	}
}

func _sgrEnhancedColour(n int32, stack []int32) *types.Colour {
	if len(stack) < 2 {
		log.Printf("SGR error: too few parameters in %d: %v", n, stack)
		return nil
	}
	switch stack[1] {
	case 5:
		colour, ok := types.SGR_COLOR_256[stack[2]]
		if !ok {
			log.Printf("WARNING: SGR error: 256 value does not exist in %d: %v", n, stack)
			return nil
		}
		return colour

	case 2:
		if len(stack) != 5 {
			log.Printf("WARNING: SGR error: too few parameters in %d (24bit): %v", n, stack)
			return nil
		}
		return &types.Colour{
			Red:   byte(stack[2]),
			Green: byte(stack[3]),
			Blue:  byte(stack[4]),
		}

	default:
		log.Printf("WARNING: SGR error: unexpected value in %d: %v", n, stack)
		return nil
	}

}
