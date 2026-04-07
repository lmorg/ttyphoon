package virtualterm

import (
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

func setTestScreenRows(term *Term, rows ...string) {
	for y := range *term.screen {
		for x := range (*term.screen)[y].Cells {
			(*term.screen)[y].Cells[x].Clear()
		}
	}

	for y, row := range rows {
		if y >= len(*term.screen) {
			break
		}

		for x, r := range row {
			if x >= len((*term.screen)[y].Cells) {
				break
			}
			(*term.screen)[y].Cells[x].Char = r
		}
	}
}

func TestCopyRangeSameLine(t *testing.T) {
	term := NewTestTerminal()
	setTestScreenRows(term, "0123456789")

	forward := string(term.CopyRange(&types.XY{X: 2, Y: 0}, &types.XY{X: 6, Y: 0}))
	backward := string(term.CopyRange(&types.XY{X: 6, Y: 0}, &types.XY{X: 2, Y: 0}))

	if forward != "23456" {
		t.Fatalf("CopyRange forwards = %q, want %q", forward, "23456")
	}

	if backward != forward {
		t.Fatalf("CopyRange backwards = %q, want %q", backward, forward)
	}
}

func TestCopyRangeMultiline(t *testing.T) {
	term := NewTestTerminal()
	setTestScreenRows(term,
		"0123456789",
		"abcdefghij",
	)

	downward := string(term.CopyRange(&types.XY{X: 7, Y: 0}, &types.XY{X: 2, Y: 1}))
	upward := string(term.CopyRange(&types.XY{X: 2, Y: 1}, &types.XY{X: 7, Y: 0}))

	const expected = "789\nabc"
	if downward != expected {
		t.Fatalf("CopyRange downward = %q, want %q", downward, expected)
	}

	if upward != expected {
		t.Fatalf("CopyRange upward = %q, want %q", upward, expected)
	}
}

func TestCopySquare(t *testing.T) {
	term := NewTestTerminal()
	setTestScreenRows(term,
		"0123456789",
		"abcdefghij",
	)

	actual := string(term.CopySquare(&types.XY{X: 1, Y: 0}, &types.XY{X: 3, Y: 1}))
	const expected = "123\nbcd"

	if actual != expected {
		t.Fatalf("CopySquare = %q, want %q", actual, expected)
	}
}
