package virtualterm

import (
	"log"
	"os"
	"strings"
)

func (term *Term) osc1337iTerm2(params []string) {
	// Reference docs:
	// - https://iterm2.com/documentation-escape-codes.html

	kv := strings.SplitN(params[0], "=", 2)

	switch kv[0] {
	case "ClearScrollback":
		term.eraseScrollBack()

	case "StealFocus":
		term.renderer.ShowAndFocusWindow()

	case "CurrentDir":
		if len(kv) == 1 {
			return
		}
		host, err := os.Hostname()
		if err != nil {
			host = "localhost"
		}
		_osc7UpdatePath(term, host, kv[1])

	default:
		log.Printf("unsupported iTerm2 escape sequence: %s", params[0])
	}
}
