package runewidth

import "testing"

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want int
	}{
		{name: "ascii", r: 'a', want: 1},
		{name: "combining mark", r: '\u0301', want: 0},
		{name: "cjk", r: '界', want: 2},
		{name: "emoji", r: '😀', want: 2},
		{name: "control", r: '\n', want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RuneWidth(tt.r); got != tt.want {
				t.Fatalf("RuneWidth(%q) = %d, want %d", tt.r, got, tt.want)
			}
		})
	}
}

func TestStringWidth(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{name: "ascii", s: "abc", want: 3},
		{name: "combining cluster", s: "e\u0301", want: 1},
		{name: "cjk", s: "界", want: 2},
		{name: "zwj emoji", s: "👨‍👩‍👧‍👦", want: 2},
		{name: "flag", s: "🇬🇧", want: 1},
		{name: "mixed", s: "a界😀", want: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringWidth(tt.s); got != tt.want {
				t.Fatalf("StringWidth(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		crop       int
		terminator string
		want       string
		wantWidth  int
	}{
		{name: "no truncation", s: "abc", crop: 3, terminator: "…", want: "abc", wantWidth: 3},
		{name: "ascii truncation", s: "abcdef", crop: 4, terminator: "…", want: "abc…", wantWidth: 4},
		{name: "emoji boundary", s: "ab😀cd", crop: 5, terminator: "…", want: "ab😀…", wantWidth: 5},
		{name: "terminator clipped", s: "abcdef", crop: 1, terminator: "..", want: "..", wantWidth: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.s, tt.crop, tt.terminator)
			if got != tt.want {
				t.Fatalf("Truncate(%q, %d, %q) = %q, want %q", tt.s, tt.crop, tt.terminator, got, tt.want)
			}
			if w := StringWidth(got); w != tt.wantWidth {
				t.Fatalf("StringWidth(Truncate(...)) = %d, want %d", w, tt.wantWidth)
			}
		})
	}
}
