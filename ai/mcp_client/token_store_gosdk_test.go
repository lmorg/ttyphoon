package mcp_client

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestTokenFileReplacementRemainsReadableDuringConcurrentWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	if err := writeTokenFile(path, &oauth2.Token{AccessToken: "initial", Expiry: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for worker := range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 20 {
				if err := writeTokenFile(path, &oauth2.Token{AccessToken: fmt.Sprintf("fixture-%d", worker), Expiry: time.Now().Add(time.Hour)}); err != nil {
					t.Error(err)
					return
				}
				if _, err := readTokenFile(path); err != nil {
					t.Errorf("read partial token file: %v", err)
					return
				}
			}
		}()
	}
	group.Wait()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("token permissions changed: %v", err)
	}
}

func TestFilePersistingTokenSource_ReadExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	tok := &oauth2.Token{AccessToken: "at-123", RefreshToken: "rt-1", Expiry: time.Now().Add(time.Hour)}
	if err := writeTokenFile(path, tok); err != nil {
		t.Fatalf("writeTokenFile failed: %v", err)
	}

	ts := NewFilePersistingTokenSource(path, nil)
	got, err := ts.Token()
	if err != nil {
		t.Fatalf("Token() returned error: %v", err)
	}
	if got.AccessToken != tok.AccessToken {
		t.Fatalf("access token mismatch: got=%q want=%q", got.AccessToken, tok.AccessToken)
	}
}

func TestFilePersistingTokenSource_PersistFromBase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	base := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "base-zzz", Expiry: time.Now().Add(2 * time.Hour)})
	ts := NewFilePersistingTokenSource(path, base)

	got, err := ts.Token()
	if err != nil {
		t.Fatalf("Token() returned error: %v", err)
	}
	if got.AccessToken != "base-zzz" {
		t.Fatalf("unexpected token from base: %v", got.AccessToken)
	}

	// ensure file was written
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected token file to be written: %v", err)
	}

	// read back via helper
	rt, err := readTokenFile(path)
	if err != nil {
		t.Fatalf("readTokenFile failed: %v", err)
	}
	if rt.AccessToken != "base-zzz" {
		t.Fatalf("persisted token mismatch: got=%q want=%q", rt.AccessToken, "base-zzz")
	}
}
