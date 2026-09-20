package virtualterm

import (
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

func setResizeTestRowText(term *Term, row int, text string) {
	for x := range (*term.screen)[row].Cells {
		(*term.screen)[row].Cells[x].Clear()
	}

	for x, r := range text {
		if x >= len((*term.screen)[row].Cells) {
			break
		}
		(*term.screen)[row].Cells[x].Char = r
	}
}

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

func TestResize_ReflowsWrappedRowsWhenShrinkingWidth(t *testing.T) {
	term := NewTestTerminal()

	setResizeTestRowText(term, 0, "abcdefghij")
	setResizeTestRowText(term, 1, "klmn")
	(*term.screen)[1].RowMeta.Set(types.META_ROW_FROM_LINE_OVERFLOW)

	src := &types.RowSource{Host: "host", Pwd: "/tmp"}
	blk := &types.BlockMeta{Id: 99}
	(*term.screen)[0].Source = src
	(*term.screen)[0].Block = blk

	term.Resize(&types.XY{X: 8, Y: term.size.Y})

	if got := (*term.screen)[0].String(); got != "abcdefgh" {
		t.Fatalf("row 0 = %q, want %q", got, "abcdefgh")
	}

	if got := (*term.screen)[1].String(); got != "ijklmn" {
		t.Fatalf("row 1 = %q, want %q", got, "ijklmn")
	}

	if (*term.screen)[0].RowMeta.Is(types.META_ROW_FROM_LINE_OVERFLOW) {
		t.Fatal("row 0 should not be overflow")
	}

	if !(*term.screen)[1].RowMeta.Is(types.META_ROW_FROM_LINE_OVERFLOW) {
		t.Fatal("row 1 should be overflow")
	}

	if (*term.screen)[1].Source != src {
		t.Fatal("row 1 should inherit line source metadata")
	}

	if (*term.screen)[1].Block != blk {
		t.Fatal("row 1 should inherit line block metadata")
	}
}

func TestResize_ReflowsWrappedRowsWhenGrowingWidth(t *testing.T) {
	term := NewTestTerminal()

	setResizeTestRowText(term, 0, "abcdefghij")
	setResizeTestRowText(term, 1, "klmn")
	(*term.screen)[1].RowMeta.Set(types.META_ROW_FROM_LINE_OVERFLOW)

	term.Resize(&types.XY{X: 20, Y: term.size.Y})

	if got := (*term.screen)[0].String(); got != "abcdefghijklmn" {
		t.Fatalf("row 0 = %q, want %q", got, "abcdefghijklmn")
	}

	if got := (*term.screen)[1].String(); got != "" {
		t.Fatalf("row 1 = %q, want empty row", got)
	}

	if (*term.screen)[1].RowMeta.Is(types.META_ROW_FROM_LINE_OVERFLOW) {
		t.Fatal("row 1 should not be overflow after unwrap")
	}
}

func TestResize_ShrinkHeightConsumesTrailingBlankRows(t *testing.T) {
	term := NewTestTerminal()

	setResizeTestRowText(term, 0, "top")
	setResizeTestRowText(term, 1, "middle")

	term.Resize(&types.XY{X: term.size.X, Y: 3})

	if got := len(term._scrollBuf); got != 0 {
		t.Fatalf("scrollback rows = %d, want 0", got)
	}

	if got := (*term.screen)[0].String(); got != "top" {
		t.Fatalf("row 0 = %q, want %q", got, "top")
	}

	if got := (*term.screen)[1].String(); got != "middle" {
		t.Fatalf("row 1 = %q, want %q", got, "middle")
	}
}

func TestResize_ClampsInvalidDimensions(t *testing.T) {
	term := NewTestTerminal()

	term.Resize(&types.XY{X: 0, Y: 0})

	if term.size.X != 1 || term.size.Y != 1 {
		t.Fatalf("term size = (%d,%d), want (1,1)", term.size.X, term.size.Y)
	}
}
