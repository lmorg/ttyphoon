package rendererwebkit

import (
	"strings"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/hotkeys"
)

func (wr *webkitRender) HandleTextInput(text string) {
	term := wr.activeTerm()
	if term == nil || text == "" {
		return
	}

	b := []byte(text)
	if len(b) == 1 {
		if wr.hotkey(codes.KeyCode(b[0]), 0) {
			wr.TriggerRedraw()
			return
		}
	}

	term.Reply(b)
	wr.TriggerRedraw()
}

func (wr *webkitRender) HandleKeyPress(key string, ctrl, alt, shift, meta bool) {
	term := wr.activeTerm()
	if term == nil || key == "" {
		return
	}

	wr.SetBlinkState(true)

	if isModifierOnlyKey(key) {
		return
	}

	mod := keyEventModToCodesModifier(ctrl, alt, shift, meta)
	keyCode := webKeyCodeLookup(key)
	if keyCode == 0 {
		return
	}

	// Let text input handle plain printable keys to preserve layout-specific glyphs.
	if keyCode > ' ' && keyCode < 127 && (mod == 0 || mod == codes.MOD_SHIFT) {
		return
	}

	if wr.hotkey(keyCode, mod) {
		wr.TriggerRedraw()
		return
	}

	b := codes.GetAnsiEscSeq(wr.keyboardMode, keyCode, mod)
	if len(b) > 0 {
		term.Reply(b)
		wr.TriggerRedraw()
	}
}

func (wr *webkitRender) hotkey(keyCode codes.KeyCode, mod codes.Modifier) bool {
	fn := hotkeys.KeyPress(keyCode, mod)
	if fn == nil {
		return false
	}

	fn()
	return true
}

func (wr *webkitRender) activeTerm() interface{ Reply([]byte) } {
	if wr.termWin == nil {
		return nil
	}

	if wr.termWin.Active != nil && wr.termWin.Active.GetTerm() != nil {
		return wr.termWin.Active.GetTerm()
	}

	for _, tile := range wr.termWin.Tiles {
		if tile != nil && tile.GetTerm() != nil {
			return tile.GetTerm()
		}
	}

	return nil
}

func keyEventModToCodesModifier(ctrl, alt, shift, meta bool) codes.Modifier {
	var mod codes.Modifier

	if ctrl {
		mod |= codes.MOD_CTRL
	}
	if alt {
		mod |= codes.MOD_ALT
	}
	if shift {
		mod |= codes.MOD_SHIFT
	}
	if meta {
		mod |= codes.MOD_META
	}

	return mod
}

func isModifierOnlyKey(key string) bool {
	switch key {
	case "Shift", "Control", "Alt", "Meta":
		return true
	default:
		return false
	}
}

func webKeyCodeLookup(key string) codes.KeyCode {
	normalized := normalizeWebKey(key)
	if normalized == "" {
		return 0
	}

	c, _ := codes.KeyNameToCode(codes.KeyName(normalized))
	return c
}

func normalizeWebKey(key string) string {
	switch key {
	case "ArrowUp":
		return "Up"
	case "ArrowDown":
		return "Down"
	case "ArrowLeft":
		return "Left"
	case "ArrowRight":
		return "Right"
	case "Backspace":
		return "BackSpace"
	case " ":
		return "Space"
	}

	if strings.HasPrefix(key, "Numpad") {
		suffix := strings.TrimPrefix(key, "Numpad")
		switch suffix {
		case "Enter":
			return "KeyPadEnter"
		case "Multiply":
			return "KeyPadMultiply"
		case "Add":
			return "KeyPadAdd"
		case "Subtract":
			return "KeyPadMinus"
		case "Decimal":
			return "KeyPadPeriod"
		case "Divide":
			return "KeyPadDivide"
		case "Equal":
			return "KeyPadEqual"
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			return "KeyPad" + suffix
		}
	}

	return key
}
