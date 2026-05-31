package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// TextEdit is the subset of LSP text edit fields used by formatting.
type TextEdit struct {
	Range struct {
		Start struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"start"`
		End struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"end"`
	} `json:"range"`
	NewText string `json:"newText"`
}

// FormatResult describes the post-format content and whether it changed.
type FormatResult struct {
	Content string `json:"content"`
	Changed bool   `json:"changed"`
}

func formatOptions() map[string]any {
	return map[string]any{
		"tabSize":      4,
		"insertSpaces": true,
	}
}

// RequestFormatting sends textDocument/formatting and applies returned edits.
func RequestFormatting(ctx context.Context, t *Transport, uri, content string, serverPosEnc PositionEncoding) (FormatResult, error) {
	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"options":      formatOptions(),
	}

	resp, err := t.Call(ctx, "textDocument/formatting", params, 1500*time.Millisecond)
	if err != nil {
		return FormatResult{}, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return FormatResult{Content: content, Changed: false}, nil
	}

	var edits []TextEdit
	if err := json.Unmarshal(resp.Result, &edits); err != nil {
		return FormatResult{}, fmt.Errorf("lsp: parse formatting payload: %w", err)
	}
	edits = convertTextEdits(content, edits, serverPosEnc, PositionEncodingUTF16)

	updated := ApplyTextEdits(content, edits)
	return FormatResult{Content: updated, Changed: updated != content}, nil
}

// RequestRangeFormatting sends textDocument/rangeFormatting and applies returned edits.
func RequestRangeFormatting(ctx context.Context, t *Transport, uri, content string, startLine, startCharacter, endLine, endCharacter int, serverPosEnc PositionEncoding) (FormatResult, error) {
	serverStartChar := convertCharacterAtLine(content, startLine, startCharacter, PositionEncodingUTF16, serverPosEnc)
	serverEndChar := convertCharacterAtLine(content, endLine, endCharacter, PositionEncodingUTF16, serverPosEnc)

	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"range": map[string]any{
			"start": map[string]int{"line": startLine, "character": serverStartChar},
			"end":   map[string]int{"line": endLine, "character": serverEndChar},
		},
		"options": formatOptions(),
	}

	resp, err := t.Call(ctx, "textDocument/rangeFormatting", params, 1500*time.Millisecond)
	if err != nil {
		return FormatResult{}, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return FormatResult{Content: content, Changed: false}, nil
	}

	var edits []TextEdit
	if err := json.Unmarshal(resp.Result, &edits); err != nil {
		return FormatResult{}, fmt.Errorf("lsp: parse rangeFormatting payload: %w", err)
	}
	edits = convertTextEdits(content, edits, serverPosEnc, PositionEncodingUTF16)

	updated := ApplyTextEdits(content, edits)
	return FormatResult{Content: updated, Changed: updated != content}, nil
}

// ApplyTextEdits applies LSP text edits to content.
func ApplyTextEdits(content string, edits []TextEdit) string {
	if len(edits) == 0 {
		return content
	}

	type spanEdit struct {
		start   int
		end     int
		newText string
	}

	spans := make([]spanEdit, 0, len(edits))
	for _, edit := range edits {
		start := PositionToOffset(content, edit.Range.Start.Line, edit.Range.Start.Character)
		end := PositionToOffset(content, edit.Range.End.Line, edit.Range.End.Character)
		if start < 0 || end < 0 || start > end || end > len(content) {
			continue
		}
		spans = append(spans, spanEdit{start: start, end: end, newText: edit.NewText})
	}
	if len(spans) == 0 {
		return content
	}

	// Apply from end -> start so offsets remain valid.
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start == spans[j].start {
			return spans[i].end > spans[j].end
		}
		return spans[i].start > spans[j].start
	})

	out := content
	for _, edit := range spans {
		out = out[:edit.start] + edit.newText + out[edit.end:]
	}

	return out
}
