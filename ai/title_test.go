package ai

import "testing"

func TestSanitizeAINoteTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "basic words", input: "Fix Kubernetes CrashLoopBackOff", want: "fix-kubernetes-crashloopbackoff"},
		{name: "strip punctuation", input: "Why does this fail?", want: "why-does-this-fail"},
		{name: "collapse separators", input: "  API   auth / token refresh  ", want: "api-auth-token-refresh"},
		{name: "empty after sanitize", input: "!!!", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeAINoteTitle(tt.input)
			if got != tt.want {
				t.Fatalf("sanitizeAINoteTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}