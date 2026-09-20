package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ReferenceLocation is a normalized textDocument/references result.
type ReferenceLocation struct {
	URI       string   `json:"uri"`
	FilePath  string   `json:"filePath,omitempty"`
	Line      int      `json:"line"`
	Character int      `json:"character"`
	EndLine   int      `json:"endLine"`
	EndColumn int      `json:"endCharacter"`
	Context   []string `json:"context,omitempty"`
}

// RequestReferences requests all references for a symbol position.
func RequestReferences(ctx context.Context, t *Transport, uri, content string, line, character int, serverPosEnc PositionEncoding, contentForURI func(string) (string, bool)) ([]ReferenceLocation, error) {
	serverChar := convertCharacterAtLine(content, line, character, PositionEncodingUTF16, serverPosEnc)
	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]int{"line": line, "character": serverChar},
		"context":      map[string]bool{"includeDeclaration": true},
	}

	resp, err := t.Call(ctx, "textDocument/references", params, 1500*time.Millisecond)
	if err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code == -32601 {
			return nil, fmt.Errorf("lsp: references unsupported: %w", err)
		}
		return nil, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil, nil
	}

	var items []referenceLocationWire
	if err := json.Unmarshal(resp.Result, &items); err != nil {
		return nil, fmt.Errorf("lsp: parse references payload: %w", err)
	}

	out := make([]ReferenceLocation, 0, len(items))
	for _, item := range items {
		if item.URI == "" {
			continue
		}
		loc := ReferenceLocation{
			URI:       item.URI,
			Line:      item.Range.Start.Line,
			Character: item.Range.Start.Character,
			EndLine:   item.Range.End.Line,
			EndColumn: item.Range.End.Character,
		}
		if path, pathErr := URIToFilePath(item.URI); pathErr == nil {
			loc.FilePath = path
		}
		if serverPosEnc != PositionEncodingUTF16 && contentForURI != nil {
			if targetContent, ok := contentForURI(item.URI); ok {
				loc.Character = convertCharacterAtLine(targetContent, loc.Line, loc.Character, serverPosEnc, PositionEncodingUTF16)
				loc.EndColumn = convertCharacterAtLine(targetContent, loc.EndLine, loc.EndColumn, serverPosEnc, PositionEncodingUTF16)
			}
		}
		out = append(out, loc)
	}
	return out, nil
}

type referenceLocationWire struct {
	URI   string    `json:"uri"`
	Range rangeWire `json:"range"`
}
