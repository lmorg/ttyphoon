package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestLinkerFlagsInjectMetadata(t *testing.T) {
	builtAt := time.Date(2026, 11, 8, 12, 0, 0, 0, time.FixedZone("test", 3600))
	info := map[string]string{
		"productName": "Morgan's Terminal",
		"description": `A "quoted" terminal with $HOME and \paths`,
		"version":     "9.8.7",
		"copyright":   "2023-2026 Laurence Morgan",
		"license":     "GPL-2.0-or-later",
	}
	data, err := yaml.Marshal(map[string]any{"info": info})
	if err != nil {
		t.Fatal(err)
	}
	flags, err := linkerFlags(data, "feature/build-metadata", builtAt)
	if err != nil {
		t.Fatal(err)
	}

	binary := filepath.Join(t.TempDir(), "metadata-probe")
	cmd := exec.Command("go", "build", "-ldflags=-w -s "+flags, "-o", binary, "./testdata/probe.go")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build metadata probe: %v\n%s", err, output)
	}
	output, err := exec.Command(binary).CombinedOutput()
	if err != nil {
		t.Fatalf("run metadata probe: %v\n%s", err, output)
	}
	var got []string
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("decode probe output: %v\n%s", err, output)
	}
	want := []string{
		info["productName"], info["description"], info["version"], info["copyright"],
		"feature/build-metadata", "2026-11-08T11:00:00Z", info["license"],
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("metadata field %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLinkerFlagsRejectInvalidConfig(t *testing.T) {
	for _, test := range []struct {
		name  string
		field string
		value string
	}{
		{"missing name", "productName", ""},
		{"blank description", "description", "  "},
		{"missing version", "version", ""},
		{"missing copyright", "copyright", ""},
		{"missing license", "license", ""},
		{"multiline", "description", "line one\nline two"},
		{"null byte", "productName", "name\x00"},
		{"both quote styles", "description", `It's a "terminal"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			info := map[string]string{
				"productName": "Test", "description": "Terminal",
				"version": "1.2.3", "copyright": "Copyright",
				"license": "GPL-2.0-or-later",
			}
			info[test.field] = test.value
			data, err := yaml.Marshal(map[string]any{"info": info})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := linkerFlags(data, "main", time.Now()); err == nil || !strings.Contains(err.Error(), "info."+test.field) {
				t.Fatalf("expected error for info.%s, got %v", test.field, err)
			}
		})
	}
	if _, err := linkerFlags([]byte("info: ["), "main", time.Now()); err == nil {
		t.Fatal("expected malformed YAML error")
	}
	if _, err := buildFlags(filepath.Join(t.TempDir(), "missing.yml")); err == nil {
		t.Fatal("expected missing config error")
	}
}

func TestBuildFlagsReadsProjectConfig(t *testing.T) {
	flags, err := buildFlags("../config.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(flags, "github.com/lmorg/ttyphoon/app.version=") ||
		!strings.Contains(flags, "github.com/lmorg/ttyphoon/app.branch=") ||
		!strings.Contains(flags, "github.com/lmorg/ttyphoon/app.buildDate=") {
		t.Fatalf("missing metadata linker flags: %s", flags)
	}
}
