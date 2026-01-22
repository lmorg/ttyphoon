package virtualterm

import (
	"log"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/debug"
)

func (term *Term) readLoop() {
	var (
		r   rune
		err error
	)

	for {
		r, err = term.Pty.Read()
		if err != nil {
			return
		}
		//term._slowBlinkState = true
		//term.renderer.SetBlinkState(true)
		term.renderer.TriggerLazyRedraw()

		term._mutex.Lock()
		term.readChar(r)
		term._mutex.Unlock()
	}
}

/*
	Reference documentation used:
	- ASCII table: https://upload.wikimedia.org/wikipedia/commons/thumb/1/1b/ASCII-Table-wide.svg/1280px-ASCII-Table-wide.svg.png
	- xterm: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Single-character-functions
*/

func (term *Term) readChar(r rune) {
	//writeDebuggingRune(r)

	/*if r < ' ' {
		term._phrase = nil
	}*/

	switch r {

	case 7:
		// Ctrl+G: Bell (BELL)
		term.renderer.Bell()

	case 8, 127:
		// Backspace (BS) aka ^H
		// Delete (DEL) aka ^?
		_ = term.csiMoveCursorBackwards(1)

	case 9:
		// Ctrl+I: Horizontal Tab (HT) aka \t
		if term.writeToElement(r) {
			return
		}
		term.printTab()

	case 10:
		// Ctrl+J: Line Feed (LF) aka \n
		if term.writeToElement(r) {
			return
		}
		//term.phraseAppend(r)
		term.lineFeed(_LINEFEED_CURSOR_MOVED)

	case 11:
		// Ctrl+K: Vertical Tab:
		term.lineFeed(_LINEFEED_CURSOR_MOVED)

	case 12:
		// Ctrl+L: Form Feed (FF)
		term.lineFeed(_LINEFEED_CURSOR_MOVED)

	case 13:
		// Ctrl+M: Carriage Return (CR) aka \r
		term.carriageReturn()

	case 14:
		// Ctrl+N: Shift Out (SO)
		term._activeCharSet = 1

	case 15:
		// Ctrl+O: Shift In (SI)
		term._activeCharSet = 0

	case codes.AsciiEscape:
		// 27: escape, {ESC}
		switch term._vtMode {
		case _VT52:
			term.parseVt52Codes()
		default:
			term.parseC1Codes()
		}

	default:
		if r < ' ' {
			log.Printf("WARNING: Unexpected ASCII control character (ignored): %d", r)
			return
		}

		if term._charSetG[term._activeCharSet] != nil {
			char := term._charSetG[term._activeCharSet][r]
			if char != 0 {
				term.writeCell(char, nil)
				return
			}
		}

		term.writeCell(r, nil)
	}
}

func writeDebuggingRune(r rune) {
	if !debug.Enabled {
		return
	}

	if r <= ' ' {
		debug.Log(r)
	} else {
		debug.Log(string(r))
	}
}
