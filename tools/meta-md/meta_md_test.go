package metamd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentIncludesInlineCodeValues(t *testing.T) {
	doc := document(Values{
		Filename:   "note.md",
		SizeHuman:  "123 B",
		SizeBytes:  123,
		PathFull:   "/tmp/note.md",
		UserOwner:  "alice",
		GroupOwner: "staff",
		UnixOctal:  "0644",
		UserACL:    "rw-",
		GroupACL:   "r--",
		OtherACL:   "r--",
	})

	if strings.Contains(doc, "<pre>") {
		t.Fatalf("expected no pre blocks in markdown document")
	}

	required := []string{
		"# note.md",
		"- Size: `123 B` (`123` bytes)",
		"- Path:",
		"```",
		"/tmp/note.md",
		"- User:  `alice`",
		"- Group: `staff`",
		"- Unix:  `0644`",
	}

	for _, s := range required {
		if !strings.Contains(doc, s) {
			t.Fatalf("expected document to contain %q", s)
		}
	}
}

func TestDocumentEscapesInlineCode(t *testing.T) {
	doc := document(Values{
		Filename:  "meta.md",
		SizeHuman: "1 B",
		SizeBytes: 1,
		PathFull:  "C:\\\\tmp\\\\tick`path",
	})

	if !strings.Contains(doc, "- Path:") {
		t.Fatalf("expected path section in markdown, got: %s", doc)
	}

	if !strings.Contains(doc, "```") {
		t.Fatalf("expected fenced code block for path value, got: %s", doc)
	}

}

func TestDocumentForPathIncludesResolvedPath(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "note.md")

	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	doc := DocumentForPath(filePath)

	if !strings.Contains(doc, "# note.md") {
		t.Fatalf("expected file name heading in markdown: %s", doc)
	}

	if !strings.Contains(doc, "(`5` bytes)") {
		t.Fatalf("expected byte count section in size field: %s", doc)
	}

	if !strings.Contains(doc, "- Path:") {
		t.Fatalf("expected path heading in markdown: %s", doc)
	}

	if !strings.Contains(doc, filePath) {
		t.Fatalf("expected fenced path block to include resolved path: %s", doc)
	}
}
