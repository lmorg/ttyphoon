package lsp

import (
	"testing"

	"github.com/lmorg/ttyphoon/utils/jupyter"
)

func TestResolverMapsExtensionAndAlias(t *testing.T) {
	resolver := NewResolverFromJupyter([]*jupyter.LanguageBindingT{
		{Aliases: []string{"go", "golang"}, FileExtension: "go"},
		{Aliases: []string{"javascript", "js"}, FileExtension: "js"},
	})

	if got := resolver.LanguageIDForFile("main.go"); got != "go" {
		t.Fatalf("LanguageIDForFile(main.go) = %q, want go", got)
	}

	if got := resolver.LanguageIDForName("golang"); got != "go" {
		t.Fatalf("LanguageIDForName(golang) = %q, want go", got)
	}

	if got := resolver.LanguageIDForName("js"); got != "javascript" {
		t.Fatalf("LanguageIDForName(js) = %q, want javascript", got)
	}
}

func TestResolverKeepsFirstMappingOnDuplicates(t *testing.T) {
	resolver := NewResolverFromJupyter([]*jupyter.LanguageBindingT{
		{Aliases: []string{"python", "py"}, FileExtension: "py"},
		{Aliases: []string{"pypy", "py"}, FileExtension: "py"},
	})

	if got := resolver.LanguageIDForName("py"); got != "python" {
		t.Fatalf("LanguageIDForName(py) = %q, want python", got)
	}

	if len(resolver.Warnings()) == 0 {
		t.Fatalf("expected duplicate warnings, got none")
	}
}

func TestResolverLanguageIDsForFileIncludesAllAliases(t *testing.T) {
	resolver := NewResolverFromJupyter([]*jupyter.LanguageBindingT{
		{Aliases: []string{"node", "js", "javascript"}, FileExtension: "js"},
		{Aliases: []string{"deno", "javascript"}, FileExtension: "js"},
	})

	got := resolver.LanguageIDsForFile("main.js")
	want := []string{"node", "js", "javascript", "deno"}
	if len(got) != len(want) {
		t.Fatalf("LanguageIDsForFile(main.js) len = %d, want %d (%v)", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("LanguageIDsForFile(main.js)[%d] = %q, want %q (all=%v)", i, got[i], want[i], got)
		}
	}
}
