package rendersdl

import (
	"fmt"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/types"
	"github.com/veandco/go-sdl2/sdl"
)

var keyCodeLookupTable = map[sdl.Keycode]codes.KeyCode{
	sdl.K_UP:    codes.AnsiUp,
	sdl.K_DOWN:  codes.AnsiDown,
	sdl.K_LEFT:  codes.AnsiLeft,
	sdl.K_RIGHT: codes.AnsiRight,

	sdl.K_INSERT:   codes.AnsiInsert,
	sdl.K_DELETE:   codes.AnsiDelete,
	sdl.K_HOME:     codes.AnsiHome,
	sdl.K_END:      codes.AnsiEnd,
	sdl.K_PAGEDOWN: codes.AnsiPageDown,
	sdl.K_PAGEUP:   codes.AnsiPageUp,

	// keypad

	sdl.K_KP_ENTER:    codes.AnsiKeyPadEnter,
	sdl.K_KP_DIVIDE:   codes.AnsiKeyPadDivide,
	sdl.K_KP_MULTIPLY: codes.AnsiKeyPadMultiply,
	sdl.K_KP_MINUS:    codes.AnsiKeyPadMinus,
	sdl.K_KP_PLUS:     codes.AnsiKeyPadAdd,
	sdl.K_KP_PERIOD:   codes.AnsiKeyPadPeriod,
	sdl.K_KP_0:        codes.AnsiKeyPad0,
	sdl.K_KP_1:        codes.AnsiKeyPad1,
	sdl.K_KP_2:        codes.AnsiKeyPad2,
	sdl.K_KP_3:        codes.AnsiKeyPad3,
	sdl.K_KP_4:        codes.AnsiKeyPad4,
	sdl.K_KP_5:        codes.AnsiKeyPad5,
	sdl.K_KP_6:        codes.AnsiKeyPad6,
	sdl.K_KP_7:        codes.AnsiKeyPad7,
	sdl.K_KP_8:        codes.AnsiKeyPad8,
	sdl.K_KP_9:        codes.AnsiKeyPad9,

	// F-Keys

	sdl.K_F1:  codes.AnsiF1,
	sdl.K_F2:  codes.AnsiF2,
	sdl.K_F3:  codes.AnsiF3,
	sdl.K_F4:  codes.AnsiF4,
	sdl.K_F5:  codes.AnsiF5,
	sdl.K_F6:  codes.AnsiF6,
	sdl.K_F7:  codes.AnsiF7,
	sdl.K_F8:  codes.AnsiF8,
	sdl.K_F9:  codes.AnsiF9,
	sdl.K_F10: codes.AnsiF10,
	sdl.K_F11: codes.AnsiF11,
	sdl.K_F12: codes.AnsiF12,
	sdl.K_F13: codes.AnsiF13,
	sdl.K_F14: codes.AnsiF14,
	sdl.K_F15: codes.AnsiF15,
	sdl.K_F16: codes.AnsiF16,
	sdl.K_F17: codes.AnsiF17,
	sdl.K_F18: codes.AnsiF18,
	sdl.K_F19: codes.AnsiF19,
	sdl.K_F20: codes.AnsiF20,
}

func (sr *sdlRender) keyCodeLookup(keyCode sdl.Keycode) (c codes.KeyCode) {
	if keyCode < 256 {
		return codes.KeyCode(keyCode)
	}

	c = keyCodeLookupTable[keyCode]
	if c == 0 {
		sr.DisplayNotification(types.NOTIFY_DEBUG, fmt.Sprintf("Unknown keycode %d", keyCode))
	}
	return
}

func keyEventModToCodesModifier(keyMod sdl.Keymod) codes.Modifier {
	var mod codes.Modifier

	if keyMod&sdl.KMOD_CTRL != 0 || keyMod&sdl.KMOD_LCTRL != 0 || keyMod&sdl.KMOD_RCTRL != 0 {
		mod |= codes.MOD_CTRL
	}

	if keyMod&sdl.KMOD_ALT != 0 || keyMod&sdl.KMOD_LALT != 0 || keyMod&sdl.KMOD_RALT != 0 {
		mod |= codes.MOD_ALT
	}

	if keyMod&sdl.KMOD_SHIFT != 0 || keyMod&sdl.KMOD_LSHIFT != 0 || keyMod&sdl.KMOD_RSHIFT != 0 {
		mod |= codes.MOD_SHIFT
	}

	if keyMod&sdl.KMOD_GUI != 0 || keyMod&sdl.KMOD_LGUI != 0 || keyMod&sdl.KMOD_RGUI != 0 {
		mod |= codes.MOD_META
	}

	return mod
}

func (sr *sdlRender) GetKeyboardModifier() int {
	keyMod := sdl.GetModState()
	return int(keyEventModToCodesModifier(keyMod))
}
