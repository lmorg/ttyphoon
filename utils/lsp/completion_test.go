package lsp

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestParseCompletionResult_DirectArray(t *testing.T) {
	raw := json.RawMessage(`[{"label":"Println","detail":"fmt","insertText":"Println"}]`)
	items, err := parseCompletionResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Label != "Println" {
		t.Fatalf("unexpected label: %q", items[0].Label)
	}
}

func TestParseCompletionResult_CompletionList(t *testing.T) {
	raw := json.RawMessage(`{"isIncomplete":true,"items":[{"label":"fmt","detail":"package"}]}`)
	items, err := parseCompletionResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].InsertText != "fmt" {
		t.Fatalf("expected insertText fallback, got %q", items[0].InsertText)
	}
}

func TestNormalizeCompletionItems_CapsPayload(t *testing.T) {
	input := make([]CompletionItem, 0, maxCompletionItems+50)
	for i := 0; i < maxCompletionItems+50; i++ {
		input = append(input, CompletionItem{Label: fmt.Sprintf("item-%03d", i)})
	}

	out := normalizeCompletionItems(input)
	if len(out) != maxCompletionItems {
		t.Fatalf("item count = %d, want %d", len(out), maxCompletionItems)
	}
	if out[0].Label != "item-000" {
		t.Fatalf("first item label = %q, want item-000", out[0].Label)
	}
	if out[maxCompletionItems-1].Label != fmt.Sprintf("item-%03d", maxCompletionItems-1) {
		t.Fatalf("last kept item label = %q", out[maxCompletionItems-1].Label)
	}
}

func TestNormalizeCompletionItems_CapsAfterFilteringInvalidLabels(t *testing.T) {
	input := make([]CompletionItem, 0, maxCompletionItems*2)
	for i := 0; i < maxCompletionItems*2; i++ {
		if i%2 == 0 {
			input = append(input, CompletionItem{})
			continue
		}
		input = append(input, CompletionItem{Label: fmt.Sprintf("ok-%03d", i)})
	}

	out := normalizeCompletionItems(input)
	if len(out) != maxCompletionItems {
		t.Fatalf("item count = %d, want %d", len(out), maxCompletionItems)
	}
	for i := range out {
		if out[i].Label == "" {
			t.Fatalf("unexpected empty label at index %d", i)
		}
		if out[i].InsertText == "" {
			t.Fatalf("expected insertText fallback at index %d", i)
		}
	}
}
