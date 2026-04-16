package codes

import (
	"fmt"

	"github.com/lmorg/ttyphoon/types"
)

type MouseTrackingMode int

const (
	MouseTrackingOff MouseTrackingMode = iota
	MouseTrackingX10
	MouseTrackingButtonEvent
	MouseTrackingAnyEvent
)

type MouseEncodingMode int

const (
	MouseEncodingDefault MouseEncodingMode = iota
	MouseEncodingUTF8
	MouseEncodingSGR
	MouseEncodingURXVT
)

const (
	TmuxMouseDown1 = "MouseDown1"
	TmuxMouseDown2 = "MouseDown2"
	TmuxMouseDown3 = "MouseDown3"
	TmuxMouseUp1   = "MouseUp1"
	TmuxMouseUp2   = "MouseUp2"
	TmuxMouseUp3   = "MouseUp3"
	TmuxMouseDrag1 = "MouseDrag1"
	TmuxMouseDrag2 = "MouseDrag2"
	TmuxMouseDrag3 = "MouseDrag3"
	TmuxMouseMove  = "MouseMove"
	TmuxWheelUp    = "WheelUp"
	TmuxWheelDown  = "WheelDownPane"
)

type MouseEventType int

const (
	MouseEventPress MouseEventType = iota
	MouseEventRelease
	MouseEventDrag
	MouseEventMove
	MouseEventWheelUp
	MouseEventWheelDown
)

type MouseEvent struct {
	Type   MouseEventType
	Button types.MouseButtonT
	X      int32
	Y      int32
}

func GetMouseEscSeq(keySet types.KeyboardMode, event MouseEvent) []byte {
	/*if keySet == types.KeysTmuxClient {
		name := getTmuxMouseKeyName(event)
		if name == "" {
			return nil
		}
		return append([]byte{0}, []byte(name+" ")...)
	}*/

	code, ok := xtermMouseCode(event)
	if !ok {
		return nil
	}

	col := int(event.X) + 1
	row := int(event.Y) + 1

	switch event.Type {
	case MouseEventRelease:
		return []byte(fmt.Sprintf("\x1b[<%d;%d;%dm", code, col, row))
	default:
		return []byte(fmt.Sprintf("\x1b[<%d;%d;%dM", code, col, row))
	}
}

func xtermMouseCode(event MouseEvent) (int, bool) {
	switch event.Type {
	case MouseEventPress:
		switch event.Button {
		case types.MOUSE_BUTTON_LEFT:
			return 0, true
		case types.MOUSE_BUTTON_MIDDLE:
			return 1, true
		case types.MOUSE_BUTTON_RIGHT:
			return 2, true
		default:
			return 0, false
		}

	case MouseEventRelease:
		return 3, true

	case MouseEventDrag:
		switch event.Button {
		case types.MOUSE_BUTTON_LEFT:
			return 32, true
		case types.MOUSE_BUTTON_MIDDLE:
			return 33, true
		case types.MOUSE_BUTTON_RIGHT:
			return 34, true
		default:
			return 35, true
		}

	case MouseEventMove:
		return 35, true

	case MouseEventWheelUp:
		return 64, true

	case MouseEventWheelDown:
		return 65, true
	}

	return 0, false
}

/*func getTmuxMouseKeyName(event MouseEvent) string {
	switch event.Type {
	case MouseEventPress:
		switch event.Button {
		case types.MOUSE_BUTTON_LEFT:
			return TmuxMouseDown1Pane
		case types.MOUSE_BUTTON_MIDDLE:
			return TmuxMouseDown2Pane
		case types.MOUSE_BUTTON_RIGHT:
			return TmuxMouseDown3Pane
		}

	case MouseEventRelease:
		switch event.Button {
		case types.MOUSE_BUTTON_LEFT:
			return TmuxMouseUp1Pane
		case types.MOUSE_BUTTON_MIDDLE:
			return TmuxMouseUp2Pane
		case types.MOUSE_BUTTON_RIGHT:
			return TmuxMouseUp3Pane
		}

	case MouseEventDrag:
		switch event.Button {
		case types.MOUSE_BUTTON_LEFT:
			return TmuxMouseDrag1Pane
		case types.MOUSE_BUTTON_MIDDLE:
			return TmuxMouseDrag2Pane
		case types.MOUSE_BUTTON_RIGHT:
			return TmuxMouseDrag3Pane
		}

	case MouseEventMove:
		return TmuxMouseMovePane

	case MouseEventWheelUp:
		return TmuxWheelUpPane

	case MouseEventWheelDown:
		return TmuxWheelDownPane
	}

	return ""
}*/
