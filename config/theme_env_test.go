package config

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
)

func setupThemeConfig(t *testing.T) (string, string) {
	t.Helper()
	previousConfig, previousCache, previousDirs := Config, langCache, xdg.ConfigDirs
	background, shadow, light := *types.SGR_COLOR_BACKGROUND, *types.COLOR_TEXT_SHADOW, types.THEME_LIGHT
	t.Cleanup(func() {
		Config, langCache, xdg.ConfigDirs = previousConfig, previousCache, previousDirs
		*types.SGR_COLOR_BACKGROUND, *types.COLOR_TEXT_SHADOW, types.THEME_LIGHT = background, shadow, light
	})
	Config, langCache = configT{}, LanguagesT{}
	xdg.ConfigDirs = []string{t.TempDir(), t.TempDir()}
	first, second := filepath.Join(xdg.ConfigDirs[0], app.DirName), filepath.Join(xdg.ConfigDirs[1], app.DirName)
	for _, dir := range []string{first, second} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return first, second
}

func writeThemeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func writeThemeTestColor(t *testing.T, path, red string) {
	t.Helper()
	writeThemeTestFile(t, path, fmt.Sprintf(`<?xml version="1.0"?><plist><dict>
<key>Background Color</key><dict><key>Red Component</key><real>%s</real>
<key>Green Component</key><real>0</real><key>Blue Component</key><real>0</real></dict>
</dict></plist>`, red))
}

func TestColorThemeScopedEnvironmentAndReload(t *testing.T) {
	first, second := setupThemeConfig(t)
	dir := t.TempDir()
	hostTheme, firstTheme, secondTheme := filepath.Join(dir, "host.itermcolors"), filepath.Join(dir, "first.itermcolors"), filepath.Join(dir, "second.itermcolors")
	writeThemeTestColor(t, hostTheme, "0.1")
	writeThemeTestColor(t, firstTheme, "0.2")
	writeThemeTestColor(t, secondTheme, "0.8")
	hostMissing := filepath.Join(dir, "host-missing.itermcolors")
	t.Setenv("TTYPHOON_TEST_THEME", hostMissing)
	writeThemeTestFile(t, filepath.Join(first, "one.yaml"), "Terminal:\n  ColorTheme: '$TTYPHOON_TEST_THEME'\n")
	writeThemeTestFile(t, filepath.Join(second, "two.yaml"), "Terminal:\n  EnvFiles: [theme.vars, last.vars]\n")
	writeThemeTestFile(t, filepath.Join(first, "theme.vars"), "TTYPHOON_TEST_THEME="+hostTheme+"\n")
	writeThemeTestFile(t, filepath.Join(second, "theme.vars"), "TTYPHOON_TEST_THEME="+secondTheme+"\n")
	last := filepath.Join(first, "last.vars")
	writeThemeTestFile(t, last, "TTYPHOON_TEST_THEME="+firstTheme+"\n")

	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if types.SGR_COLOR_BACKGROUND.Red != 51 {
		t.Fatal("theme did not use the last scoped environment file")
	}
	if os.Getenv("TTYPHOON_TEST_THEME") != hostMissing {
		t.Fatal("theme environment changed the host")
	}
	writeThemeTestFile(t, filepath.Join(second, "two.yaml"), "Terminal:\n  EnvFiles: [theme.vars]\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if types.SGR_COLOR_BACKGROUND.Red != 204 {
		t.Fatal("later XDG environment file did not override the earlier file")
	}
	writeThemeTestFile(t, filepath.Join(second, "theme.vars"), "TTYPHOON_TEST_THEME="+firstTheme+"\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if types.SGR_COLOR_BACKGROUND.Red != 51 {
		t.Fatal("reload did not reread the theme environment files")
	}
	t.Setenv("TTYPHOON_TEST_THEME", hostTheme)
	writeThemeTestFile(t, filepath.Join(second, "two.yaml"), "Terminal: {}\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if len(Config.Terminal.EnvFiles) != 0 || types.SGR_COLOR_BACKGROUND.Red != 26 {
		t.Fatal("removed environment files did not restore host fallback")
	}
	writeThemeTestFile(t, filepath.Join(first, "one.yaml"), "Terminal: {}\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if Config.Terminal.ColorTheme != "" {
		t.Fatal("reload retained a removed theme")
	}
}

func TestColorThemeScopedEnvironmentWithHomeExpansion(t *testing.T) {
	first, _ := setupThemeConfig(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeThemeTestColor(t, filepath.Join(home, "scoped.itermcolors"), "0.8")
	writeThemeTestFile(t, filepath.Join(first, "theme.vars"), "TTYPHOON_THEME_NAME=scoped.itermcolors\n")
	writeThemeTestFile(t, filepath.Join(first, "config.yaml"), "Terminal:\n  ColorTheme: '~/$TTYPHOON_THEME_NAME'\n  EnvFiles: [theme.vars]\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if types.SGR_COLOR_BACKGROUND.Red != 204 {
		t.Fatal("theme did not expand the home directory and scoped filename")
	}
}

func TestColorThemeStillSearchesThemeAndConfigDirectories(t *testing.T) {
	first, _ := setupThemeConfig(t)
	themes := filepath.Join(first, "themes")
	if err := os.MkdirAll(themes, 0700); err != nil {
		t.Fatal(err)
	}
	themePath := filepath.Join(themes, "example.itermcolors")
	writeThemeTestColor(t, themePath, "0.2")
	writeThemeTestColor(t, filepath.Join(first, "example.itermcolors"), "0.8")
	writeThemeTestFile(t, filepath.Join(first, "config.yaml"), "Terminal:\n  ColorTheme: example.itermcolors\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if types.SGR_COLOR_BACKGROUND.Red != 51 {
		t.Fatal("theme directory did not take precedence")
	}
	if err := os.Remove(themePath); err != nil {
		t.Fatal(err)
	}
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if types.SGR_COLOR_BACKGROUND.Red != 204 {
		t.Fatal("theme was not loaded from the config directory")
	}
}

func TestColorThemeEnvironmentErrorsAndRedaction(t *testing.T) {
	first, _ := setupThemeConfig(t)
	dir := t.TempDir()
	theme := filepath.Join(dir, "theme.itermcolors")
	writeThemeTestColor(t, theme, "0.2")
	t.Setenv("TTYPHOON_TEST_THEME", theme)
	configFile := filepath.Join(first, "config.yaml")
	envFile := filepath.Join(dir, "theme.vars")
	writeThemeTestFile(t, configFile, fmt.Sprintf("Terminal:\n  ColorTheme: '$TTYPHOON_TEST_THEME'\n  EnvFiles: [%q]\n", envFile))
	writeThemeTestFile(t, envFile, "TTYPHOON_TEST_THEME="+theme+"\n")
	if err := LoadConfig(); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(dir, "missing.vars")
	writeThemeTestFile(t, configFile, fmt.Sprintf("Terminal:\n  ColorTheme: '$DIFFERENT_THEME'\n  EnvFiles: [%q]\n", missing))
	if err := LoadConfig(); err == nil {
		t.Fatal("missing replacement environment file was accepted")
	}
	if Config.Terminal.ColorTheme != "$TTYPHOON_TEST_THEME" || len(Config.Terminal.EnvFiles) != 1 || Config.Terminal.EnvFiles[0] != envFile {
		t.Fatal("failed reload did not restore previous theme settings")
	}
	writeThemeTestFile(t, configFile, fmt.Sprintf("Terminal:\n  ColorTheme: '$TTYPHOON_TEST_THEME'\n  EnvFiles: [%q]\n", envFile))
	writeThemeTestFile(t, envFile, "TTYPHOON_TEST_THEME=\n")
	if err := LoadConfig(); err == nil {
		t.Fatal("empty scoped theme value incorrectly fell back to the host")
	}
	if Config.Terminal.ColorTheme != "$TTYPHOON_TEST_THEME" || len(Config.Terminal.EnvFiles) != 1 || Config.Terminal.EnvFiles[0] != envFile {
		t.Fatal("failed theme reload did not restore previous settings")
	}

	writeThemeTestFile(t, envFile, "invalid-sensitive-line\n")
	if err := LoadConfig(); err == nil || strings.Contains(err.Error(), "invalid-sensitive-line") {
		t.Fatal("malformed environment file was not reported safely")
	}
	if err := os.Remove(envFile); err != nil {
		t.Fatal(err)
	}
	if err := LoadConfig(); err == nil {
		t.Fatal("missing environment file was accepted")
	}

	const secret = "scoped-private-theme-value"
	writeThemeTestFile(t, envFile, "TTYPHOON_TEST_THEME="+secret+"\n")
	var logs bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousOutput) })
	err := LoadConfig()
	if err == nil || strings.Contains(err.Error(), secret) || !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatal("theme lookup error did not redact scoped value")
	}
	if strings.Contains(logs.String(), secret) {
		t.Fatal("theme lookup logged a scoped value")
	}
}
