package virtualterm

import (
	"testing"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/types"
)

func TestCsiEscapeCancelsAndStartsReplacementSequence(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)
	stream := "A\x1b[38;2;198;208;245\x1b[31mB"

	pty.FeedInput([]byte(stream))
	drainMockPtyInput(t, term, len(stream)*2)

	assertRecoveryText(t, term, 'A', 'B')
	assertCellForeground(t, term, 1, types.SGR_COLOR_RED)
}

func TestPrivateCsiEscapeCancelsAndStartsReplacementSequence(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)
	stream := "A\x1b[?2026\x1b[32mB"

	pty.FeedInput([]byte(stream))
	drainMockPtyInput(t, term, len(stream)*2)

	assertRecoveryText(t, term, 'A', 'B')
	assertCellForeground(t, term, 1, types.SGR_COLOR_GREEN)
	if !term._synchronizedUpdateDeadline.IsZero() {
		t.Fatal("expected interrupted private mode not to begin synchronized output")
	}
}

func TestRepeatedEscapeRestartsEscapeParsing(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)
	stream := "A\x1b\x1b[31mB"

	pty.FeedInput([]byte(stream))
	drainMockPtyInput(t, term, len(stream)*2)

	assertRecoveryText(t, term, 'A', 'B')
	assertCellForeground(t, term, 1, types.SGR_COLOR_RED)
}

func TestCsiCancellationControlsReturnToGround(t *testing.T) {
	for _, prefix := range []string{"A\x1b[38;2;198", "A\x1b[?2026"} {
		for _, cancel := range []byte{codes.AsciiCtrlX, codes.AsciiCtrlZ} {
			t.Run(prefix+string([]byte{cancel}), func(t *testing.T) {
				term := newParserTestTerm()
				pty := term.Pty.(*parserTestPty)
				stream := append([]byte(prefix), cancel)
				stream = append(stream, 'B')

				pty.FeedInput(stream)
				drainMockPtyInput(t, term, len(stream)*2)

				assertRecoveryText(t, term, 'A', 'B')
				if !term._synchronizedUpdateDeadline.IsZero() {
					t.Fatal("expected cancellation not to begin synchronized output")
				}
			})
		}
	}
}

func assertRecoveryText(t *testing.T, term *Term, expected ...rune) {
	t.Helper()
	for i, want := range expected {
		if got := (*term.screen)[0].Cells[i].Char; got != want {
			t.Fatalf("unexpected character at column %d: got %q, want %q", i, got, want)
		}
	}
	if got := (*term.screen)[0].Cells[len(expected)].Char; got != 0 {
		t.Fatalf("escape bytes leaked into the screen at column %d: %q", len(expected), got)
	}
}

func assertCellForeground(t *testing.T, term *Term, column int, want *types.Colour) {
	t.Helper()
	got := (*term.screen)[0].Cells[column].Sgr.Fg
	if got.Red != want.Red || got.Green != want.Green || got.Blue != want.Blue {
		t.Fatalf("unexpected foreground at column %d: got (%d,%d,%d), want (%d,%d,%d)",
			column, got.Red, got.Green, got.Blue, want.Red, want.Green, want.Blue)
	}
}
