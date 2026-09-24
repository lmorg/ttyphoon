package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newAssetTestApp(t *testing.T) (*WApp, string) {
	t.Helper()

	baseDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(baseDir, "diagram.png"), []byte("not-really-a-png"), 0o600); err != nil {
		t.Fatalf("unable to seed asset: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(baseDir, "nested"), 0o700); err != nil {
		t.Fatalf("unable to create nested dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "nested", "chart.svg"), []byte("<svg/>"), 0o600); err != nil {
		t.Fatalf("unable to seed nested asset: %v", err)
	}

	app := &WApp{}
	app.setMarkdownBaseDir(baseDir)
	return app, baseDir
}

func TestResolveNotesAssetAllowsFilesUnderTheDocumentDirectory(t *testing.T) {
	app, baseDir := newAssetTestApp(t)

	for _, requested := range []string{"diagram.png", "nested/chart.svg", filepath.Join(baseDir, "diagram.png")} {
		resolved, err := app.resolveNotesAsset(requested)
		if err != nil {
			t.Fatalf("resolveNotesAsset(%q) returned error: %v", requested, err)
		}
		if !pathWithinRoot(resolveSymlinks(baseDir), resolved) {
			t.Fatalf("resolveNotesAsset(%q) escaped the base dir: %s", requested, resolved)
		}
	}
}

func TestResolveNotesAssetAllowsFilesUnderDocumentsTtyphoon(t *testing.T) {
	app, _ := newAssetTestApp(t)
	app.homeDir = t.TempDir()
	documentsDir := app.documentsTtyphoonDir()
	if err := os.MkdirAll(documentsDir, 0o700); err != nil {
		t.Fatalf("unable to create Documents/ttyphoon: %v", err)
	}
	imagePath := filepath.Join(documentsDir, "reference.png")
	if err := os.WriteFile(imagePath, []byte("not-really-a-png"), 0o600); err != nil {
		t.Fatalf("unable to seed Documents asset: %v", err)
	}

	resolved, err := app.resolveNotesAsset(imagePath)
	if err != nil {
		t.Fatalf("resolveNotesAsset(%q) returned error: %v", imagePath, err)
	}
	if !pathWithinRoot(resolveSymlinks(documentsDir), resolved) {
		t.Fatalf("resolveNotesAsset(%q) escaped Documents/ttyphoon: %s", imagePath, resolved)
	}
}

func TestResolveNotesAssetRejectsEscapesAndNonImages(t *testing.T) {
	app, baseDir := newAssetTestApp(t)

	outside := filepath.Join(filepath.Dir(baseDir), "outside.png")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatalf("unable to seed outside asset: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })

	secret := filepath.Join(baseDir, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatalf("unable to seed secret: %v", err)
	}

	tests := map[string]string{
		"empty path":            "",
		"traversal above base":  "../outside.png",
		"absolute outside base": outside,
		"non-image extension":   "secret.txt",
		"no extension":          "/etc/passwd",
		"missing file":          "absent.png",
		"directory":             "nested",
	}

	for name, requested := range tests {
		t.Run(name, func(t *testing.T) {
			if resolved, err := app.resolveNotesAsset(requested); err == nil {
				t.Fatalf("expected %q to be rejected, but it resolved to %s", requested, resolved)
			}
		})
	}
}

func TestNotesAssetHandlerServesAndRefusesAppropriately(t *testing.T) {
	app, _ := newAssetTestApp(t)
	handler := app.notesAssetHandler()

	t.Run("serves a permitted asset", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, notesAssetURLPath+"?path=diagram.png", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "not-really-a-png" {
			t.Fatalf("unexpected body: %q", body)
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
			t.Fatalf("expected revalidation, got Cache-Control %q", cc)
		}
	})

	t.Run("404s a traversal attempt", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, notesAssetURLPath+"?path=../../../../etc/hosts", nil))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("refuses non-read methods", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, notesAssetURLPath+"?path=diagram.png", nil))

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})
}
