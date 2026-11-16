package envvars

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/lmorg/ttyphoon/app"
)

func writeEnvFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestResolvePrecedenceAndOverlay(t *testing.T) {
	t.Setenv("RESOLVER_SHARED", "host")
	t.Setenv("RESOLVER_HOST_ONLY", "host-only")
	t.Setenv("RESOLVER_EMPTY", "host-value")
	dir := t.TempDir()
	first := filepath.Join(dir, "first.env")
	second := filepath.Join(dir, "second.env")
	writeEnvFile(t, first, "# comment\n\nRESOLVER_SHARED=first\nRESOLVER_FILE=one=two\nRESOLVER_FILE_EMPTY=\nRESOLVER_EMPTY=first\n")
	writeEnvFile(t, second, "RESOLVER_SHARED=second\r\nRESOLVER_EMPTY=file-value")

	env, err := Resolve([]string{first, second}, map[string]string{
		"RESOLVER_SHARED": "explicit",
		"RESOLVER_EMPTY":  "",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, want string
		present    bool
	}{
		{"RESOLVER_SHARED", "explicit", true},
		{"RESOLVER_EMPTY", "", true},
		{"RESOLVER_FILE", "one=two", true},
		{"RESOLVER_FILE_EMPTY", "", true},
		{"RESOLVER_HOST_ONLY", "host-only", true},
		{"RESOLVER_MISSING_TEST_KEY", "", false},
	} {
		value, ok := env.Lookup(test.name)
		if value != test.want || ok != test.present {
			t.Errorf("Lookup(%q) = %q, %t; want %q, %t", test.name, value, ok, test.want, test.present)
		}
	}

	base := map[string]string{"RESOLVER_SHARED": "base", "BASE_ONLY": "base"}
	overlay := env.Overlay(base)
	if overlay["RESOLVER_SHARED"] != "explicit" || overlay["BASE_ONLY"] != "base" || overlay["RESOLVER_EMPTY"] != "" {
		t.Fatalf("unexpected overlay: %#v", overlay)
	}
	if _, ok := overlay["RESOLVER_HOST_ONLY"]; ok {
		t.Fatal("overlay included host values not present in the base")
	}
	overlay["RESOLVER_SHARED"] = "modified"
	if base["RESOLVER_SHARED"] != "base" {
		t.Fatal("overlay modified the base")
	}
	if value, _ := env.Lookup("RESOLVER_SHARED"); value != "explicit" {
		t.Fatal("overlay modified the resolved environment")
	}
	if got := os.Getenv("RESOLVER_SHARED"); got != "host" {
		t.Fatalf("host environment changed: %q", got)
	}
}

func TestRedactorUsesScopedValuesOnly(t *testing.T) {
	t.Setenv("REDACTION_HOST", "host-value")
	env, err := Resolve(nil, map[string]string{
		"FIRST": "secret", "LONGER": "secret-longer", "EMPTY": "",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := env.Redactor()("secret-longer secret host-value")
	if got != "[REDACTED] [REDACTED] host-value" {
		t.Fatal("scoped redactor did not preserve host values or overlapping secrets")
	}
}

func TestResolveRelativeFilesAcrossConfigDirectories(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	writeEnvFile(t, filepath.Join(first, app.DirName, "nested", "settings.env"), "ORDER=first\nFIRST=present\n")
	writeEnvFile(t, filepath.Join(second, app.DirName, "nested", "settings.env"), "ORDER=second\n")
	writeEnvFile(t, filepath.Join(first, app.DirName, "last.env"), "ORDER=last-entry\n")

	env, err := resolve([]string{"nested/settings.env", "last.env"}, nil, []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := env.Lookup("ORDER"); got != "last-entry" {
		t.Fatalf("ORDER = %q, want last entry", got)
	}
	env, err = resolve([]string{"nested/settings.env"}, nil, []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := env.Lookup("ORDER"); got != "second" {
		t.Fatalf("ORDER = %q, want second config directory", got)
	}
	if got, _ := env.Lookup("FIRST"); got != "present" {
		t.Fatalf("FIRST = %q, want present", got)
	}
	writeEnvFile(t, filepath.Join(second, app.DirName, "nested", "settings.env"), "!INVALID=value\n")
	if env, err := resolve([]string{"nested/settings.env"}, nil, []string{first, second}); err == nil || env != nil {
		t.Fatalf("resolve() = %v, %v; want error and no partial environment", env, err)
	}
}

func TestResolveRejectsInvalidFilesAndInputs(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside.env")
	writeEnvFile(t, outside, "SECRET=private-value\n")
	root := filepath.Join(dir, "config", app.DirName)
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.env")); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name, content, entry string
		envs                 map[string]string
	}{
		{name: "missing", entry: "missing.env"},
		{name: "missing absolute", entry: filepath.Join(dir, "missing.env")},
		{name: "directory", entry: root},
		{name: "traversal", entry: "../outside.env"},
		{name: "symlink escape", entry: "escape.env"},
		{name: "malformed", content: "private-value\n"},
		{name: "legacy prefix", content: "!SECRET=private-value\n"},
		{name: "invalid name", content: "BAD NAME=private-value\n"},
		{name: "nul value", content: "SECRET=private-value\x00\n"},
		{name: "invalid explicit", envs: map[string]string{"!SECRET": "private-value"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			entry := test.entry
			if test.content != "" {
				entry = filepath.Join(dir, test.name+".env")
				writeEnvFile(t, entry, test.content)
			}
			var files []string
			if entry != "" {
				files = []string{entry}
			}
			env, err := resolve(files, test.envs, []string{filepath.Join(dir, "config")})
			if err == nil || env != nil {
				t.Fatalf("resolve() = %v, %v; want error and no environment", env, err)
			}
			if strings.Contains(err.Error(), "private-value") {
				t.Fatalf("error exposed a value: %v", err)
			}
		})
	}
}

func TestResolveKeepsFeatureScopesIndependent(t *testing.T) {
	t.Setenv("RESOLVER_SCOPE", "host")
	dir := t.TempDir()
	first := filepath.Join(dir, "first.env")
	second := filepath.Join(dir, "second.env")
	writeEnvFile(t, first, "RESOLVER_SCOPE=first\n")
	writeEnvFile(t, second, "RESOLVER_SCOPE=second\n")

	var wg sync.WaitGroup
	for _, test := range []struct{ path, want string }{{first, "first"}, {second, "second"}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			env, err := Resolve([]string{test.path}, nil)
			if err != nil {
				t.Error(err)
				return
			}
			if value, _ := env.Lookup("RESOLVER_SCOPE"); value != test.want {
				t.Errorf("scoped value = %q, want %q", value, test.want)
			}
		}()
	}
	wg.Wait()
	if value := os.Getenv("RESOLVER_SCOPE"); value != "host" {
		t.Fatalf("host value changed to %q", value)
	}
}
