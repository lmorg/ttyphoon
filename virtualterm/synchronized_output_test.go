package virtualterm

import (
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/types"
)

type synchronizedOutputTestRenderer struct {
	types.Renderer
	redraws int
	frames  int
}

func (renderer *synchronizedOutputTestRenderer) TriggerRedraw() {
	renderer.redraws++
}

func (renderer *synchronizedOutputTestRenderer) DrawFrame(types.Tile) {
	renderer.frames++
}

func (renderer *synchronizedOutputTestRenderer) PrintCell(types.Tile, *types.Cell, *types.XY) {}

func (renderer *synchronizedOutputTestRenderer) PrintRow(types.Tile, []*types.Cell, *types.XY) {}

func (renderer *synchronizedOutputTestRenderer) DrawRectWithColourAndBorder(types.Tile, *types.XY, *types.XY, *types.Colour, bool, bool) {
}

func TestSynchronizedOutputPrivateModeLifecycle(t *testing.T) {
	term := newParserTestTerm()
	renderer := &synchronizedOutputTestRenderer{}
	term.renderer = renderer
	pty := term.Pty.(*parserTestPty)

	begin := "\x1b[?2026h"
	pty.FeedInput([]byte(begin))
	drainMockPtyInput(t, term, len(begin)*2)

	if !term.synchronizedUpdateActive(time.Now()) {
		t.Fatal("expected DECSET 2026 to begin synchronized output")
	}
	if rendered := term.Render(); rendered {
		t.Fatal("expected rendering to be deferred during synchronized output")
	}
	if renderer.frames != 0 {
		t.Fatalf("expected no frame while synchronized, got %d", renderer.frames)
	}

	end := "\x1b[?2026l"
	pty.FeedInput([]byte(end))
	drainMockPtyInput(t, term, len(end)*2)

	if term.synchronizedUpdateActive(time.Now()) {
		t.Fatal("expected DECRST 2026 to end synchronized output")
	}
	if renderer.redraws != 1 {
		t.Fatalf("expected end to request one redraw, got %d", renderer.redraws)
	}
}

func TestSynchronizedOutputRefreshesDeadlineAndExpires(t *testing.T) {
	term := newParserTestTerm()
	start := time.Unix(100, 0)

	term.beginSynchronizedUpdate(start)
	firstDeadline := term._synchronizedUpdateDeadline
	term.beginSynchronizedUpdate(start.Add(500 * time.Millisecond))

	if !term._synchronizedUpdateDeadline.After(firstDeadline) {
		t.Fatal("expected repeated begin to refresh the synchronized-output deadline")
	}
	if !term.synchronizedUpdateActive(term._synchronizedUpdateDeadline.Add(-time.Nanosecond)) {
		t.Fatal("expected synchronized output to remain active before its deadline")
	}
	if term.synchronizedUpdateActive(term._synchronizedUpdateDeadline) {
		t.Fatal("expected synchronized output to expire at its deadline")
	}

	renderer := &synchronizedOutputTestRenderer{}
	term.renderer = renderer
	term._synchronizedUpdateDeadline = time.Now().Add(-time.Nanosecond)
	if rendered := term.Render(); !rendered {
		t.Fatal("expected rendering to resume after the synchronized-output timeout")
	}
	if renderer.frames != 1 {
		t.Fatalf("expected one frame after timeout, got %d", renderer.frames)
	}
}

func TestSynchronizedOutputIgnoresUnmatchedEnd(t *testing.T) {
	term := newParserTestTerm()
	renderer := &synchronizedOutputTestRenderer{}
	term.renderer = renderer

	term.endSynchronizedUpdate()

	if renderer.redraws != 0 {
		t.Fatalf("expected unmatched end not to request a redraw, got %d", renderer.redraws)
	}
}
