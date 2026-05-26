package lsp

import (
	"testing"
)

// ---------------------------------------------------------------------------
// ValidateConfig
// ---------------------------------------------------------------------------

func TestValidateConfig_valid(t *testing.T) {
	cfg := map[string][]string{
		"go":         {"gopls"},
		"javascript": {"typescript-language-server", "--stdio"},
		"python":     {"/usr/local/bin/pyright-langserver", "--stdio"},
	}

	if errs := ValidateConfig(cfg); len(errs) != 0 {
		t.Fatalf("unexpected validation errors: %v", errs)
	}
}

func TestValidateConfig_emptyArgv(t *testing.T) {
	cfg := map[string][]string{
		"go": {},
	}

	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected error for empty argv, got none")
	}
}

func TestValidateConfig_blankExecutable(t *testing.T) {
	cfg := map[string][]string{
		"go": {"   ", "--stdio"},
	}

	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected error for blank executable, got none")
	}
}

func TestValidateConfig_empty(t *testing.T) {
	if errs := ValidateConfig(nil); len(errs) != 0 {
		t.Fatalf("nil config should be valid, got: %v", errs)
	}
}

// ---------------------------------------------------------------------------
// LookupArgv
// ---------------------------------------------------------------------------

func TestLookupArgv_found(t *testing.T) {
	cfg := map[string][]string{
		"go": {"gopls", "serve"},
	}

	argv := LookupArgv(cfg, "go")
	if len(argv) != 2 || argv[0] != "gopls" || argv[1] != "serve" {
		t.Errorf("LookupArgv(go) = %v, want [gopls serve]", argv)
	}
}

func TestLookupArgv_missing(t *testing.T) {
	cfg := map[string][]string{
		"go": {"gopls"},
	}

	if got := LookupArgv(cfg, "rust"); got != nil {
		t.Errorf("LookupArgv(rust) = %v, want nil", got)
	}
}

func TestLookupArgv_returnIsACopy(t *testing.T) {
	cfg := map[string][]string{
		"go": {"gopls"},
	}

	a := LookupArgv(cfg, "go")
	b := LookupArgv(cfg, "go")
	a[0] = "mutated"

	if b[0] != "gopls" {
		t.Error("LookupArgv should return a copy, but mutation affected original")
	}
}

// ---------------------------------------------------------------------------
// Manager
// ---------------------------------------------------------------------------

func TestNewManager_notNil(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestManager_Stop_unknownKey(t *testing.T) {
	m := NewManager()
	// Should not panic.
	m.Stop("root", "go")
}

func TestManager_StopAll_empty(t *testing.T) {
	m := NewManager()
	// Should not panic.
	m.StopAll()
}
