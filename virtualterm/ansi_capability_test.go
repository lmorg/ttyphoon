package virtualterm

import (
	"testing"
	"time"
)

func TestPrivateModeQueriesReportSynchronizedAndUnsupportedModes(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)
	stream := "\x1b[?2026$p\x1b[?2048$p"

	pty.FeedInput([]byte(stream))
	drainMockPtyInput(t, term, len(stream)*2)

	want := "\x1b[?2026;2$y\x1b[?2048;0$y"
	if got := string(pty.out); got != want {
		t.Fatalf("unexpected private-mode reports: got %q, want %q", got, want)
	}
}

func TestPrivateModeQueryReportsActiveSynchronizedOutput(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)
	term.beginSynchronizedUpdate(time.Now())
	stream := "\x1b[?2026$p"

	pty.FeedInput([]byte(stream))
	drainMockPtyInput(t, term, len(stream)*2)

	want := "\x1b[?2026;1$y"
	if got := string(pty.out); got != want {
		t.Fatalf("unexpected active synchronized-output report: got %q, want %q", got, want)
	}
}

func TestUnsupportedKittyKeyboardNegotiationStaysInLegacyMode(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)
	stream := "A\x1b[?u\x1b[=25;1u\x1b[>25u\x1b[<uB"

	pty.FeedInput([]byte(stream))
	drainMockPtyInput(t, term, len(stream)*2)

	assertRecoveryText(t, term, 'A', 'B')
	if len(pty.out) != 0 {
		t.Fatalf("expected unsupported kitty keyboard negotiation not to advertise support, got %q", pty.out)
	}
}
