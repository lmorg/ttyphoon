package codes


type KeyName string

func (k KeyName) Code() (KeyCode, Modifier) {
	return KeyNameToCode(k)
}

func KeyNameToCode(key KeyName) (KeyCode, Modifier) {
	var mod Modifier

	for len(key) > 2 && key[1] == '-' {
		switch key[0] {
		case 'C':
			mod |= MOD_CTRL
		case 'A', 'O':
			mod |= MOD_ALT // or option on macOS
		case 'S':
			mod |= MOD_SHIFT
		case 'M':
			mod |= MOD_META
		default:
			return 0, 0
		}
		key = key[2:]
	}

	switch key {
	case "Enter":
		return '\n', mod
	case "Return":
		return '\n', mod
	case "Escape":
		return AsciiEscape, mod
	case "Space":
		return ' ', mod
	case "Tab":
		return '\t', mod
	case "BackSpace":
		return AsciiBackspace, mod // TODO: ASCII or ISO?

	case "F1":
		return AnsiF1, mod
	case "F2":
		return AnsiF2, mod
	case "F3":
		return AnsiF3, mod
	case "F4":
		return AnsiF4, mod
	case "F5":
		return AnsiF5, mod
	case "F6":
		return AnsiF6, mod
	case "F7":
		return AnsiF7, mod
	case "F8":
		return AnsiF8, mod
	case "F9":
		return AnsiF9, mod
	case "F10":
		return AnsiF10, mod
	case "F11":
		return AnsiF11, mod
	case "F12":
		return AnsiF12, mod
	case "F13":
		return AnsiF13, mod
	case "F14":
		return AnsiF14, mod
	case "F15":
		return AnsiF15, mod
	case "F16":
		return AnsiF16, mod
	case "F17":
		return AnsiF17, mod
	case "F18":
		return AnsiF18, mod
	case "F19":
		return AnsiF19, mod
	case "F20":
		return AnsiF20, mod

	case "Up":
		return AnsiUp, mod
	case "Down":
		return AnsiDown, mod
	case "Right":
		return AnsiRight, mod
	case "Left":
		return AnsiLeft, mod
	case "Insert":
		return AnsiInsert, mod
	case "Home":
		return AnsiHome, mod
	case "Delete":
		return AnsiDelete, mod
	case "PageUp":
		return AnsiPageUp, mod
	case "PageDown":
		return AnsiPageDown, mod

	case "KeyPadSpace":
		return AnsiKeyPadSpace, mod
	case "KeyPadTab":
		return AnsiKeyPadTab, mod
	case "KeyPadEnter":
		return AnsiKeyPadEnter, mod
	case "KeyPadMultiply":
		return AnsiKeyPadMultiply, mod
	case "KeyPadAdd":
		return AnsiKeyPadAdd, mod
	case "KeyPadComma":
		return AnsiKeyPadComma, mod
	case "KeyPadMinus":
		return AnsiKeyPadMinus, mod
	case "KeyPadPeriod":
		return AnsiKeyPadPeriod, mod
	case "KeyPadDivide":
		return AnsiKeyPadDivide, mod
	case "KeyPad0":
		return AnsiKeyPad0, mod
	case "KeyPad1":
		return AnsiKeyPad1, mod
	case "KeyPad2":
		return AnsiKeyPad2, mod
	case "KeyPad3":
		return AnsiKeyPad3, mod
	case "KeyPad4":
		return AnsiKeyPad4, mod
	case "KeyPad5":
		return AnsiKeyPad5, mod
	case "KeyPad6":
		return AnsiKeyPad6, mod
	case "KeyPad7":
		return AnsiKeyPad7, mod
	case "KeyPad8":
		return AnsiKeyPad8, mod
	case "KeyPad9":
		return AnsiKeyPad9, mod
	case "KeyPadEqual":
		return AnsiKeyPadEqual, mod

	default:
		if len(key) != 1 {
			return 0, mod
		}

		return KeyCode(key[0]), mod
	}
}
