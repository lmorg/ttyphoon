package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"gopkg.in/yaml.v3"
)

func TestAIEnvironmentFilesDecodeFromYAML(t *testing.T) {
	var settings AiT
	if err := yaml.Unmarshal([]byte("EnvFiles: [global.vars]\nServices:\n  example:\n    EnvFiles: [service.vars]\n"), &settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.EnvFiles) != 1 || settings.EnvFiles[0] != "global.vars" ||
		len(settings.ServicesYaml["example"].EnvFiles) != 1 || settings.ServicesYaml["example"].EnvFiles[0] != "service.vars" {
		t.Fatal("AI environment files did not decode")
	}
}

func TestReloadClearsRemovedAIEnvironmentFiles(t *testing.T) {
	previousConfig, previousCache, previousDirs := Config, langCache, xdg.ConfigDirs
	t.Cleanup(func() {
		Config, langCache, xdg.ConfigDirs = previousConfig, previousCache, previousDirs
	})
	Config, langCache = configT{}, LanguagesT{}
	xdg.ConfigDirs = []string{t.TempDir()}
	dir := filepath.Join(xdg.ConfigDirs[0], app.DirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "test.yaml")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("Social:\n  S3:\n    EnvFiles: [s3.vars]\n    Bucket: test\nAI:\n  EnvFiles: [global.vars]\n  Services:\n    test-service:\n      Label: test-service\n      EnvFiles: [service.vars]\n      Env:\n        TEST_KEY: old-value\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if len(Config.Ai.EnvFiles) != 1 || len(Config.Ai.Services()["test-service"].EnvFiles) != 1 {
		t.Fatal("initial AI environment files were not loaded")
	}
	if Config.Social.S3 == nil {
		t.Fatal("initial S3 configuration was not loaded")
	}
	write("AI:\n  Services:\n    test-service:\n      Label: test-service\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if len(Config.Ai.EnvFiles) != 0 || len(Config.Ai.Services()["test-service"].EnvFiles) != 0 || len(Config.Ai.Services()["test-service"].Env) != 0 {
		t.Fatal("reload retained removed AI environment settings")
	}
	if Config.Social.S3 != nil {
		t.Fatal("reload retained a removed S3 configuration")
	}
}

func TestReloadLSPEnvironmentFiles(t *testing.T) {
	previousConfig, previousCache, previousDirs := Config, langCache, xdg.ConfigDirs
	t.Cleanup(func() { Config, langCache, xdg.ConfigDirs = previousConfig, previousCache, previousDirs })
	Config, langCache = configT{}, LanguagesT{}
	xdg.ConfigDirs = []string{t.TempDir()}
	dir := filepath.Join(xdg.ConfigDirs[0], app.DirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "test.yaml")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("Notes:\n  Languages:\n    go:\n      LSP:\n        command: [gopls]\n        EnvFiles: [old.vars]\n  SpellCheck:\n    TyposLsp:\n      EnvFiles: [typos.vars]\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if len(Config.Notes.Languages["go"].LSP.EnvFiles) != 1 || len(Config.Notes.SpellCheck.TyposLsp.EnvFiles) != 1 {
		t.Fatal("initial LSP environment files not loaded")
	}
	write("Notes: [")
	if err := LoadConfig(); err == nil {
		t.Fatal("invalid configuration was accepted")
	}
	if len(Config.Notes.Languages["go"].LSP.EnvFiles) != 1 || len(Config.Notes.SpellCheck.TyposLsp.EnvFiles) != 1 {
		t.Fatal("failed reload lost LSP environment files")
	}
	write("Notes:\n  Languages:\n    go:\n      LSP:\n        command: [gopls]\n        EnvFiles: [new.vars]\n  SpellCheck:\n    TyposLsp:\n      EnvFiles: [new-typos.vars]\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if files := Config.Notes.Languages["go"].LSP.EnvFiles; len(files) != 1 || files[0] != "new.vars" {
		t.Fatal("reload did not replace language server environment files")
	}
	if files := Config.Notes.SpellCheck.TyposLsp.EnvFiles; len(files) != 1 || files[0] != "new-typos.vars" {
		t.Fatal("reload did not replace spellchecker environment files")
	}
	write("Notes:\n  Languages:\n    go:\n      LSP:\n        command: [gopls]\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if len(Config.Notes.Languages["go"].LSP.EnvFiles) != 0 || len(Config.Notes.SpellCheck.TyposLsp.EnvFiles) != 0 {
		t.Fatal("reload retained removed LSP environment files")
	}
}

func TestPaneAutoCloseDelayUsesConfiguredGoDuration(t *testing.T) {
	previous := Config.Terminal.PaneAutoCloseDelay
	t.Cleanup(func() { Config.Terminal.PaneAutoCloseDelay = previous })

	for _, test := range []struct {
		configured string
		want       time.Duration
	}{
		{configured: "17s", want: 17 * time.Second},
		{configured: "1m30s", want: 90 * time.Second},
		{configured: "invalid", want: defaultPaneAutoCloseDelay},
		{configured: "0s", want: defaultPaneAutoCloseDelay},
	} {
		Config.Terminal.PaneAutoCloseDelay = test.configured
		if got := PaneAutoCloseDelay(); got != test.want {
			t.Errorf("PaneAutoCloseDelay(%q) = %s, want %s", test.configured, got, test.want)
		}
	}
}

func TestMergeMapMergesNestedMapsAndOverwritesValues(t *testing.T) {
	older := map[string]any{
		"name": "old",
		"nested": map[string]any{
			"keep":    true,
			"replace": "old",
		},
		"list": []any{"old"},
	}
	newer := map[string]any{
		"name": "new",
		"nested": map[string]any{
			"add":     "value",
			"replace": "new",
		},
		"list": []any{"new"},
	}

	got := MergeMap(older, newer)
	if got["name"] != "new" {
		t.Fatalf("name = %v, want new", got["name"])
	}
	if got["list"].([]any)[0] != "new" {
		t.Fatalf("list = %v, want replacement", got["list"])
	}

	nested := got["nested"].(map[string]any)
	if nested["keep"] != true || nested["add"] != "value" || nested["replace"] != "new" {
		t.Fatalf("nested = %#v, want merged values", nested)
	}
}

func TestMergeMapAcceptsNestedMapWithDifferentKeyType(t *testing.T) {
	older := map[string]any{
		"nested": map[string]any{"keep": "yes"},
	}
	newer := map[string]any{
		"nested": map[any]any{"add": "value"},
	}

	MergeMap(older, newer)
	nested := older["nested"].(map[string]any)
	if nested["keep"] != "yes" || nested["add"] != "value" {
		t.Fatalf("nested = %#v, want merged values", nested)
	}
}

func TestMergeMapHandlesNilOlderMap(t *testing.T) {
	got := MergeMap(nil, map[string]any{"key": "value"})
	if got["key"] != "value" {
		t.Fatalf("got = %#v, want key=value", got)
	}
}

func TestMergeMapMergesStructPointerValues(t *testing.T) {
	older := map[string]*AIServiceT{
		"service": {
			Label:        "Service",
			Provider:     "openai",
			DefaultModel: "gpt-4",
			Env:          map[string]string{"OPENAI_API_KEY": "key"},
		},
	}
	newer := map[string]*AIServiceT{
		"service": {
			Description:        "Updated description",
			DefaultModel:       "gpt-5",
			SummariseModelYaml: "gpt-5-mini",
			Env:                map[string]string{"OPENAI_BASE_URL": "https://example.test"},
		},
	}

	MergeMap(older, newer)
	service := older["service"]
	if service.Label != "Service" || service.Provider != "openai" || service.DefaultModel != "gpt-5" {
		t.Fatalf("service identity/model = %#v, want preserved identity and updated model", service)
	}
	if service.Description != "Updated description" || service.SummariseModelYaml != "gpt-5-mini" {
		t.Fatalf("service descriptions = %#v, want updated values", service)
	}
	if service.Env["OPENAI_API_KEY"] != "key" || service.Env["OPENAI_BASE_URL"] != "https://example.test" {
		t.Fatalf("service env = %#v, want merged values", service.Env)
	}
}
