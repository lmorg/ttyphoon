package virtualterm

import (
	"fmt"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/types"
)

/*
	Reference documentation used:
	- xterm: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Application-Program-Command-functions
*/

func (term *Term) parseApcCodes() {
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
			if r == '\\' { // ST (APC terminator)
				text = text[:len(text)-1]
				break
			}
			text = append(text, r)
			continue
		}
		if r == codes.AsciiCtrlG { // bell (xterm OSC terminator)
			text = text[:len(text)-1]
			break
		}
	}

	apc := types.NewApcSlice(text)

	switch apc.Index(0) {
	case "begin":
		/*if term._apcStack > 0 {
			term._apcStack++
			break
		}
		term._apcStack = 1*/

		switch apc.Index(1) {
		case "csv":
			term.mxapcBegin(types.ELEMENT_ID_CSV, apc)

		case "md-table":
			term.mxapcBegin(types.ELEMENT_ID_MARKDOWN_TABLE, apc)

		case "code-block":
			term.mxapcBegin(types.ELEMENT_ID_CODEBLOCK, apc)

		case "output-block":
			//term._apcStack--
			term.mxapcBeginOutputBlock(apc)

		default:
			//term._apcStack--
			term.renderer.DisplayNotification(types.NOTIFY_DEBUG,
				fmt.Sprintf("Unknown mxAPC code %s: %s", apc.Index(1), string(text[:len(text)-1])))
		}

	case "end":
		/*if term._apcStack > 1 {
			term._apcStack--
			break
		}
		term._apcStack = 0*/

		switch apc.Index(1) {
		case "csv":
			term.mxapcEnd(apc)

		case "md-table":
			term.mxapcEnd(apc)

		case "code-block":
			term.mxapcEnd(apc)

		case "output-block":
			//term._apcStack++
			term.mxapcEndOutputBlock(apc)

		default:
			//term._apcStack++
			term.renderer.DisplayNotification(types.NOTIFY_DEBUG,
				fmt.Sprintf("Unknown mxAPC code %s: %s", apc.Index(1), string(text[:len(text)-1])))
		}

	case "insert":
		/*if term._apcStack > 0 {
			break
		}*/

		switch apc.Index(1) {
		case "image":
			term.mxapcInsert(types.ELEMENT_ID_IMAGE, apc)
		default:
			term.renderer.DisplayNotification(types.NOTIFY_DEBUG,
				fmt.Sprintf("Unknown mxAPC code %s: %s", apc.Index(1), string(text[:len(text)-1])))
		}

	case "config":
		switch apc.Index(1) {
		case "export":
			term.mxapcConfigExport(apc)
		//case "var":
		//	term.mxapcConfigVariables(apc)
		case "unset":
			term.mxapcConfigUnset(apc)
		case "mcp":
			term.mxapcConfigMcp(apc)
		default:
			term.renderer.DisplayNotification(types.NOTIFY_DEBUG,
				fmt.Sprintf("Unknown mxAPC code %s: %s", apc.Index(1), string(text[:len(text)-1])))
		}

	default:
		term.renderer.DisplayNotification(types.NOTIFY_DEBUG,
			fmt.Sprintf("Unknown mxAPC code %s: %s", apc.Index(1), string(text[:len(text)-1])))
	}
}
