package find

import (
	"testing"
)

// ----------------------------------------------------------------------------
// MatchString tests
// ----------------------------------------------------------------------------

var matchStringTests = []struct {
	name    string
	pattern string
	input   string
	want    bool
}{
	// AND (default) ─ all tokens must appear
	{name: "and/single token match", pattern: "foo", input: "foobar", want: true},
	{name: "and/single token no match", pattern: "foo", input: "barbaz", want: false},
	{name: "and/multi token both match", pattern: "foo bar", input: "foobar", want: true},
	{name: "and/multi token one missing", pattern: "foo baz", input: "foobar", want: false},
	{name: "and/case insensitive", pattern: "FOO", input: "foobar", want: true},
	{name: "and/empty pattern matches all", pattern: "", input: "anything", want: true},

	// OR ─ any token must appear
	{name: "or/first token matches", pattern: "or foo baz", input: "foobar", want: true},
	{name: "or/second token matches", pattern: "or foo baz", input: "bazqux", want: true},
	{name: "or/no token matches", pattern: "or foo baz", input: "quux", want: false},
	{name: "or/case insensitive", pattern: "or FOO", input: "foobar", want: true},

	// NOT ─ none of the tokens may appear
	{name: "not/excluded word present", pattern: "! foo", input: "foobar", want: false},
	{name: "not/excluded word absent", pattern: "! foo", input: "barbaz", want: true},
	{name: "not/multiple tokens one present", pattern: "! foo baz", input: "bazqux", want: false},
	{name: "not/multiple tokens none present", pattern: "! foo baz", input: "quux", want: true},
	{name: "not/case insensitive", pattern: "! FOO", input: "foobar", want: false},

	// Regexp ─ rx prefix
	{name: "rx/matches", pattern: `rx foo\d+`, input: "foo123", want: true},
	{name: "rx/no match", pattern: `rx foo\d+`, input: "fooabc", want: false},
	{name: "rx/case insensitive flag added", pattern: "rx FOO", input: "foobar", want: true},
	{name: "rx/anchored", pattern: "rx ^foo$", input: "foo", want: true},
	{name: "rx/anchored no match", pattern: "rx ^foo$", input: "foobar", want: false},

	// Glob ─ g prefix
	{name: "glob/star matches", pattern: "g *.go", input: "main.go", want: true},
	{name: "glob/star no match", pattern: "g *.go", input: "main.js", want: false},
	{name: "glob/question mark", pattern: "g foo?ar", input: "foobar", want: true},
	{name: "glob/case insensitive", pattern: "g *.GO", input: "main.go", want: true},
	{name: "glob/matches basename in subdir", pattern: "g *.go", input: "utils/find/find.go", want: true},

	// Auto-glob (pattern contains *)
	{name: "autoglob/matches", pattern: "*.go", input: "main.go", want: true},
	{name: "autoglob/no match", pattern: "*.go", input: "main.js", want: false},
}

func TestMatchString(t *testing.T) {
	for _, tt := range matchStringTests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := New(tt.pattern)
			if err != nil {
				t.Fatalf("New(%q) error: %v", tt.pattern, err)
			}
			if got := f.MatchString(tt.input); got != tt.want {
				t.Errorf("MatchString(%q) = %v, want %v (pattern %q)", tt.input, got, tt.want, tt.pattern)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// Filter tests
// ----------------------------------------------------------------------------

func TestFilter(t *testing.T) {
	files := []string{
		"README.md",
		"main.go",
		"main_test.go",
		"frontend/src/notes.js",
		"utils/find/find.go",
	}

	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{
			name:    "single token",
			pattern: "main",
			want:    []string{"main.go", "main_test.go"},
		},
		{
			name:    "multi token AND",
			pattern: "main go",
			want:    []string{"main.go", "main_test.go"},
		},
		{
			name:    "multi token AND excludes partial",
			pattern: "main test",
			want:    []string{"main_test.go"},
		},
		{
			name:    "or mode",
			pattern: "or readme js",
			want:    []string{"README.md", "frontend/src/notes.js"},
		},
		{
			name:    "not mode",
			pattern: "! go",
			want:    []string{"README.md", "frontend/src/notes.js"},
		},
		{
			name:    "regexp mode",
			pattern: `rx \.go$`,
			want:    []string{"main.go", "main_test.go", "utils/find/find.go"},
		},
		{
			name:    "glob mode",
			pattern: "g *.go",
			want:    []string{"main.go", "main_test.go", "utils/find/find.go"},
		},
		{
			name:    "autoglob",
			pattern: "*.md",
			want:    []string{"README.md"},
		},
		{
			name:    "empty returns all",
			pattern: "",
			want:    files,
		},
		{
			name:    "no matches",
			pattern: "zzznomatch",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := New(tt.pattern)
			if err != nil {
				t.Fatalf("New(%q) error: %v", tt.pattern, err)
			}
			got := f.Filter(files)
			if len(got) != len(tt.want) {
				t.Fatalf("Filter() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Filter()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ----------------------------------------------------------------------------
// Error handling
// ----------------------------------------------------------------------------

func TestNewInvalidRegexp(t *testing.T) {
	_, err := New(`rx (unclosed`)
	if err == nil {
		t.Error("expected error for invalid regexp, got nil")
	}
}
