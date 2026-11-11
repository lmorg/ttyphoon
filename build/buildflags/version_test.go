package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIncrementConfigVersionPreservesYamlFormatting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	content := []byte("# build config\nversion: '3'\ninfo:\n  productName: Ttyphoon\n  version: \"1.3.0099\" # app version\n")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}

	previous, next, err := incrementConfigVersion(path)
	if err != nil {
		t.Fatal(err)
	}
	if previous != "1.3.0099" || next != "1.3.0100" {
		t.Fatalf("incremented version = %q -> %q", previous, next)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "# build config\nversion: '3'\ninfo:\n  productName: Ttyphoon\n  version: \"1.3.0100\" # app version\n"
	if string(got) != want {
		t.Fatalf("config after increment:\n%s\nwant:\n%s", got, want)
	}
}

func TestIncrementVersionRejectsUnsupportedVersions(t *testing.T) {
	for _, config := range []string{
		"info: {}\n",
		"info:\n  version: 1.2\n",
		"info:\n  version: 1.2.x\n",
	} {
		if _, _, _, err := incrementVersion([]byte(config)); err == nil {
			t.Errorf("accepted invalid version config %q", config)
		}
	}

	config := []byte("info:\n  version: 1.2.3\n")
	updated, _, _, err := incrementVersion(config)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "1.2.4") {
		t.Fatalf("version was not incremented: %s", updated)
	}
}
