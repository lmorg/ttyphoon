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

func TestResolverPathRegexpOverridesExtension(t *testing.T) {
	resolver := NewResolverFromJupyter([]*jupyter.LanguageBindingT{
		{Aliases: []string{"yaml"}, FileExtension: "yaml"},
		{Aliases: []string{"kubernetes"}, FileExtension: "yaml", PathRegexp: `k8s/.*\.yaml$`},
	})

	if got := resolver.LanguageIDForFile("manifests/deployment.yaml"); got != "yaml" {
		t.Fatalf("LanguageIDForFile(manifests/deployment.yaml) = %q, want yaml", got)
	}

	if got := resolver.LanguageIDForFile("k8s/deployment.yaml"); got != "kubernetes" {
		t.Fatalf("LanguageIDForFile(k8s/deployment.yaml) = %q, want kubernetes", got)
	}
}

func TestResolverPathRegexpLanguageIDsForFile(t *testing.T) {
	resolver := NewResolverFromJupyter([]*jupyter.LanguageBindingT{
		{Aliases: []string{"yaml"}, FileExtension: "yaml"},
		{Aliases: []string{"kubernetes", "k8s"}, FileExtension: "yaml", PathRegexp: `k8s/.*\.yaml$`},
	})

	got := resolver.LanguageIDsForFile("k8s/service.yaml")
	want := []string{"kubernetes", "k8s"}
	if len(got) != len(want) {
		t.Fatalf("LanguageIDsForFile(k8s/service.yaml) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("LanguageIDsForFile(k8s/service.yaml)[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestResolverInvalidPathRegexpWarning(t *testing.T) {
	resolver := NewResolverFromJupyter([]*jupyter.LanguageBindingT{
		{Aliases: []string{"go"}, FileExtension: "go", PathRegexp: `[invalid`},
	})

	if len(resolver.Warnings()) == 0 {
		t.Fatal("expected warning for invalid path regexp, got none")
	}
}

func TestResolverPathRegexpSkipsAliasAndExtension(t *testing.T) {
	resolver := NewResolverFromJupyter([]*jupyter.LanguageBindingT{
		{Aliases: []string{"yaml"}, FileExtension: "yaml"},
		{Aliases: []string{"kubernetes", "k8s"}, FileExtension: "yaml", PathRegexp: `k8s/.*\.yaml$`},
	})

	// alias "kubernetes" must not resolve — it belongs to a regexp-only binding
	if got := resolver.LanguageIDForName("kubernetes"); got != "" {
		t.Fatalf("LanguageIDForName(kubernetes) = %q, want empty (regexp-only binding)", got)
	}

	// extension ".yaml" must not pull in "kubernetes" candidates
	ids := resolver.LanguageIDsForFile("other/deployment.yaml")
	for _, id := range ids {
		if id == "kubernetes" || id == "k8s" {
			t.Fatalf("LanguageIDsForFile(other/deployment.yaml) contains %q, but regexp binding should be excluded from extension lookup", id)
		}
	}
}
