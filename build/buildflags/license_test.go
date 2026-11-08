package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSyncPackageLicense(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	nfpmPath := filepath.Join(dir, "nfpm.yaml")
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(configPath, "info:\n  license: GPL-2.0-or-later\n")
	original := "# Package metadata\nname: ttyphoon\nlicense: \"MIT\" # license label\narch: ${GOARCH}\ndepends:\n  - libgtk-4-1\n"
	write(nfpmPath, original)
	if err := syncPackageLicense(configPath, nfpmPath); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(nfpmPath)
	if err != nil {
		t.Fatal(err)
	}
	var got, want map[string]any
	if err := yaml.Unmarshal(output, &got); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal([]byte(original), &want); err != nil {
		t.Fatal(err)
	}
	if got["license"] != "GPL-2.0-or-later" {
		t.Fatalf("license = %v", got["license"])
	}
	delete(got, "license")
	delete(want, "license")
	gotData, err := yaml.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	wantData, err := yaml.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotData) != string(wantData) {
		t.Fatal("unrelated package metadata changed")
	}
	for _, comment := range []string{"# Package metadata", "# license label"} {
		if !strings.Contains(string(output), comment) {
			t.Errorf("lost comment %q", comment)
		}
	}
	if err := syncPackageLicense(configPath, nfpmPath); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(nfpmPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != string(second) {
		t.Fatal("license update is not idempotent")
	}
	for _, invalid := range []string{"{}", "[]", "license: [MIT]", "license: ["} {
		write(nfpmPath, invalid)
		if err := syncPackageLicense(configPath, nfpmPath); err == nil {
			t.Errorf("expected error for %q", invalid)
		}
	}
	write(configPath, "info: {}")
	write(nfpmPath, original)
	if err := syncPackageLicense(configPath, nfpmPath); err == nil {
		t.Fatal("expected missing license error")
	}
}
