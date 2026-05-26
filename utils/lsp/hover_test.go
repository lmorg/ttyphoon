package lsp

import (
	"encoding/json"
	"testing"
)

func TestHoverTextFromContents_String(t *testing.T) {
	raw := json.RawMessage(`"hello"`)
	if got := HoverTextFromContents(raw); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
}

func TestHoverTextFromContents_MarkedStringArray(t *testing.T) {
	raw := json.RawMessage(`[{"language":"go","value":"fmt.Println()"},"extra"]`)
	got := HoverTextFromContents(raw)
	want := "```go\nfmt.Println()\n```\n\nextra"
	if got != want {
		t.Fatalf("unexpected hover text\nwant: %q\ngot:  %q", want, got)
	}
}

func TestHoverTextFromContents_MarkupContent(t *testing.T) {
	raw := json.RawMessage(`{"kind":"markdown","value":"**symbol**"}`)
	if got := HoverTextFromContents(raw); got != "**symbol**" {
		t.Fatalf("expected markdown value, got %q", got)
	}
}
