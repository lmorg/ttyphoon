package virtualterm

import (
	"testing"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/types"
)

type mouseLeaveTestElement struct {
	types.Element
	mouseOutCalls int
}

func (element *mouseLeaveTestElement) MouseOut() {
	element.mouseOutCalls++
}

func TestMouseLeaveClearsHoverWithoutSendingInput(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)
	element := &mouseLeaveTestElement{}
	term._mouseIn = element
	hoverCalls := 0
	term._mousePosRenderer.Set(func() { hoverCalls++ })
	term.MouseLeave()
	term.MouseLeave()
	if term._mousePosRenderer.Call() || hoverCalls != 0 {
		t.Fatal("mouse leave retained the hover drawing callback")
	}
	if term._mouseIn != nil || element.mouseOutCalls != 1 {
		t.Fatalf("hovered element = %v, MouseOut calls = %d, want nil and 1", term._mouseIn, element.mouseOutCalls)
	}
	if len(pty.out) != 0 {
		t.Fatalf("mouse leave wrote terminal input: %q", pty.out)
	}
}

func TestMouseClickSendsPressAndReleaseWithPosition(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)
	term._mouseTracking = codes.MouseTrackingButtonEvent

	pos := &types.XY{X: 2, Y: 3}
	term.MouseClick(pos, types.MOUSE_BUTTON_LEFT, 1, types.BUTTON_PRESSED, func() {})
	term.MouseClick(pos, types.MOUSE_BUTTON_LEFT, 1, types.BUTTON_RELEASED, func() {})

	want := "\x1b[<0;3;4M\x1b[<3;3;4m"
	got := string(pty.out)
	if got != want {
		t.Fatalf("unexpected PTY output for click at (%d,%d): got %q want %q", pos.X, pos.Y, got, want)
	}
}

func TestMouseWheelSendsWheelCodeWithPosition(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)
	term._mouseTracking = codes.MouseTrackingButtonEvent

	pos := &types.XY{X: 4, Y: 1}
	term.MouseWheel(pos, &types.XY{X: 0, Y: 1})

	want := "\x1b[<64;5;2M"
	got := string(pty.out)
	if got != want {
		t.Fatalf("unexpected PTY output for wheel at (%d,%d): got %q want %q", pos.X, pos.Y, got, want)
	}
}

func TestMouseDragSendsDragCodeWithPosition(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)
	term._mouseTracking = codes.MouseTrackingButtonEvent
	term._mouseButtonDown = true
	term._mouseButton = types.MOUSE_BUTTON_LEFT

	pos := &types.XY{X: 7, Y: 0}
	term.MouseMotion(pos, &types.XY{X: 1, Y: 0}, func() {})

	want := "\x1b[<32;8;1M"
	got := string(pty.out)
	if got != want {
		t.Fatalf("unexpected PTY output for drag at (%d,%d): got %q want %q", pos.X, pos.Y, got, want)
	}
}

func TestMouseMotionAnyEventSendsMoveWhenNoButtonDown(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)
	term._mouseTracking = codes.MouseTrackingAnyEvent
	term._mouseButtonDown = false

	pos := &types.XY{X: 1, Y: 1}
	term.MouseMotion(pos, &types.XY{X: 1, Y: 1}, func() {})

	want := "\x1b[<35;2;2M"
	got := string(pty.out)
	if got != want {
		t.Fatalf("unexpected PTY output for move at (%d,%d): got %q want %q", pos.X, pos.Y, got, want)
	}
}
