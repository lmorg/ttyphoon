package virtualterm

import (
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

func TestParseOSC7URI(t *testing.T) {
	tests := []struct {
		name     string
		params   []string
		wantHost string
		wantPath string
		wantOK   bool
	}{
		{
			name:     "decodes escaped path bytes",
			params:   []string{"file://localhost/tmp/my%20folder/caf%C3%A9"},
			wantHost: "localhost",
			wantPath: "/tmp/my folder/café",
			wantOK:   true,
		},
		{
			name:     "preserves plus in path",
			params:   []string{"file:///tmp/a+b"},
			wantPath: "/tmp/a+b",
			wantOK:   true,
		},
		{
			name:     "rejoins semicolon-delimited OSC parameters",
			params:   []string{"file:///tmp/a", "b"},
			wantPath: "/tmp/a;b",
			wantOK:   true,
		},
		{
			name:   "rejects empty payload",
			params: nil,
		},
		{
			name:   "rejects other schemes",
			params: []string{"https://example.com/tmp"},
		},
		{
			name:   "rejects relative file URI",
			params: []string{"file:relative/path"},
		},
		{
			name:   "rejects invalid percent escape",
			params: []string{"file:///tmp/%zz"},
		},
		{
			name:   "rejects query",
			params: []string{"file:///tmp/path?query"},
		},
		{
			name:   "rejects fragment",
			params: []string{"file:///tmp/path#fragment"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host, path, ok := parseOSC7URI(test.params)
			if host != test.wantHost || path != test.wantPath || ok != test.wantOK {
				t.Fatalf("parseOSC7URI(%q) = (%q, %q, %t), want (%q, %q, %t)",
					test.params, host, path, ok, test.wantHost, test.wantPath, test.wantOK)
			}
		})
	}
}

func TestOSC133UpdatesTerminalState(t *testing.T) {
	term := &Term{}
	if got := term.State(); got != types.TermStateUnknown {
		t.Fatalf("initial State() = %d, want %d", got, types.TermStateUnknown)
	}

	tests := []struct {
		marker string
		state  types.TermStateT
	}{
		{marker: "A", state: types.TermStatePromptStart},
		{marker: "B", state: types.TermStatePromptEnd},
		{marker: "C", state: types.TermStateCommandOutputStart},
		{marker: "D", state: types.TermStateCommandFinished},
	}

	for _, test := range tests {
		t.Run(test.marker, func(t *testing.T) {
			params := []string{test.marker}
			if test.marker == "D" {
				params = append(params, "0")
			}
			term.osc133UpdateState(params)
			if got := term.State(); got != test.state {
				t.Fatalf("State() after OSC 133;%s = %d, want %d", test.marker, got, test.state)
			}
		})
	}

	term.osc133UpdateState(nil)
	term.osc133UpdateState([]string{"E"})
	if got := term.State(); got != types.TermStateCommandFinished {
		t.Fatalf("State() after invalid marker = %d, want unchanged %d", got, types.TermStateCommandFinished)
	}
}
