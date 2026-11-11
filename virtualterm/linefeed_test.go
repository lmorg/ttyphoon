package virtualterm

import (
	"testing"
)

func TestLineFeedRedrawsAtJumpScrollFrequency(t *testing.T) {
	term := newParserTestTerm()
	renderer := &synchronizedOutputTestRenderer{}
	term.renderer = renderer
	term._ssFrequency = 2

	for _, want := range []int32{0, 1, 1, 2} {
		term._mutex.Lock()
		term.lineFeed(_LINEFEED_CURSOR_MOVED)
		term._mutex.Unlock()

		if got := renderer.redraws.Load(); got != want {
			t.Fatalf("redraw count after line feed = %d, want %d", got, want)
		}
	}
}

func TestLineFeedRedrawCadenceChangesWithScrollMode(t *testing.T) {
	term := newParserTestTerm()
	renderer := &synchronizedOutputTestRenderer{}
	term.renderer = renderer

	term._mutex.Lock()
	defer term._mutex.Unlock()

	pty := term.Pty.(*parserTestPty)
	smoothScroll := "\x1b[?4h"
	pty.FeedInput([]byte(smoothScroll))
	drainMockPtyInput(t, term, len(smoothScroll)*2)
	if term._ssFrequency != 0 {
		t.Fatalf("smooth-scroll frequency = %d, want 0", term._ssFrequency)
	}

	term.lineFeed(_LINEFEED_CURSOR_MOVED)
	if got := renderer.redraws.Load(); got != 1 {
		t.Fatalf("redraw count in smooth-scroll mode = %d, want 1", got)
	}

	jumpScroll := "\x1b[?4l"
	pty.FeedInput([]byte(jumpScroll))
	drainMockPtyInput(t, term, len(jumpScroll)*2)
	frequency := term._ssFrequency
	if frequency <= 0 {
		t.Fatalf("jump-scroll frequency = %d, want positive", frequency)
	}

	for i := int32(1); i <= frequency; i++ {
		term.lineFeed(_LINEFEED_CURSOR_MOVED)
		want := int32(1)
		if i == frequency {
			want = 2
		}
		if got := renderer.redraws.Load(); got != want {
			t.Fatalf("redraw count after %d jump-scroll line feeds = %d, want %d", i, got, want)
		}
	}
}
