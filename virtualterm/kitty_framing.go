package virtualterm

import (
	"fmt"
	"log"
	"strings"
)

func (term *Term) readKittyAPC() {
	text, err := term.readGraphicsString(false, 8192)
	if err != nil {
		log.Printf("Kitty graphics: %v", err)
		if term._kitty != nil {
			term._kitty.upload.Reset()
		}
		return
	}
	term.kittyCommand(text)
}

// readGraphicsString consumes through ST even on oversized input, so payload
// bytes never escape into normal terminal text. tmux doubles embedded ESCs.
func (term *Term) readGraphicsString(tmux bool, limit int) (string, error) {
	var text strings.Builder
	oversized := false
	for {
		r, err := term.Pty.Read()
		if err != nil {
			return "", err
		}
		if r == '\x1b' {
			next, err := term.Pty.Read()
			if err != nil {
				return "", err
			}
			if next == '\\' {
				if oversized {
					return "", fmt.Errorf("E2BIG:graphics escape sequence exceeds limit")
				}
				return text.String(), nil
			}
			if tmux && next == '\x1b' {
				r = '\x1b'
			} else {
				oversized = true
				continue
			}
		}
		if text.Len() < limit {
			text.WriteRune(r)
		} else {
			oversized = true
		}
	}
}

func (term *Term) readTmuxGraphics() {
	for _, expected := range "mux;" {
		r, err := term.Pty.Read()
		if err != nil {
			return
		}
		if r != expected {
			_, _ = term.readGraphicsString(false, 32768)
			log.Printf("WARNING: Unsupported DCS sequence")
			return
		}
	}
	text, err := term.readGraphicsString(true, 32768)
	if err == nil {
		err = term.dispatchTmuxGraphics(text, 0)
	}
	if err != nil {
		log.Printf("Kitty graphics passthrough: %v", err)
		if term._kitty != nil {
			term._kitty.upload.Reset()
		}
	}
}

func (term *Term) dispatchTmuxGraphics(text string, depth int) error {
	if depth >= 4 {
		return fmt.Errorf("E2BIG:tmux passthrough nesting limit")
	}
	if strings.HasPrefix(text, "\x1bPtmux;") && strings.HasSuffix(text, "\x1b\\") {
		inner := strings.TrimSuffix(strings.TrimPrefix(text, "\x1bPtmux;"), "\x1b\\")
		return term.dispatchTmuxGraphics(strings.ReplaceAll(inner, "\x1b\x1b", "\x1b"), depth+1)
	}
	for text != "" {
		if !strings.HasPrefix(text, "\x1b_G") {
			return fmt.Errorf("ENOTSUP:only Kitty graphics passthrough is supported")
		}
		payload, remainder, ok := strings.Cut(text[3:], "\x1b\\")
		if !ok || len(payload) > 8192 {
			return fmt.Errorf("EINVAL:invalid graphics passthrough")
		}
		term.kittyCommand(payload)
		text = remainder
	}
	return nil
}
