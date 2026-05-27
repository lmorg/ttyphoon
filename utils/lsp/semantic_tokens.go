package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SemanticTokenItem is a frontend-ready semantic token entry.
type SemanticTokenItem struct {
	Line      int `json:"line"`
	Character int `json:"character"`
	Length    int `json:"length"`
	TokenType int `json:"tokenType"`
	TokenMods int `json:"tokenModifiers"`
}

type semanticTokensWire struct {
	Data []int `json:"data"`
}

// RequestSemanticTokens sends textDocument/semanticTokens/full and normalizes tokens.
func RequestSemanticTokens(ctx context.Context, t *Transport, uri, content string, serverPosEnc PositionEncoding) ([]SemanticTokenItem, error) {
	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
	}

	resp, err := t.Call(ctx, "textDocument/semanticTokens/full", params, 1500*time.Millisecond)
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

	items, err := parseSemanticTokensResult(resp.Result, content, serverPosEnc)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	return items, nil
}

func parseSemanticTokensResult(raw json.RawMessage, content string, serverPosEnc PositionEncoding) ([]SemanticTokenItem, error) {
	var payload semanticTokensWire
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("lsp: parse semanticTokens payload: %w", err)
	}

	if len(payload.Data) == 0 {
		return nil, nil
	}
	if len(payload.Data)%5 != 0 {
		return nil, fmt.Errorf("lsp: invalid semanticTokens payload length")
	}

	lines := strings.Split(content, "\n")
	items := make([]SemanticTokenItem, 0, len(payload.Data)/5)

	line := 0
	character := 0
	for i := 0; i < len(payload.Data); i += 5 {
		deltaLine := payload.Data[i]
		deltaStart := payload.Data[i+1]
		rawLength := payload.Data[i+2]
		tokenType := payload.Data[i+3]
		tokenMods := payload.Data[i+4]

		if deltaLine < 0 || deltaStart < 0 || rawLength < 0 {
			continue
		}

		line += deltaLine
		if deltaLine > 0 {
			character = deltaStart
		} else {
			character += deltaStart
		}

		if line < 0 || line >= len(lines) {
			continue
		}

		lineLen := len(lines[line])
		if character < 0 || character > lineLen {
			continue
		}

		endCharacter := character + rawLength
		if endCharacter > lineLen {
			endCharacter = lineLen
		}
		if endCharacter <= character {
			continue
		}

		outChar := character
		outLen := endCharacter - character
		if serverPosEnc != PositionEncodingUTF16 {
			startUTF16 := convertCharacterAtLine(content, line, character, serverPosEnc, PositionEncodingUTF16)
			endUTF16 := convertCharacterAtLine(content, line, endCharacter, serverPosEnc, PositionEncodingUTF16)
			if endUTF16 <= startUTF16 {
				continue
			}
			outChar = startUTF16
			outLen = endUTF16 - startUTF16
		}

		items = append(items, SemanticTokenItem{
			Line:      line,
			Character: outChar,
			Length:    outLen,
			TokenType: tokenType,
			TokenMods: tokenMods,
		})
	}

	if len(items) == 0 {
		return nil, nil
	}

	return items, nil
}
