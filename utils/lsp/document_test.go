package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// URI helpers
// ---------------------------------------------------------------------------

func TestFilePathToURI_unix(t *testing.T) {
	got := FilePathToURI("/home/user/project/main.go")
	const want = "file:///home/user/project/main.go"
	if got != want {
		t.Errorf("FilePathToURI = %q, want %q", got, want)
	}
}

func TestURIToFilePath_roundtrip(t *testing.T) {
	original := "/home/user/project/notes.md"
	uri := FilePathToURI(original)
	got, err := URIToFilePath(uri)
	if err != nil {
		t.Fatalf("URIToFilePath: %v", err)
	}
	if got != original {
		t.Errorf("round-trip: got %q, want %q", got, original)
	}
}

func TestURIToFilePath_nonFileScheme(t *testing.T) {
	_, err := URIToFilePath("https://example.com/file.go")
	if err == nil {
		t.Fatal("expected error for non-file URI, got nil")
	}
}

// ---------------------------------------------------------------------------
// Document
// ---------------------------------------------------------------------------

func TestDocument_versionIncrementsOnUpdate(t *testing.T) {
	d := newDocument("/a/b/main.go", "go", "package main")
	if d.Version() != 1 {
		t.Fatalf("initial version = %d, want 1", d.Version())
	}
	v := d.update("package main\n\nfunc main() {}")
	if v != 2 {
		t.Fatalf("after update version = %d, want 2", v)
	}
}

func TestDocument_closeMarksAsNotOpen(t *testing.T) {
	d := newDocument("/a/b/main.go", "go", "")
	d.close()
	if d.IsOpen() {
		t.Fatal("expected IsOpen() = false after close")
	}
}

// ---------------------------------------------------------------------------
// DocumentStore with transport spy
// ---------------------------------------------------------------------------

// notifySpy captures Notify calls sent through the transport.
type notifySpy struct {
	methods []string
	params  []json.RawMessage
}

func newSpyTransport() (*Transport, *notifySpy, io.Closer) {
	spy := &notifySpy{}
	pr, pw := io.Pipe()
	t := NewTransport(pw, strings.NewReader("")) // write to pw, never read from server

	// Drain the write side so writes don't block.
	go func() {
		r := bufio.NewReader(pr)
		for {
			msg, err := ReadMessage(r)
			if err != nil {
				return
			}
			spy.methods = append(spy.methods, msg.Method)
			spy.params = append(spy.params, msg.Params)
		}
	}()

	return t, spy, pw
}

func TestDocumentStore_didOpenSendsNotification(t *testing.T) {
	tr, spy, closer := newSpyTransport()
	defer closer.Close()

	ds := NewDocumentStore()
	ctx := context.Background()

	if err := ds.DidOpen(ctx, tr, "/proj/main.go", "go", "package main"); err != nil {
		t.Fatalf("DidOpen: %v", err)
	}

	// Allow goroutine to drain.
	waitForSpy(t, spy, 1)

	if len(spy.methods) == 0 || spy.methods[0] != "textDocument/didOpen" {
		t.Errorf("expected textDocument/didOpen, got %v", spy.methods)
	}

	if !ds.IsOpen("/proj/main.go") {
		t.Error("document should be open after DidOpen")
	}
}

func TestDocumentStore_didChangeSendsNotificationAndIncrementsVersion(t *testing.T) {
	tr, spy, closer := newSpyTransport()
	defer closer.Close()

	ds := NewDocumentStore()
	ctx := context.Background()

	_ = ds.DidOpen(ctx, tr, "/proj/main.go", "go", "package main")
	waitForSpy(t, spy, 1)

	newContent := "package main\n\nfunc main() {}"
	if err := ds.DidChange(ctx, tr, "/proj/main.go", newContent); err != nil {
		t.Fatalf("DidChange: %v", err)
	}

	waitForSpy(t, spy, 2)

	if spy.methods[1] != "textDocument/didChange" {
		t.Errorf("expected textDocument/didChange, got %q", spy.methods[1])
	}

	doc := ds.Get("/proj/main.go")
	if doc == nil {
		t.Fatal("doc is nil after DidChange")
	}
	if doc.Version() != 2 {
		t.Errorf("version after DidChange = %d, want 2", doc.Version())
	}
}

func TestDocumentStore_didCloseSendsNotificationAndRemovesDoc(t *testing.T) {
	tr, spy, closer := newSpyTransport()
	defer closer.Close()

	ds := NewDocumentStore()
	ctx := context.Background()

	_ = ds.DidOpen(ctx, tr, "/proj/main.go", "go", "package main")
	waitForSpy(t, spy, 1)
	_ = ds.DidClose(ctx, tr, "/proj/main.go")
	waitForSpy(t, spy, 2)

	if spy.methods[1] != "textDocument/didClose" {
		t.Errorf("expected textDocument/didClose, got %q", spy.methods[1])
	}
	if ds.IsOpen("/proj/main.go") {
		t.Error("document should not be open after DidClose")
	}
}

func TestDocumentStore_didCloseTwiceIsIdempotent(t *testing.T) {
	tr, _, closer := newSpyTransport()
	defer closer.Close()

	ds := NewDocumentStore()
	ctx := context.Background()

	_ = ds.DidOpen(ctx, tr, "/proj/main.go", "go", "")
	if err := ds.DidClose(ctx, tr, "/proj/main.go"); err != nil {
		t.Fatalf("first DidClose: %v", err)
	}
	if err := ds.DidClose(ctx, tr, "/proj/main.go"); err != nil {
		t.Fatalf("second DidClose should be idempotent: %v", err)
	}
}

func TestDocumentStore_didChangeUnknownDocReturnsError(t *testing.T) {
	tr, _, closer := newSpyTransport()
	defer closer.Close()

	ds := NewDocumentStore()
	err := ds.DidChange(context.Background(), tr, "/nonexistent.go", "")
	if err == nil {
		t.Fatal("expected error for DidChange on unopened document")
	}
}

// ---------------------------------------------------------------------------
// Position helpers
// ---------------------------------------------------------------------------

func TestPositionToOffset_simpleASCII(t *testing.T) {
	content := "hello\nworld"
	// line 1, character 3 => "ld" starts there; byte offset = len("hello\n") + 3 = 9
	got := PositionToOffset(content, 1, 3)
	if got != 9 {
		t.Errorf("PositionToOffset(1,3) = %d, want 9", got)
	}
}

func TestPositionToOffset_outOfBounds(t *testing.T) {
	got := PositionToOffset("hello", 99, 0)
	if got != -1 {
		t.Errorf("out-of-bounds line should return -1, got %d", got)
	}
}

func TestOffsetToPosition_roundtrip(t *testing.T) {
	content := "line one\nline two\nline three"
	offset := strings.Index(content, "two")
	line, char := OffsetToPosition(content, offset)
	got := PositionToOffset(content, line, char)
	if got != offset {
		t.Errorf("round-trip offset: got %d, want %d", got, offset)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func waitForSpy(t *testing.T, spy *notifySpy, wantCount int) {
	t.Helper()
	// Busy-poll up to 200 iterations of 1ms each — sufficient for in-process pipes.
	for i := 0; i < 200; i++ {
		if len(spy.methods) >= wantCount {
			return
		}
		// Tiny yield; avoids importing time just for this.
		for j := 0; j < 1e5; j++ {
		}
	}
	t.Fatalf("spy: got %d notifications, want %d", len(spy.methods), wantCount)
}
