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

// WorkspaceSymbolItem is a frontend-ready workspace symbol entry.
type WorkspaceSymbolItem struct {
	Name          string `json:"name"`
	Detail        string `json:"detail,omitempty"`
	Kind          int    `json:"kind"`
	Line          int    `json:"line"`
	Character     int    `json:"character"`
	ContainerName string `json:"containerName,omitempty"`
	URI           string `json:"uri,omitempty"`
	FilePath      string `json:"filePath,omitempty"`
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
	Detail        string `json:"detail,omitempty"`
	Kind          int    `json:"kind"`
	ContainerName string `json:"containerName,omitempty"`
	Location      struct {
		URI   string    `json:"uri"`
		Range rangeWire `json:"range"`
	} `json:"location"`
}

type workspaceSymbolWire struct {
	Name          string `json:"name"`
	Detail        string `json:"detail,omitempty"`
	Kind          int    `json:"kind"`
	ContainerName string `json:"containerName,omitempty"`
	Location      struct {
		URI   string    `json:"uri,omitempty"`
		Range rangeWire `json:"range,omitempty"`
	} `json:"location"`
	URI            string    `json:"uri,omitempty"`
	Range          rangeWire `json:"range,omitempty"`
	SelectionRange rangeWire `json:"selectionRange,omitempty"`
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

// RequestWorkspaceSymbols sends workspace/symbol and normalizes results.
func RequestWorkspaceSymbols(ctx context.Context, t *Transport, query string, readFile func(uri string) (string, bool), serverPosEnc PositionEncoding) ([]WorkspaceSymbolItem, error) {
	params := map[string]any{
		"query": query,
	}

	resp, err := t.Call(ctx, "workspace/symbol", params, 1500*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil, nil
	}

	items, err := parseWorkspaceSymbolsResult(resp.Result, readFile, serverPosEnc)
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

func parseWorkspaceSymbolsResult(raw json.RawMessage, readFile func(uri string) (string, bool), serverPosEnc PositionEncoding) ([]WorkspaceSymbolItem, error) {
	var infos []symbolInformationWire
	if err := json.Unmarshal(raw, &infos); err == nil {
		out := make([]WorkspaceSymbolItem, 0, len(infos))
		for _, info := range infos {
			if info.Name == "" || info.Location.URI == "" {
				continue
			}

			content, ok := readFile(info.Location.URI)
			if !ok {
				continue
			}

			character := info.Location.Range.Start.Character
			if serverPosEnc != PositionEncodingUTF16 {
				character = convertCharacterAtLine(content, info.Location.Range.Start.Line, character, serverPosEnc, PositionEncodingUTF16)
			}

			filePath, err := URIToFilePath(info.Location.URI)
			if err != nil {
				filePath = ""
			}

			out = append(out, WorkspaceSymbolItem{
				Name:          info.Name,
				Detail:        info.Detail,
				Kind:          info.Kind,
				Line:          info.Location.Range.Start.Line,
				Character:     character,
				ContainerName: info.ContainerName,
				URI:           info.Location.URI,
				FilePath:      filePath,
			})
		}
		return out, nil
	}

	var symbols []workspaceSymbolWire
	if err := json.Unmarshal(raw, &symbols); err == nil {
		out := make([]WorkspaceSymbolItem, 0, len(symbols))
		for _, symbol := range symbols {
			uri := symbol.Location.URI
			start := symbol.Location.Range.Start
			if uri == "" {
				uri = symbol.URI
				if symbol.SelectionRange != (rangeWire{}) {
					start = symbol.SelectionRange.Start
				} else if symbol.Range != (rangeWire{}) {
					start = symbol.Range.Start
				}
			}
			if symbol.Name == "" || uri == "" {
				continue
			}

			content, ok := readFile(uri)
			if !ok {
				continue
			}

			character := start.Character
			if serverPosEnc != PositionEncodingUTF16 {
				character = convertCharacterAtLine(content, start.Line, character, serverPosEnc, PositionEncodingUTF16)
			}

			filePath, err := URIToFilePath(uri)
			if err != nil {
				filePath = ""
			}

			out = append(out, WorkspaceSymbolItem{
				Name:          symbol.Name,
				Detail:        symbol.Detail,
				Kind:          symbol.Kind,
				Line:          start.Line,
				Character:     character,
				ContainerName: symbol.ContainerName,
				URI:           uri,
				FilePath:      filePath,
			})
		}
		return out, nil
	}

	return nil, fmt.Errorf("lsp: parse workspace/symbol payload")
}
