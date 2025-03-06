package types

type MouseButtonT int

const (
	MOUSE_BUTTON_LEFT MouseButtonT = 1 + iota
	MOUSE_BUTTON_MIDDLE
	MOUSE_BUTTON_RIGHT
	MOUSE_BUTTON_X1
	MOUSE_BUTTON_X2
)

type ButtonStateT int

const (
	BUTTON_PRESSED  ButtonStateT = 1
	BUTTON_RELEASED ButtonStateT = 0
)
