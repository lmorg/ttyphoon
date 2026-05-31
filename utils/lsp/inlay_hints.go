package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// InlayHintItem is a flattened, frontend-ready inlay hint entry.
type InlayHintItem struct {
	Label        string `json:"label"`
	Tooltip      string `json:"tooltip,omitempty"`
	Kind         int    `json:"kind,omitempty"`
	Line         int    `json:"line"`
	Character    int    `json:"character"`
	PaddingLeft  bool   `json:"paddingLeft,omitempty"`
	PaddingRight bool   `json:"paddingRight,omitempty"`
}

type inlayHintWire struct {
	Position     positionWire    `json:"position"`
	Label        json.RawMessage `json:"label"`
	Tooltip      json.RawMessage `json:"tooltip,omitempty"`
	Kind         int             `json:"kind,omitempty"`
	PaddingLeft  bool            `json:"paddingLeft,omitempty"`
	PaddingRight bool            `json:"paddingRight,omitempty"`
}

type inlayHintLabelPartWire struct {
	Value string `json:"value"`
}

// RequestInlayHints sends textDocument/inlayHint for the current document range.
func RequestInlayHints(ctx context.Context, t *Transport, uri, content string, serverPosEnc PositionEncoding) ([]InlayHintItem, error) {
	lines := strings.Split(content, "\n")
	endLine := len(lines) - 1
	if endLine < 0 {
		endLine = 0
	}

	endCharacter := 0
	if len(lines) > 0 {
		endCharacter = len(lines[endLine])
	}

	serverEndCharacter := convertCharacterAtLine(content, endLine, endCharacter, PositionEncodingUTF16, serverPosEnc)

	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"range": map[string]any{
			"start": map[string]int{"line": 0, "character": 0},
			"end":   map[string]int{"line": endLine, "character": serverEndCharacter},
		},
	}

	resp, err := t.Call(ctx, "textDocument/inlayHint", params, 1500*time.Millisecond)
	if err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code == -32601 {
			return nil, nil
		}
		return nil, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil, nil
	}

	items, err := parseInlayHintsResult(resp.Result, content, serverPosEnc)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	return items, nil
}

func parseInlayHintsResult(raw json.RawMessage, content string, serverPosEnc PositionEncoding) ([]InlayHintItem, error) {
	var hints []inlayHintWire
	if err := json.Unmarshal(raw, &hints); err != nil {
		return nil, fmt.Errorf("lsp: parse inlayHint payload: %w", err)
	}

	out := make([]InlayHintItem, 0, len(hints))
	for _, hint := range hints {
		label := inlayHintLabelText(hint.Label)
		if label == "" {
			continue
		}

		character := hint.Position.Character
		if serverPosEnc != PositionEncodingUTF16 {
			character = convertCharacterAtLine(content, hint.Position.Line, character, serverPosEnc, PositionEncodingUTF16)
		}

		out = append(out, InlayHintItem{
			Label:        label,
			Tooltip:      HoverTextFromContents(hint.Tooltip),
			Kind:         hint.Kind,
			Line:         hint.Position.Line,
			Character:    character,
			PaddingLeft:  hint.PaddingLeft,
			PaddingRight: hint.PaddingRight,
		})
	}

	return out, nil
}

func inlayHintLabelText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var direct string
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct
	}

	var parts []inlayHintLabelPartWire
	if err := json.Unmarshal(raw, &parts); err == nil {
		var b strings.Builder
		for _, part := range parts {
			b.WriteString(part.Value)
		}
		return b.String()
	}

	return ""
}
