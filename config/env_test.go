package config

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/utils/envvars"
)

func TestTerminalEnvironmentHelper(t *testing.T) {
	if os.Getenv("TTYPHOON_TEST_TERMINAL_HELPER") != "1" {
		return
	}
	values := make(map[string]string)
	for _, name := range []string{
		"TTYPHOON_TEST_HOST", "TTYPHOON_TEST_FILE_ONLY", "TMUX", "TERM",
		"TERM_PROGRAM", "GITHUB_TOKEN", "ANTHROPIC_API_KEY", "OPENAI_API_KEY",
		"OLLAMA_HOST", "MXTTY", "MXTTY_TMUX",
	} {
		if value, ok := os.LookupEnv(name); ok {
			values[name] = value
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(values); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestUnreferencedFilesAndScopedValuesDoNotLeakToTerminal(t *testing.T) {
	previousConfig, previousCache, previousDirs := Config, langCache, xdg.ConfigDirs
	t.Cleanup(func() { Config, langCache, xdg.ConfigDirs = previousConfig, previousCache, previousDirs })
	Config, langCache = configT{}, LanguagesT{}
	xdg.ConfigDirs = []string{t.TempDir()}
	dir := filepath.Join(xdg.ConfigDirs[0], app.DirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TTYPHOON_TEST_HOST", "host")
	t.Setenv("TTYPHOON_TEST_FILE_ONLY", "")
	if err := os.Unsetenv("TTYPHOON_TEST_FILE_ONLY"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"TMUX", "TERM", "TERM_PROGRAM", "GITHUB_TOKEN", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OLLAMA_HOST"} {
		t.Setenv(name, "host-value")
	}
	unreferenced := filepath.Join(dir, "unreferenced.env")
	if err := os.WriteFile(unreferenced, []byte("TTYPHOON_TEST_HOST=unreferenced\nTTYPHOON_TEST_FILE_ONLY=unreferenced\n!LEGACY_NAME=value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	scopedFile := filepath.Join(dir, "referenced.env")
	if err := os.WriteFile(scopedFile, []byte("TTYPHOON_TEST_HOST=scoped\nTTYPHOON_TEST_FILE_ONLY=scoped\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	scoped, err := envvars.Resolve([]string{scopedFile}, map[string]string{"TTYPHOON_TEST_HOST": ""})
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := scoped.Lookup("TTYPHOON_TEST_HOST"); !ok || value != "" {
		t.Fatal("explicit empty value did not override the scoped file and host")
	}
	if value, ok := scoped.Lookup("TTYPHOON_TEST_FILE_ONLY"); !ok || value != "scoped" {
		t.Fatal("referenced environment file was not resolved")
	}
	Config.Tmux.Enabled = true
	cmd := exec.Command(os.Args[0], "-test.run=^TestTerminalEnvironmentHelper$")
	cmd.Env = append(SetEnv(), "TTYPHOON_TEST_TERMINAL_HELPER=1")
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]string
	if err := json.Unmarshal(output, &values); err != nil {
		t.Fatal(err)
	}
	if values["TTYPHOON_TEST_HOST"] != "host" {
		t.Fatal("terminal did not inherit the original host value")
	}
	for _, name := range []string{"TTYPHOON_TEST_FILE_ONLY", "TMUX", "GITHUB_TOKEN", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OLLAMA_HOST"} {
		if _, ok := values[name]; ok {
			t.Errorf("terminal inherited excluded or scoped variable %s", name)
		}
	}
	for name, want := range map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "mxtty", "MXTTY": "true", "MXTTY_TMUX": "true"} {
		if values[name] != want {
			t.Errorf("terminal export %s = %q, want %q", name, values[name], want)
		}
	}
}
