package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RequestHover sends textDocument/hover and returns a flattened text payload.
func RequestHover(ctx context.Context, t *Transport, uri, content string, line, character int, serverPosEnc PositionEncoding) (string, error) {
	serverChar := convertCharacterAtLine(content, line, character, PositionEncodingUTF16, serverPosEnc)

	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]int{"line": line, "character": serverChar},
	}

	resp, err := t.Call(ctx, "textDocument/hover", params, 1200*time.Millisecond)
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return "", nil
	}

	var payload struct {
		Contents json.RawMessage `json:"contents"`
	}
	if err := json.Unmarshal(resp.Result, &payload); err != nil {
		return "", fmt.Errorf("lsp: parse hover payload: %w", err)
	}

	return HoverTextFromContents(payload.Contents), nil
}

// HoverTextFromContents flattens LSP hover contents into plain text.
func HoverTextFromContents(contents json.RawMessage) string {
	if len(contents) == 0 || string(contents) == "null" {
		return ""
	}

	var value any
	if err := json.Unmarshal(contents, &value); err != nil {
		return ""
	}

	parts := flattenHoverValue(value)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	return strings.TrimSpace(strings.Join(filterNonEmpty(parts), "\n\n"))
}

func flattenHoverValue(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case map[string]any:
		// MarkedString: { language, value }
		if lang, ok := x["language"].(string); ok {
			if value, ok := x["value"].(string); ok {
				if strings.TrimSpace(lang) == "" {
					return []string{value}
				}
				return []string{"```" + lang + "\n" + value + "\n```"}
			}
		}
		// MarkupContent: { kind, value }
		if value, ok := x["value"].(string); ok {
			return []string{value}
		}
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			out = append(out, flattenHoverValue(item)...)
		}
		return out
	}
	return nil
}

func filterNonEmpty(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			out = append(out, part)
		}
	}
	return out
}
