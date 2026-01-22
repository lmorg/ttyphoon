package codes

import (
	"fmt"

	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/octal"
)

const esc = 27

/*
	Reference documentation used:
	- ASCII table: https://upload.wikimedia.org/wikipedia/commons/thumb/1/1b/ASCII-Table-wide.svg/1280px-ASCII-Table-wide.svg.png
	- xterm: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-PC-Style-Function-Keys
*/

var (
	Ss2 = []byte{esc, 'N'}
	Ss3 = []byte{esc, 'O'}
	Csi = []byte{esc, '['}
)

func ss3(b ...byte) []byte { return append(Ss3, b...) }
func csi(b ...byte) []byte { return append(Csi, b...) }

func _tmuxKeyResponse(keyName string) []byte { return append([]byte{0}, []byte(keyName+" ")...) }

var _retryLookUpTable = map[types.KeyboardMode]*[]types.KeyboardMode{
	types.KeysNormal:      {types.KeysNormal, types.KeysVT220},
	types.KeysApplication: {types.KeysApplication, types.KeysNormal, types.KeysVT220},
	types.KeysVT52:        {types.KeysVT52, types.KeysNormal},
	types.KeysVT220:       {types.KeysVT220, types.KeysNormal},
	types.KeysTmuxClient:  {types.KeysTmuxClient},
}

var _ansiLookUpTable = map[types.KeyboardMode]map[KeyCode][]byte{
	types.KeysNormal: {
		AnsiUp:    csi('A'),
		AnsiDown:  csi('B'),
		AnsiRight: csi('C'),
		AnsiLeft:  csi('D'),
		AnsiHome:  csi('H'),
		AnsiEnd:   csi('E'),

		AnsiKeyPadSpace: []byte{' '},
		AnsiKeyPadTab:   []byte{'\t'},
		AnsiKeyPadEnter: []byte{'\r'},

		/*AnsiShiftTab:    csi('Z'),
		AnsiOptUp:       []byte{esc, esc, '[', 'A'},
		AnsiOptDown:     []byte{esc, esc, '[', 'B'},
		AnsiOptLeft:     []byte{esc, esc, '[', 'D'},
		AnsiOptRight:    []byte{esc, esc, '[', 'C'},
		AnsiCtrlUp:      csi('1', ';', '5', 'A'),
		AnsiCtrlDown:    csi('1', ';', '5', 'B'),
		AnsiCtrlLeft:    csi('1', ';', '5', 'D'),
		AnsiCtrlRight:   csi('1', ';', '5', 'C'),*/

		AnsiF1:  ss3('P'),
		AnsiF2:  ss3('Q'),
		AnsiF3:  ss3('R'),
		AnsiF4:  ss3('S'),
		AnsiF5:  csi('1', '5', '~'),
		AnsiF6:  csi('1', '7', '~'),
		AnsiF7:  csi('1', '8', '~'),
		AnsiF8:  csi('1', '9', '~'),
		AnsiF9:  csi('2', '0', '~'),
		AnsiF10: csi('2', '1', '~'),
		AnsiF11: csi('2', '3', '~'),
		AnsiF12: csi('2', '4', '~'),

		/*AnsiShiftF1:  csi('1', ';', '2', 'P'),
		AnsiShiftF2:  csi('1', ';', '2', 'Q'),
		AnsiShiftF3:  csi('1', ';', '2', 'R'),
		AnsiShiftF4:  csi('1', ';', '2', 'S'),
		AnsiShiftF5:  csi('1', '5', ';', '2', '~'),
		AnsiShiftF6:  csi('1', '7', ';', '2', '~'),
		AnsiShiftF7:  csi('1', '8', ';', '2', '~'),
		AnsiShiftF8:  csi('1', '9', ';', '2', '~'),
		AnsiShiftF9:  csi('2', '0', ';', '2', '~'),
		AnsiShiftF10: csi('2', '1', ';', '2', '~'),
		AnsiShiftF11: csi('2', '3', ';', '2', '~'),
		AnsiShiftF12: csi('2', '4', ';', '2', '~'),*/
	},

	types.KeysApplication: {
		AnsiUp:    ss3('A'),
		AnsiDown:  ss3('B'),
		AnsiRight: ss3('C'),
		AnsiLeft:  ss3('D'),
		AnsiHome:  ss3('H'),
		AnsiEnd:   ss3('E'),
	},

	types.KeysVT220: {
		AnsiHome:     csi('1', '~'),
		AnsiInsert:   csi('2', '~'),
		AnsiDelete:   csi('3', '~'),
		AnsiEnd:      csi('4', '~'),
		AnsiPageUp:   csi('5', '~'),
		AnsiPageDown: csi('6', '~'),

		AnsiKeyPadSpace:    ss3(' '),
		AnsiKeyPadTab:      ss3('I'),
		AnsiKeyPadEnter:    ss3('M'),
		AnsiKeyPadMultiply: ss3('j'),
		AnsiKeyPadAdd:      ss3('k'),
		AnsiKeyPadComma:    ss3('l'),
		AnsiKeyPadMinus:    ss3('m'),
		AnsiKeyPadPeriod:   ss3('n'),
		AnsiKeyPadDivide:   ss3('o'),
		AnsiKeyPad0:        ss3('p'),
		AnsiKeyPad1:        ss3('q'),
		AnsiKeyPad2:        ss3('r'),
		AnsiKeyPad3:        ss3('s'),
		AnsiKeyPad4:        ss3('t'),
		AnsiKeyPad5:        ss3('u'),
		AnsiKeyPad6:        ss3('v'),
		AnsiKeyPad7:        ss3('w'),
		AnsiKeyPad8:        ss3('x'),
		AnsiKeyPad9:        ss3('y'),
		AnsiKeyPadEqual:    ss3('X'),

		AnsiF13: csi('2', '5', '~'),
		AnsiF14: csi('2', '6', '~'),
		AnsiF15: csi('2', '8', '~'),
		AnsiF16: csi('2', '9', '~'),
		AnsiF17: csi('3', '1', '~'),
		AnsiF18: csi('3', '2', '~'),
		AnsiF19: csi('3', '3', '~'),
		AnsiF20: csi('3', '4', '~'),
	},

	types.KeysVT52: {
		AnsiUp:    []byte{esc, 'A'},
		AnsiDown:  []byte{esc, 'B'},
		AnsiRight: []byte{esc, 'C'},
		AnsiLeft:  []byte{esc, 'D'},

		AnsiKeyPadSpace:    []byte{esc, '?', ' '},
		AnsiKeyPadTab:      []byte{esc, '?', '\t'},
		AnsiKeyPadEnter:    []byte{esc, '?', 'M'},
		AnsiKeyPadMultiply: []byte{esc, '?', 'j'},
		AnsiKeyPadAdd:      []byte{esc, '?', 'k'},
		AnsiKeyPadComma:    []byte{esc, '?', 'l'},
		AnsiKeyPadMinus:    []byte{esc, '?', 'm'},
		AnsiKeyPadPeriod:   []byte{esc, '?', 'n'},
		AnsiKeyPadDivide:   []byte{esc, '?', 'o'},
		AnsiKeyPad0:        []byte{esc, '?', 'p'},
		AnsiKeyPad1:        []byte{esc, '?', 'q'},
		AnsiKeyPad2:        []byte{esc, '?', 'r'},
		AnsiKeyPad3:        []byte{esc, '?', 's'},
		AnsiKeyPad4:        []byte{esc, '?', 't'},
		AnsiKeyPad5:        []byte{esc, '?', 'u'},
		AnsiKeyPad6:        []byte{esc, '?', 'v'},
		AnsiKeyPad7:        []byte{esc, '?', 'w'},
		AnsiKeyPad8:        []byte{esc, '?', 'x'},
		AnsiKeyPad9:        []byte{esc, '?', 'y'},
		AnsiKeyPadEqual:    []byte{esc, '?', 'X'},
	},

	types.KeysTmuxClient: {
		0:    _tmuxKeyResponse(`\000`),   // 00, null
		1:    _tmuxKeyResponse(`C-a`),    // 01, start of heading
		2:    _tmuxKeyResponse(`C-b`),    // 02, start of text
		3:    _tmuxKeyResponse(`C-c`),    // 03, end of text
		4:    _tmuxKeyResponse(`C-d`),    // 04, end of file
		5:    _tmuxKeyResponse(`C-e`),    // 05, enquiry
		6:    _tmuxKeyResponse(`C-f`),    // 06, acknowledge
		7:    _tmuxKeyResponse(`C-g`),    // 07, bell
		8:    _tmuxKeyResponse(`BSpace`), // 08, backspace
		'\t': _tmuxKeyResponse(`Tab`),    // 09, horizontal tab
		'\n': _tmuxKeyResponse(`Enter`),  // 0a, line feed
		11:   _tmuxKeyResponse(`C-k`),    // 0b, vertical tab
		12:   _tmuxKeyResponse(`C-l`),    // 0c, form feed
		'\r': _tmuxKeyResponse(`Enter`),  // 0d, carriage return
		14:   _tmuxKeyResponse(`C-n`),    // 0e, shift out
		15:   _tmuxKeyResponse(`C-o`),    // 0f, shift in
		16:   _tmuxKeyResponse(`C-p`),    // 10, data link escape
		17:   _tmuxKeyResponse(`C-q`),    // 11, device control 1
		18:   _tmuxKeyResponse(`C-r`),    // 12, device control 2
		19:   _tmuxKeyResponse(`C-s`),    // 13, device control 3
		20:   _tmuxKeyResponse(`C-t`),    // 14, device control 4
		21:   _tmuxKeyResponse(`C-u`),    // 15, negative acknowledge
		22:   _tmuxKeyResponse(`C-v`),    // 16, synchronous idle
		23:   _tmuxKeyResponse(`C-w`),    // 17, end of transmission block
		24:   _tmuxKeyResponse(`C-x`),    // 18, cancel
		25:   _tmuxKeyResponse(`C-y`),    // 19, end of medium
		26:   _tmuxKeyResponse(`C-z`),    // 1a, substitute
		esc:  _tmuxKeyResponse(`Escape`), // 1b. escape
		28:   _tmuxKeyResponse(`\034`),   // 1c, file separator
		29:   _tmuxKeyResponse(`\035`),   // 1d, group separator
		30:   _tmuxKeyResponse(`\036`),   // 1e, record separator
		31:   _tmuxKeyResponse(`\034`),   // 1f, unit separator
		' ':  _tmuxKeyResponse(`Space`),  // 20, space
		'"':  _tmuxKeyResponse(`'"'`),    // 22, double quote
		'\'': _tmuxKeyResponse(`"'"`),    // 27, single quote
		//'-':  _tmuxKeyResponse(`-`),      // 2d, hyphen
		//'_':  _tmuxKeyResponse(`_`),      // 5f, underscore
		127: _tmuxKeyResponse(`Delete`), // 7f, del

		AnsiUp:       _tmuxKeyResponse("Up"),
		AnsiDown:     _tmuxKeyResponse("Down"),
		AnsiRight:    _tmuxKeyResponse("Right"),
		AnsiLeft:     _tmuxKeyResponse("Left"),
		AnsiHome:     _tmuxKeyResponse("Home"),
		AnsiEnd:      _tmuxKeyResponse("End"),
		AnsiInsert:   _tmuxKeyResponse("Insert"),
		AnsiDelete:   _tmuxKeyResponse("Delete"),
		AnsiPageUp:   _tmuxKeyResponse("PageUp"),
		AnsiPageDown: _tmuxKeyResponse("PageDown"),

		AnsiKeyPadSpace:    _tmuxKeyResponse("Space"),
		AnsiKeyPadTab:      _tmuxKeyResponse("Tab"),
		AnsiKeyPadEnter:    _tmuxKeyResponse("KPEnter"),
		AnsiKeyPadMultiply: _tmuxKeyResponse("KP*"),
		AnsiKeyPadAdd:      _tmuxKeyResponse("KP+"),
		AnsiKeyPadComma:    _tmuxKeyResponse("KP,"),
		AnsiKeyPadMinus:    _tmuxKeyResponse("KP-"),
		AnsiKeyPadPeriod:   _tmuxKeyResponse("KP."),
		AnsiKeyPadDivide:   _tmuxKeyResponse("KP/"),
		AnsiKeyPad0:        _tmuxKeyResponse("KP0"),
		AnsiKeyPad1:        _tmuxKeyResponse("KP1"),
		AnsiKeyPad2:        _tmuxKeyResponse("KP2"),
		AnsiKeyPad3:        _tmuxKeyResponse("KP3"),
		AnsiKeyPad4:        _tmuxKeyResponse("KP4"),
		AnsiKeyPad5:        _tmuxKeyResponse("KP5"),
		AnsiKeyPad6:        _tmuxKeyResponse("KP6"),
		AnsiKeyPad7:        _tmuxKeyResponse("KP7"),
		AnsiKeyPad8:        _tmuxKeyResponse("KP8"),
		AnsiKeyPad9:        _tmuxKeyResponse("KP9"),
		AnsiKeyPadEqual:    _tmuxKeyResponse("KP="),

		AnsiF1:  _tmuxKeyResponse("F1"),
		AnsiF2:  _tmuxKeyResponse("F2"),
		AnsiF3:  _tmuxKeyResponse("F3"),
		AnsiF4:  _tmuxKeyResponse("F4"),
		AnsiF5:  _tmuxKeyResponse("F5"),
		AnsiF6:  _tmuxKeyResponse("F6"),
		AnsiF7:  _tmuxKeyResponse("F7"),
		AnsiF8:  _tmuxKeyResponse("F8"),
		AnsiF9:  _tmuxKeyResponse("F9"),
		AnsiF10: _tmuxKeyResponse("F10"),
		AnsiF11: _tmuxKeyResponse("F11"),
		AnsiF12: _tmuxKeyResponse("F12"),
		// F13->F20 are not supported in tmux
	},
}

func getAnsiEscSeq(keySet types.KeyboardMode, keyPress KeyCode) []byte {
	b, ok := _ansiLookUpTable[keySet][keyPress]
	if !ok {
		if keySet != types.KeysNormal {
			return getAnsiEscSeq(types.KeysNormal, keyPress)
		}

		debug.Log(fmt.Sprintf("No sequence available for %d in %d", keyPress, keySet))
		return b
	}

	return b
}

// TODO:
// As a special case, the SS3 sent before F1 through F4 is altered to CSI when
// sending a function key modifier as a parameter.
func GetAnsiEscSeq(keySet types.KeyboardMode, keyPress KeyCode, modifier Modifier) []byte {
	// check for hardcoded exceptions
	keySeq := specialCaseSequences(keySet, keyPress, modifier)
	if len(keySeq) != 0 {
		return keySeq
	}

	// fallback to generalized formats
	lookupSet := _retryLookUpTable[keySet]
	for _, set := range *lookupSet {
		keySeq = getAnsiEscSeq(set, keyPress)
		if len(keySeq) == 0 {
			if keySet == types.KeysTmuxClient && keyPress < 255 {
				keySeq = octal.Escape([]byte{byte(keyPress)})
			} else {
				continue
			}
		}

		// no modifiers
		if modifier == 0 {
			return keySeq
		}

		// contains modifiers
		if keySet == types.KeysTmuxClient && !modifier.Is(MOD_ALT) {
			return append(translateModToTmuxCode(modifier), keySeq...)
		}

		ending := keySeq[len(keySeq)-1]
		seq := append(keySeq[:len(keySeq)-1], translateModToAnsiCode(modifier)...)
		return append(seq, ending)
	}

	return keySeq
}
