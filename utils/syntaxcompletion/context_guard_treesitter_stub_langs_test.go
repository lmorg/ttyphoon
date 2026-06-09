//go:build !cgo

package syntaxcompletion

import "testing"

func TestSupportedTreeSitterLanguages_NoCGO(t *testing.T) {
	langs := SupportedTreeSitterLanguages()
	if len(langs) != 0 {
		t.Fatalf("expected no tree-sitter languages in non-cgo build, got %v", langs)
	}
}
