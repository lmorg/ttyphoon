package virtualterm

import (
	"strings"

	"github.com/lmorg/ttyphoon/window/backend/cursor"
)

func pointerCursorCSS(params []string) string {
	if len(params) == 0 {
		return "default"
	}

	switch css := strings.TrimSpace(params[0]); css {
	case "default", "pointer", "text":
		return css
	default:
		return "default"
	}
}

func (term *Term) osc22SetPointerCursor(params []string) {
	term._pointerCursorCSS = pointerCursorCSS(params)
	term.applyPointerCursor()
}

func (term *Term) applyPointerCursor() {
	if !term._isFocused {
		return
	}

	cursor.SetBase(pointerCursorCSS([]string{term._pointerCursorCSS}))
}
