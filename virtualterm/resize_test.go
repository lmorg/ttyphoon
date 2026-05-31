package virtualterm

import (
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

func TestResize_ReportsExactSizeToPty(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)

	nextSize := &types.XY{X: 132, Y: 42}
	term.Resize(nextSize)

	if len(pty.res) == 0 {
		t.Fatal("expected PTY resize to be called")
	}

	got := pty.res[len(pty.res)-1]
	if got == nil {
		t.Fatal("expected PTY resize size to be non-nil")
	}

	if got.X != nextSize.X || got.Y != nextSize.Y {
		t.Fatalf("unexpected PTY size: got (%d,%d), want (%d,%d)", got.X, got.Y, nextSize.X, nextSize.Y)
	}

	if term.GetSize().X != nextSize.X || term.GetSize().Y != nextSize.Y {
		t.Fatalf("unexpected terminal size: got (%d,%d), want (%d,%d)", term.GetSize().X, term.GetSize().Y, nextSize.X, nextSize.Y)
	}
}

func TestResizePty_ReportsCurrentTermSizeToPty(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)

	term.resizePty()

	if len(pty.res) != 1 {
		t.Fatalf("expected exactly one PTY resize call, got %d", len(pty.res))
	}

	got := pty.res[0]
	if got == nil {
		t.Fatal("expected PTY resize size to be non-nil")
	}

	want := term.GetSize()
	if got.X != want.X || got.Y != want.Y {
		t.Fatalf("unexpected PTY size: got (%d,%d), want (%d,%d)", got.X, got.Y, want.X, want.Y)
	}
}
