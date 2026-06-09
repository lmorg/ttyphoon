//go:build cgo

package syntaxcompletion

import "testing"

func TestSupportedTreeSitterLanguages_CGO(t *testing.T) {
	langs := SupportedTreeSitterLanguages()
	if len(langs) == 0 {
		t.Fatalf("expected compiled tree-sitter languages in cgo build")
	}

	want := map[string]bool{
		"bash":       true,
		"c":          true,
		"cpp":        true,
		"csharp":     true,
		"css":        true,
		"go":         true,
		"html":       true,
		"java":       true,
		"javascript": true,
		"json":       true,
		"julia":      true,
		"ocaml":      true,
		"php":        true,
		"python":     true,
		"ruby":       true,
		"rust":       true,
		"scala":      true,
		"typescript": true,
		"verilog":    true,
	}
	for _, lang := range langs {
		delete(want, lang)
	}
	if len(want) != 0 {
		t.Fatalf("missing expected languages: %v", want)
	}
}

func TestEngineSupportedTreeSitterLanguages_CGO(t *testing.T) {
	eng := mustDefaultEngine(t)
	langs := eng.SupportedTreeSitterLanguages()
	if len(langs) == 0 {
		t.Fatalf("expected engine to report compiled tree-sitter languages")
	}
}
