package subagent

import "testing"

func TestQuote(t *testing.T) {
	got := Quote("first line\nsecond line\n")
	want := "first line\n> second line\n> "
	if got != want {
		t.Fatalf("Quote() = %q, want %q", got, want)
	}
}
