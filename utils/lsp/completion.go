package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const maxCompletionItems = 200

// CompletionItem is the frontend-ready subset of LSP completion item fields.
type CompletionItem struct {
	Label      string `json:"label"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insertText,omitempty"`
	Kind       int    `json:"kind,omitempty"`
	Deprecated bool   `json:"deprecated,omitempty"`
	Tags       []int  `json:"tags,omitempty"`
}

// RequestCompletion sends textDocument/completion and normalizes the result.
func RequestCompletion(ctx context.Context, t *Transport, uri, content string, line, character, triggerKind int, triggerChar string, serverPosEnc PositionEncoding) ([]CompletionItem, error) {
	completionContext := map[string]any{"triggerKind": triggerKind}
	if triggerKind == 2 && triggerChar != "" {
		completionContext["triggerCharacter"] = triggerChar
	}

	serverChar := convertCharacterAtLine(content, line, character, PositionEncodingUTF16, serverPosEnc)

	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]int{"line": line, "character": serverChar},
		"context":      completionContext,
	}

	resp, err := t.Call(ctx, "textDocument/completion", params, 1200*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil, nil
	}

	items, err := parseCompletionResult(resp.Result)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func parseCompletionResult(raw json.RawMessage) ([]CompletionItem, error) {
	var direct []CompletionItem
	if err := json.Unmarshal(raw, &direct); err == nil {
		return normalizeCompletionItems(direct), nil
	}

	var list struct {
		Items []CompletionItem `json:"items"`
	}
	if err := json.Unmarshal(raw, &list); err == nil {
		return normalizeCompletionItems(list.Items), nil
	}

	return nil, fmt.Errorf("lsp: parse completion payload")
}

func normalizeCompletionItems(items []CompletionItem) []CompletionItem {
	if len(items) == 0 {
		return nil
	}

	capacity := len(items)
	if capacity > maxCompletionItems {
		capacity = maxCompletionItems
	}

	out := make([]CompletionItem, 0, capacity)
	for _, item := range items {
		if item.Label == "" {
			continue
		}
		if !item.Deprecated {
			for _, tag := range item.Tags {
				if tag == 1 {
					item.Deprecated = true
					break
				}
			}
		}
		if item.InsertText == "" {
			item.InsertText = item.Label
		}
		out = append(out, item)
		if len(out) >= maxCompletionItems {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
