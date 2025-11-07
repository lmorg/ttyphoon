package codes

type KeyCode int

const (
	// 0 -> 255 is ascii
	AnsiUp KeyCode = 1000 + iota
	AnsiDown
	AnsiRight
	AnsiLeft
	AnsiInsert
	AnsiHome
	AnsiEnd
	AnsiDelete
	AnsiPageUp
	AnsiPageDown

	AnsiKeyPadSpace
	AnsiKeyPadTab
	AnsiKeyPadEnter
	AnsiKeyPadMultiply
	AnsiKeyPadAdd
	AnsiKeyPadComma
	AnsiKeyPadMinus
	AnsiKeyPadPeriod
	AnsiKeyPadDivide
	AnsiKeyPad0
	AnsiKeyPad1
	AnsiKeyPad2
	AnsiKeyPad3
	AnsiKeyPad4
	AnsiKeyPad5
	AnsiKeyPad6
	AnsiKeyPad7
	AnsiKeyPad8
	AnsiKeyPad9
	AnsiKeyPadEqual

	AnsiF1
	AnsiF2
	AnsiF3
	AnsiF4
	AnsiF5
	AnsiF6
	AnsiF7
	AnsiF8
	AnsiF9
	AnsiF10
	AnsiF11
	AnsiF12
	AnsiF13
	AnsiF14
	AnsiF15
	AnsiF16
	AnsiF17
	AnsiF18
	AnsiF19
	AnsiF20
)
