package virtualterm

import "testing"

func TestWriteCellZeroWidthVariationSelectorDoesNotWrap(t *testing.T) {
	term := NewTestTerminal()
	term.setJumpScroll()

	term.writeCells("12345678A\uFE0EB")

	if term._curPos.Y != 0 {
		t.Fatalf("expected cursor Y to stay on first row, got %d", term._curPos.Y)
	}

	if got := (*term.screen)[0].String(); got != "12345678AB" {
		t.Fatalf("row content mismatch, got %q", got)
	}
}
