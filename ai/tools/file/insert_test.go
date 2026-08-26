package filetools

import (
	"strings"
	"testing"
)

func TestApplyLineInserts(t *testing.T) {
	tests := []struct {
		name    string
		content string
		inserts []insertT
		want    string
		wantErr string
	}{
		{
			name:    "insert after first line",
			content: "one\ntwo\n",
			inserts: []insertT{{Line: 1, Text: "one and a half"}},
			want:    "one\none and a half\ntwo\n",
		},
		{
			name:    "insert at start",
			content: "one\ntwo\n",
			inserts: []insertT{{Line: 0, Text: "zero"}},
			want:    "zero\none\ntwo\n",
		},
		{
			name:    "append to end",
			content: "one\ntwo\n",
			inserts: []insertT{{Line: 2, Text: "three"}},
			want:    "one\ntwo\nthree\n",
		},
		{
			name:    "multi line text",
			content: "one\ntwo\n",
			inserts: []insertT{{Line: 1, Text: "a\nb"}},
			want:    "one\na\nb\ntwo\n",
		},
		{
			name:    "text with trailing newline does not add blank line",
			content: "one\ntwo\n",
			inserts: []insertT{{Line: 1, Text: "a\n"}},
			want:    "one\na\ntwo\n",
		},
		{
			name:    "line numbers address original content",
			content: "one\ntwo\nthree\n",
			inserts: []insertT{{Line: 1, Text: "after one"}, {Line: 3, Text: "after three"}},
			want:    "one\nafter one\ntwo\nthree\nafter three\n",
		},
		{
			name:    "multiple inserts at same line keep declared order",
			content: "one\ntwo\n",
			inserts: []insertT{{Line: 1, Text: "first"}, {Line: 1, Text: "second"}},
			want:    "one\nfirst\nsecond\ntwo\n",
		},
		{
			name:    "file without trailing newline stays without one",
			content: "one\ntwo",
			inserts: []insertT{{Line: 2, Text: "three"}},
			want:    "one\ntwo\nthree",
		},
		{
			name:    "empty file",
			content: "",
			inserts: []insertT{{Line: 0, Text: "one"}},
			want:    "one",
		},
		{
			name:    "line out of range",
			content: "one\n",
			inserts: []insertT{{Line: 5, Text: "x"}},
			wantErr: "out of range",
		},
		{
			name:    "negative line",
			content: "one\n",
			inserts: []insertT{{Line: -1, Text: "x"}},
			wantErr: "out of range",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := applyLineInserts(test.content, test.inserts)

			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("applyLineInserts() error = nil, want error containing %q", test.wantErr)
				}
				if !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("applyLineInserts() error = %q, want error containing %q", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("applyLineInserts() error = %v", err)
			}
			if got != test.want {
				t.Errorf("applyLineInserts() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		content      string
		wantLines    []string
		wantTrailing bool
	}{
		{content: "", wantLines: nil, wantTrailing: false},
		{content: "one", wantLines: []string{"one"}, wantTrailing: false},
		{content: "one\n", wantLines: []string{"one"}, wantTrailing: true},
		{content: "one\ntwo\n", wantLines: []string{"one", "two"}, wantTrailing: true},
		{content: "one\n\n", wantLines: []string{"one", ""}, wantTrailing: true},
	}

	for _, test := range tests {
		lines, trailing := splitLines(test.content)

		if trailing != test.wantTrailing {
			t.Errorf("splitLines(%q) trailingNewline = %v, want %v", test.content, trailing, test.wantTrailing)
		}
		if len(lines) != len(test.wantLines) {
			t.Fatalf("splitLines(%q) = %q, want %q", test.content, lines, test.wantLines)
		}
		for i := range lines {
			if lines[i] != test.wantLines[i] {
				t.Fatalf("splitLines(%q) = %q, want %q", test.content, lines, test.wantLines)
			}
		}
	}
}
