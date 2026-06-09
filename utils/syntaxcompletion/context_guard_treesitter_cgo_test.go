//go:build cgo

package syntaxcompletion

import (
	"strings"
	"testing"
)

func TestTreeSitterBlocksInGoComment(t *testing.T) {
	eng := mustDefaultEngine(t)
	src := "package main\n// comment"
	res, err := eng.Complete(Request{Language: "go", Source: src, Cursor: len(src), Trigger: "["})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if res.Applied {
		t.Fatalf("did not expect completion inside go comment")
	}
}

func TestTreeSitterBlocksInGoString(t *testing.T) {
	eng := mustDefaultEngine(t)
	src := "package main\nvar s = \"abc"
	res, err := eng.Complete(Request{Language: "go", Source: src, Cursor: len(src), Trigger: "["})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if res.Applied {
		t.Fatalf("did not expect completion inside go string")
	}
}

func TestTreeSitterBlocksInHTMLComment(t *testing.T) {
	eng := mustDefaultEngine(t)
	src := "<!-- note -->"
	cursor := strings.Index(src, "note") + 2
	res, err := eng.Complete(Request{Language: "html", Source: src, Cursor: cursor, Trigger: "\""})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if res.Applied {
		t.Fatalf("did not expect completion inside html comment")
	}
}
