package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SemanticTokensLegend names the token types and modifiers a server emits, in
// the index order used by the token data stream.
type SemanticTokensLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

// SemanticTokensResult carries the server's relative-encoded token stream
// through to Monaco unchanged, alongside the legend needed to decode it.
type SemanticTokensResult struct {
	Legend   SemanticTokensLegend `json:"legend"`
	Data     []int                `json:"data,omitempty"`
	ResultID string               `json:"resultId,omitempty"`
	Edits    []SemanticTokenEdit  `json:"edits,omitempty"`
}

// SemanticTokenEdit models an incremental semantic token delta edit.
type SemanticTokenEdit struct {
	Start       int   `json:"start"`
	DeleteCount int   `json:"deleteCount"`
	Data        []int `json:"data,omitempty"`
}

type semanticTokensWire struct {
	Data []int `json:"data"`
}

type semanticTokensDeltaWire struct {
	ResultID string                  `json:"resultId,omitempty"`
	Edits    []semanticTokenEditWire `json:"edits,omitempty"`
}

type semanticTokenEditWire struct {
	Start       int   `json:"start"`
	DeleteCount int   `json:"deleteCount"`
	Data        []int `json:"data,omitempty"`
}

// RequestSemanticTokens sends textDocument/semanticTokens/full and returns the
// raw relative-encoded data. Character offsets are converted to UTF-16 when the
// server negotiated another encoding, because Monaco columns are UTF-16.
func RequestSemanticTokens(ctx context.Context, t *Transport, uri, content string, serverPosEnc PositionEncoding, legend SemanticTokensLegend) (*SemanticTokensResult, error) {
	if len(legend.TokenTypes) == 0 {
		return nil, nil
	}

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
	// A server that supports semantic tokens may still have none to report yet, so
	// keep returning the legend: the caller registers its provider from it.
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return &SemanticTokensResult{Legend: legend}, nil
	}

	var payload semanticTokensWire
	if err := json.Unmarshal(resp.Result, &payload); err != nil {
		return nil, fmt.Errorf("lsp: parse semanticTokens payload: %w", err)
	}
	if len(payload.Data) == 0 {
		return &SemanticTokensResult{Legend: legend}, nil
	}
	if len(payload.Data)%5 != 0 {
		return nil, fmt.Errorf("lsp: invalid semanticTokens payload length")
	}

	data := payload.Data
	if serverPosEnc != PositionEncodingUTF16 {
		data = convertSemanticTokensToUTF16(data, content, serverPosEnc)
	}

	return &SemanticTokensResult{Legend: legend, Data: data}, nil
}

// RequestSemanticTokensDelta sends textDocument/semanticTokens/full/delta and
// returns the delta payload expected by Monaco's incremental provider. When a
// server does not support the delta form, it falls back to the full request.
func RequestSemanticTokensDelta(ctx context.Context, t *Transport, uri, content string, serverPosEnc PositionEncoding, legend SemanticTokensLegend, previousResultID string) (*SemanticTokensResult, error) {
	if len(legend.TokenTypes) == 0 {
		return nil, nil
	}

	params := map[string]any{
		"textDocument":     map[string]any{"uri": uri},
		"previousResultId": previousResultID,
	}

	resp, err := t.Call(ctx, "textDocument/semanticTokens/full/delta", params, 1500*time.Millisecond)
	if err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code == -32601 {
			return RequestSemanticTokens(ctx, t, uri, content, serverPosEnc, legend)
		}
		return nil, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return &SemanticTokensResult{Legend: legend}, nil
	}

	var payload semanticTokensDeltaWire
	if err := json.Unmarshal(resp.Result, &payload); err != nil {
		return nil, fmt.Errorf("lsp: parse semanticTokens delta payload: %w", err)
	}

	result := &SemanticTokensResult{Legend: legend, ResultID: payload.ResultID}
	for _, edit := range payload.Edits {
		if edit.DeleteCount < 0 || edit.Start < 0 {
			continue
		}
		data := edit.Data
		if serverPosEnc != PositionEncodingUTF16 {
			data = convertSemanticTokensToUTF16(data, content, serverPosEnc)
		}
		result.Edits = append(result.Edits, SemanticTokenEdit{
			Start:       edit.Start,
			DeleteCount: edit.DeleteCount,
			Data:        data,
		})
	}
	if len(result.Edits) == 0 && len(payload.Edits) == 0 {
		return &SemanticTokensResult{Legend: legend, ResultID: payload.ResultID}, nil
	}
	return result, nil
}

// convertSemanticTokensToUTF16 re-encodes character offsets and lengths while
// keeping the relative encoding, so each delta stays valid for the next token.
func convertSemanticTokensToUTF16(data []int, content string, from PositionEncoding) []int {
	lineCount := strings.Count(content, "\n") + 1

	out := make([]int, 0, len(data))
	var line, char, prevLine, prevChar int

	for i := 0; i+4 < len(data); i += 5 {
		deltaLine := data[i]
		deltaStart := data[i+1]
		length := data[i+2]

		if deltaLine < 0 || deltaStart < 0 || length < 0 {
			continue
		}

		line += deltaLine
		if deltaLine > 0 {
			char = deltaStart
		} else {
			char += deltaStart
		}

		if line < 0 || line >= lineCount {
			continue
		}

		startUTF16 := convertCharacterAtLine(content, line, char, from, PositionEncodingUTF16)
		endUTF16 := convertCharacterAtLine(content, line, char+length, from, PositionEncodingUTF16)
		if endUTF16 <= startUTF16 {
			continue
		}

		outDeltaLine := line - prevLine
		outDeltaStart := startUTF16
		if outDeltaLine == 0 {
			outDeltaStart = startUTF16 - prevChar
		}
		if outDeltaStart < 0 {
			continue
		}

		out = append(out, outDeltaLine, outDeltaStart, endUTF16-startUTF16, data[i+3], data[i+4])
		prevLine = line
		prevChar = startUTF16
	}

	return out
}
