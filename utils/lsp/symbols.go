package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DocumentSymbolItem is a flattened, frontend-ready symbol entry.
type DocumentSymbolItem struct {
	Name          string `json:"name"`
	Detail        string `json:"detail,omitempty"`
	Kind          int    `json:"kind"`
	Line          int    `json:"line"`
	Character     int    `json:"character"`
	ContainerName string `json:"containerName,omitempty"`
}

type documentSymbolWire struct {
	Name           string               `json:"name"`
	Detail         string               `json:"detail,omitempty"`
	Kind           int                  `json:"kind"`
	Range          rangeWire            `json:"range"`
	SelectionRange rangeWire            `json:"selectionRange"`
	Children       []documentSymbolWire `json:"children,omitempty"`
}

type symbolInformationWire struct {
	Name          string `json:"name"`
	Kind          int    `json:"kind"`
	ContainerName string `json:"containerName,omitempty"`
	Location      struct {
		URI   string    `json:"uri"`
		Range rangeWire `json:"range"`
	} `json:"location"`
}

// RequestDocumentSymbols sends textDocument/documentSymbol and flattens results.
func RequestDocumentSymbols(ctx context.Context, t *Transport, uri, content string, serverPosEnc PositionEncoding) ([]DocumentSymbolItem, error) {
	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
	}

	resp, err := t.Call(ctx, "textDocument/documentSymbol", params, 1500*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil, nil
	}

	items, err := parseDocumentSymbolsResult(resp.Result, uri, content, serverPosEnc)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, nil
	}

	return items, nil
}

func parseDocumentSymbolsResult(raw json.RawMessage, uri, content string, serverPosEnc PositionEncoding) ([]DocumentSymbolItem, error) {
	var infos []symbolInformationWire
	if err := json.Unmarshal(raw, &infos); err == nil {
		out := make([]DocumentSymbolItem, 0, len(infos))
		hasLocationURI := false
		for _, info := range infos {
			if info.Name == "" {
				continue
			}
			if info.Location.URI != "" {
				hasLocationURI = true
			}
			if info.Location.URI != "" && info.Location.URI != uri {
				continue
			}
			if info.Location.URI == "" {
				continue
			}

			character := info.Location.Range.Start.Character
			if serverPosEnc != PositionEncodingUTF16 {
				character = convertCharacterAtLine(content, info.Location.Range.Start.Line, character, serverPosEnc, PositionEncodingUTF16)
			}

			out = append(out, DocumentSymbolItem{
				Name:          info.Name,
				Kind:          info.Kind,
				Line:          info.Location.Range.Start.Line,
				Character:     character,
				ContainerName: info.ContainerName,
			})
		}
		if hasLocationURI {
			return out, nil
		}
	}

	var documentSymbols []documentSymbolWire
	if err := json.Unmarshal(raw, &documentSymbols); err == nil {
		out := make([]DocumentSymbolItem, 0, len(documentSymbols))
		for _, symbol := range documentSymbols {
			flattenDocumentSymbol(&out, symbol, "", content, serverPosEnc)
		}
		return out, nil
	}

	return nil, fmt.Errorf("lsp: parse documentSymbol payload")
}

func flattenDocumentSymbol(out *[]DocumentSymbolItem, symbol documentSymbolWire, container, content string, serverPosEnc PositionEncoding) {
	if symbol.Name == "" {
		for _, child := range symbol.Children {
			flattenDocumentSymbol(out, child, container, content, serverPosEnc)
		}
		return
	}

	start := symbol.SelectionRange.Start
	if symbol.SelectionRange.Start == (positionWire{}) {
		start = symbol.Range.Start
	}

	character := start.Character
	if serverPosEnc != PositionEncodingUTF16 {
		character = convertCharacterAtLine(content, start.Line, character, serverPosEnc, PositionEncodingUTF16)
	}

	*out = append(*out, DocumentSymbolItem{
		Name:          symbol.Name,
		Detail:        symbol.Detail,
		Kind:          symbol.Kind,
		Line:          start.Line,
		Character:     character,
		ContainerName: container,
	})

	nextContainer := symbol.Name
	for _, child := range symbol.Children {
		flattenDocumentSymbol(out, child, nextContainer, content, serverPosEnc)
	}
}
