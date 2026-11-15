package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

// ----------------------------------------------------------------------------
// URI helpers
// ----------------------------------------------------------------------------

// FilePathToURI converts an absolute file path to a file:// URI.
// Handles Windows drive letters and forward/back slashes.
func FilePathToURI(absPath string) string {
	abs := filepath.ToSlash(absPath)

	// Windows absolute paths start with a drive letter, e.g. C:/...
	if runtime.GOOS == "windows" && len(abs) >= 2 && abs[1] == ':' {
		abs = "/" + abs
	}

	u := url.URL{Scheme: "file", Path: abs}
	return u.String()
}

// URIToFilePath converts a file:// URI back to an OS-native file path.
func URIToFilePath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("lsp: parse URI %q: %w", uri, err)
	}

	if u.Scheme != "file" {
		return "", fmt.Errorf("lsp: URI scheme %q is not 'file'", u.Scheme)
	}

	path := filepath.FromSlash(u.Path)

	// Strip leading slash on Windows paths like /C:/...
	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == filepath.Separator && path[2] == ':' {
		path = path[1:]
	}

	return path, nil
}

// ----------------------------------------------------------------------------
// Document model
// ----------------------------------------------------------------------------

// Document tracks the LSP-synced state of a single open Notes file.
type Document struct {
	URI        string // file:// URI
	FilePath   string // OS-native absolute path
	LanguageID string
	version    atomic.Int64
	mu         sync.RWMutex
	content    string
	open       bool
	refs       int // guarded by DocumentStore.mu
}

func newDocument(absPath, languageID, content string) *Document {
	d := &Document{
		URI:        FilePathToURI(absPath),
		FilePath:   absPath,
		LanguageID: languageID,
		content:    content,
		open:       true,
		refs:       1,
	}
	d.version.Store(1)
	return d
}

func (d *Document) Version() int64  { return d.version.Load() }
func (d *Document) IsOpen() bool    { d.mu.RLock(); defer d.mu.RUnlock(); return d.open }
func (d *Document) Content() string { d.mu.RLock(); defer d.mu.RUnlock(); return d.content }

func (d *Document) update(content string) int64 {
	d.mu.Lock()
	d.content = content
	d.mu.Unlock()
	return d.version.Add(1)
}

func (d *Document) close() {
	d.mu.Lock()
	d.open = false
	d.mu.Unlock()
}

// ----------------------------------------------------------------------------
// DocumentStore — owns all open documents for one language server session
// ----------------------------------------------------------------------------

// DocumentStore tracks open documents and sends sync notifications to a
// Transport. Opens are reference counted per URI so that several holders (for
// example two editor views of one file) can share a document: the first
// DidOpen sends didOpen and the last DidClose sends didClose.
type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]*Document // keyed by URI
}

func NewDocumentStore() *DocumentStore {
	return &DocumentStore{docs: make(map[string]*Document)}
}

// DidOpen adds a reference to the document. The first reference sends
// textDocument/didOpen; later ones send textDocument/didChange if content
// differs from what the server last saw.
func (ds *DocumentStore) DidOpen(ctx context.Context, t *Transport, absPath, languageID, content string) error {
	uri := FilePathToURI(absPath)

	ds.mu.Lock()
	if doc, ok := ds.docs[uri]; ok {
		doc.refs++
		ds.mu.Unlock()
		if doc.Content() == content {
			return nil
		}
		return ds.notifyChange(t, doc, content)
	}
	doc := newDocument(absPath, languageID, content)
	ds.docs[uri] = doc
	ds.mu.Unlock()

	params := map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": languageID,
			"version":    doc.Version(),
			"text":       content,
		},
	}
	return t.Notify("textDocument/didOpen", params)
}

// DidChange updates the document content and sends textDocument/didChange
// (full sync).
func (ds *DocumentStore) DidChange(ctx context.Context, t *Transport, absPath, content string) error {
	uri := FilePathToURI(absPath)

	ds.mu.RLock()
	doc, ok := ds.docs[uri]
	ds.mu.RUnlock()

	if !ok {
		return fmt.Errorf("lsp: DidChange: document %q is not open", uri)
	}

	return ds.notifyChange(t, doc, content)
}

func (ds *DocumentStore) notifyChange(t *Transport, doc *Document, content string) error {
	version := doc.update(content)

	params := map[string]any{
		"textDocument": map[string]any{
			"uri":     doc.URI,
			"version": version,
		},
		"contentChanges": []map[string]any{
			{"text": content},
		},
	}
	return t.Notify("textDocument/didChange", params)
}

// DidSave sends textDocument/didSave.
func (ds *DocumentStore) DidSave(ctx context.Context, t *Transport, absPath string) error {
	uri := FilePathToURI(absPath)
	return t.Notify("textDocument/didSave", map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})
}

// DidClose drops a reference to the document. The last reference removes it
// and sends textDocument/didClose; t may be nil to only update the store.
func (ds *DocumentStore) DidClose(ctx context.Context, t *Transport, absPath string) error {
	uri := FilePathToURI(absPath)

	ds.mu.Lock()
	doc, ok := ds.docs[uri]
	if ok {
		doc.refs--
		if doc.refs > 0 {
			ds.mu.Unlock()
			return nil
		}
		doc.close()
		delete(ds.docs, uri)
	}
	ds.mu.Unlock()

	if !ok || t == nil {
		return nil // already closed; not an error
	}

	return t.Notify("textDocument/didClose", map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})
}

// Get returns the Document for a path, or nil if not open.
func (ds *DocumentStore) Get(absPath string) *Document {
	uri := FilePathToURI(absPath)
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.docs[uri]
}

// GetByURI returns the Document for a URI, or nil if not open.
func (ds *DocumentStore) GetByURI(uri string) *Document {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.docs[uri]
}

// IsOpen reports whether the path is currently tracked as open.
func (ds *DocumentStore) IsOpen(absPath string) bool {
	return ds.Get(absPath) != nil
}

// ----------------------------------------------------------------------------
// LSP position helpers (utf-16 <-> utf-8 offset conversion)
// ----------------------------------------------------------------------------

// PositionToOffset converts a 0-based LSP line/character (utf-16 code units)
// to a byte offset in content. Returns -1 if out of bounds.
func PositionToOffset(content string, line, character int) int {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return -1
	}

	offset := 0
	for i := range line {
		offset += len(lines[i]) + 1 // +1 for \n
	}

	// character is utf-16 code units; walk the line rune-by-rune.
	lineStr := lines[line]
	col := 0
	for i, r := range lineStr {
		if col >= character {
			return offset + i
		}
		if r > 0xFFFF {
			col += 2 // surrogate pair
		} else {
			col++
		}
	}

	return offset + len(lineStr)
}

// OffsetToPosition converts a byte offset to an LSP line/character pair.
func OffsetToPosition(content string, byteOffset int) (line, character int) {
	if byteOffset < 0 {
		return 0, 0
	}

	col := 0
	ln := 0
	for i, r := range content {
		if i == byteOffset {
			return ln, col
		}
		if r == '\n' {
			ln++
			col = 0
		} else if r > 0xFFFF {
			col += 2
		} else {
			col++
		}
	}
	return ln, col
}

// MarshalPosition returns a JSON-ready position object.
func MarshalPosition(line, character int) json.RawMessage {
	b, _ := json.Marshal(map[string]int{"line": line, "character": character})
	return b
}

func lineTextAt(content string, line int) (string, bool) {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return "", false
	}
	return lines[line], true
}
