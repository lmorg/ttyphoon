package codes

import (
	"github.com/lmorg/mxtty/types"
)

/*
	  Code     Modifiers
	---------+---------------------------
	   2     | Shift
	   3     | Alt
	   4     | Shift + Alt
	   5     | Control
	   6     | Shift + Control
	   7     | Alt + Control
	   8     | Shift + Alt + Control
	   9     | Meta
	   10    | Meta + Shift
	   11    | Meta + Alt
	   12    | Meta + Alt + Shift
	   13    | Meta + Ctrl
	   14    | Meta + Ctrl + Shift
	   15    | Meta + Ctrl + Alt
	   16    | Meta + Ctrl + Alt + Shift
	---------+---------------------------

	For example, shift-F5 would be sent as CSI 1 5 ; 2 ~
*/

type Modifier int

const (
	// No modifiers used
	MOD_NONE = 0

	// Shift
	MOD_SHIFT Modifier = 1 << iota

	// Alt / Option
	MOD_ALT

	// Ctrl
	MOD_CTRL

	// Meta / Command
	MOD_META
)

func (mod Modifier) Is(flag Modifier) bool {
	return mod&flag != 0
}

func translateModToAnsiCode(mod Modifier) []byte {
	switch mod {
	case MOD_SHIFT:
		return []byte{';', '2'}

	case MOD_ALT:
		return []byte{';', '3'}

	case MOD_SHIFT | MOD_ALT:
		return []byte{';', '4'}

	case MOD_CTRL:
		return []byte{';', '5'}

	case MOD_SHIFT | MOD_CTRL:
		return []byte{';', '6'}

	case MOD_ALT | MOD_CTRL:
		return []byte{';', '7'}

	case MOD_SHIFT | MOD_ALT | MOD_CTRL:
		return []byte{';', '8'}

	case MOD_META:
		return []byte{';', '9'}

	case MOD_META | MOD_SHIFT:
		return []byte{';', '1', '0'}

	case MOD_META | MOD_ALT:
		return []byte{';', '1', '1'}

	case MOD_META | MOD_ALT | MOD_SHIFT:
		return []byte{';', '1', '2'}

	case MOD_META | MOD_CTRL:
		return []byte{';', '1', '3'}

	case MOD_META | MOD_CTRL | MOD_SHIFT:
		return []byte{';', '1', '4'}

	case MOD_META | MOD_CTRL | MOD_ALT:
		return []byte{';', '1', '5'}

	case MOD_META | MOD_CTRL | MOD_ALT | MOD_SHIFT:
		return []byte{';', '1', '6'}

	default:
		panic("invalid modifier")
	}
}

func translateModToTmuxCode(mod Modifier) []byte {
	b := []byte{0} // zero prefix to work around char codes vs char names!

	if mod.Is(MOD_CTRL) {
		b = append(b, []byte{'C', '-'}...)
	}

	if mod.Is(MOD_SHIFT) {
		b = append(b, []byte{'S', '-'}...)
	}

	if mod.Is(MOD_META) {
		b = append(b, []byte{'M', '-'}...)
	}

	return b
}

func specialCaseSequences(keySet types.KeyboardMode, keyPress KeyCode, modifier Modifier) []byte {
	switch {
	case keySet == types.KeysTmuxClient:
		return nil

	case keyPress < 256 && modifier == 0:
		return []byte{byte(keyPress)}

	case keyPress > '`' && keyPress < 'z' && modifier == MOD_CTRL:
		return []byte{byte(keyPress) - 0x60}
	}

	return nil
}
