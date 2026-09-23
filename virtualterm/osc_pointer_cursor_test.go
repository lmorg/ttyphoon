package virtualterm

import (
	"reflect"
	"testing"

	"github.com/lmorg/ttyphoon/window/backend/cursor"
)

type oscPointerTestRenderer struct {
	synchronizedOutputTestRenderer
}

func (renderer *oscPointerTestRenderer) SetBlinkState(bool) {}

func (renderer *oscPointerTestRenderer) RefreshNotes() {}

func TestOsc22AppliesValidatedPointerCursors(t *testing.T) {
	var emitted []string
	cursor.Register(func(css string) {
		emitted = append(emitted, css)
	})
	t.Cleanup(func() { cursor.Register(nil) })

	term := newParserTestTerm()
	term.renderer = &oscPointerTestRenderer{}
	term.SetFocus(true)
	emitted = nil
	pty := term.Pty.(*parserTestPty)

	stream := "\x1b]22;pointer\x1b\\\x1b]22;text\a\x1b]22;unsupported\x1b\\"
	pty.FeedInput([]byte(stream))
	drainMockPtyInput(t, term, len(stream)*2)

	if term._pointerCursorCSS != "default" {
		t.Fatalf("expected unsupported pointer shape to fall back to default, got %q", term._pointerCursorCSS)
	}
	if expected := []string{"pointer", "text", "default"}; !reflect.DeepEqual(emitted, expected) {
		t.Fatalf("unexpected OSC 22 cursor events: got %v, want %v", emitted, expected)
	}
}

func TestOsc22DefersBackgroundPaneCursorUntilFocus(t *testing.T) {
	var emitted []string
	cursor.Register(func(css string) {
		emitted = append(emitted, css)
	})
	t.Cleanup(func() { cursor.Register(nil) })

	term := newParserTestTerm()
	term.renderer = &oscPointerTestRenderer{}
	pty := term.Pty.(*parserTestPty)

	stream := "\x1b]22;text\x1b\\"
	pty.FeedInput([]byte(stream))
	drainMockPtyInput(t, term, len(stream)*2)

	if len(emitted) != 0 {
		t.Fatalf("expected background pane not to change the global cursor, got %v", emitted)
	}
	if term._pointerCursorCSS != "text" {
		t.Fatalf("expected background pane to retain its cursor request, got %q", term._pointerCursorCSS)
	}

	term.SetFocus(true)
	if expected := []string{"text"}; !reflect.DeepEqual(emitted, expected) {
		t.Fatalf("expected focus to apply the stored cursor, got %v", emitted)
	}
}
