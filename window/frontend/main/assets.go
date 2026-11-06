package mainwindow

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lmorg/ttyphoon/utils/file"
)

var rxExtension = regexp.MustCompile(`.[a-zA-Z0-9]+$`)

// notesAssetURLPath serves Notes images to the webview directly. Returning them
// as base64 over the JS bridge put the whole file on the main thread.
const notesAssetURLPath = "/__asset"

var notesAssetExtensions = map[string]bool{
	".apng": true, ".avif": true, ".bmp": true, ".gif": true, ".ico": true,
	".jpeg": true, ".jpg": true, ".png": true, ".svg": true, ".tif": true,
	".tiff": true, ".webp": true,
}

func AssetHandler(a *WApp, frontend http.Handler) http.Handler {
	notesAssets := a.notesAssetHandler()
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == notesAssetURLPath {
			notesAssets.ServeHTTP(writer, request)
			return
		}
		frontend.ServeHTTP(writer, request)
	})
}

func (a *WApp) setMarkdownBaseDir(dir string) {
	a.mdBaseDirMu.Lock()
	a.mdBaseDir = dir
	a.mdBaseDirMu.Unlock()
}

func (a *WApp) markdownBaseDir() string {
	a.mdBaseDirMu.RLock()
	defer a.mdBaseDirMu.RUnlock()
	return a.mdBaseDir
}

func (a *WApp) documentsTtyphoonDir() string {
	return file.DocumentsDir(a.homeDir)
}

func (a *WApp) notesAssetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		resolved, err := a.resolveNotesAsset(r.URL.Query().Get("path"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Revalidate rather than cache: an edited image keeps the same URL.
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, resolved)
	})
}

// resolveNotesAsset confines requests to directories Notes owns, so a crafted
// markdown link cannot walk out and read arbitrary files.
func (a *WApp) resolveNotesAsset(requested string) (string, error) {
	if requested == "" {
		return "", errors.New("empty asset path")
	}

	if !notesAssetExtensions[strings.ToLower(filepath.Ext(requested))] {
		return "", errors.New("unsupported asset type")
	}

	baseDir := a.markdownBaseDir()
	candidate := requested
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(baseDir, candidate)
	}
	candidate = resolveSymlinks(filepath.Clean(candidate))

	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return "", errors.New("asset not found")
	}

	for _, root := range []string{baseDir, a.GetCurrentProject(), a.currentNotesContext().PathUserNotes, a.currentNotesContext().PathGlobalNotes, a.documentsTtyphoonDir()} {
		if root == "" {
			continue
		}
		if pathWithinRoot(resolveSymlinks(filepath.Clean(root)), candidate) {
			return candidate, nil
		}
	}

	return "", errors.New("asset outside permitted directories")
}

func resolveSymlinks(path string) string {
	if actual, err := filepath.EvalSymlinks(path); err == nil {
		return actual
	}
	return path
}

func pathWithinRoot(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// resolveMarkdownAssetPath mirrors how the Notes viewer locates relative asset
// paths: absolute-from-root first, falling back to the current document's directory.
func (a *WApp) resolveMarkdownAssetPath(path string) string {
	resolvedPath := path
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = string(filepath.Separator) + strings.TrimLeft(resolvedPath, string(filepath.Separator))
	}
	if _, err := os.Stat(resolvedPath); err != nil {
		resolvedPath = filepath.Join(a.markdownBaseDir(), strings.TrimLeft(path, "/\\"))
	}
	return resolvedPath
}

func (a *WApp) GetImage(path string) string {
	if len(path) == 0 {
		return "error: empty string"
	}

	ext := strings.ToLower(rxExtension.FindString(path))
	if len(ext) == 0 {
		return "error: extension not found"
	}

	resolvedPath := a.resolveMarkdownAssetPath(path)

	f, err := os.Open(resolvedPath)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	base64 := base64.StdEncoding.EncodeToString(b)

	return fmt.Sprintf("data:%s;base64,%s", imageMime(ext), base64)
}

func imageMime(ext string) string {
	if ext == ".svg" {
		return "image/svg+xml"
	}
	return "image/" + ext[1:]
}
