package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DefinitionLocation is a normalized textDocument/definition target.
type DefinitionLocation struct {
	URI       string `json:"uri"`
	FilePath  string `json:"filePath,omitempty"`
	Line      int    `json:"line"`
	Character int    `json:"character"`
}

// RequestDefinition sends textDocument/definition and normalizes the result payload.
func RequestDefinition(ctx context.Context, t *Transport, uri, content string, line, character int, serverPosEnc PositionEncoding, contentForURI func(uri string) (string, bool)) ([]DefinitionLocation, error) {
	serverChar := convertCharacterAtLine(content, line, character, PositionEncodingUTF16, serverPosEnc)

	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]int{"line": line, "character": serverChar},
	}

	resp, err := t.Call(ctx, "textDocument/definition", params, 1200*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil, nil
	}

	locations, err := parseDefinitionResult(resp.Result)
	if err != nil {
		return nil, err
	}

	if serverPosEnc != PositionEncodingUTF16 {
		for i := range locations {
			locContent, ok := "", false
			if contentForURI != nil {
				locContent, ok = contentForURI(locations[i].URI)
			}
			if !ok {
				continue
			}
			locations[i].Character = convertCharacterAtLine(locContent, locations[i].Line, locations[i].Character, serverPosEnc, PositionEncodingUTF16)
		}
	}

	return locations, nil
}

func parseDefinitionResult(raw json.RawMessage) ([]DefinitionLocation, error) {
	var single definitionLocationWire
	if err := json.Unmarshal(raw, &single); err == nil && single.URI != "" {
		return normalizeDefinitionLocations([]definitionLocationWire{single}), nil
	}

	var list []definitionLocationWire
	if err := json.Unmarshal(raw, &list); err == nil {
		locations := normalizeDefinitionLocations(list)
		if len(locations) > 0 {
			return locations, nil
		}
	}

	var linkList []definitionLinkWire
	if err := json.Unmarshal(raw, &linkList); err == nil {
		mapped := make([]definitionLocationWire, 0, len(linkList))
		for _, link := range linkList {
			uri := link.TargetURI
			start := link.TargetSelectionRange.Start
			if link.TargetSelectionRange.Start == (positionWire{}) {
				start = link.TargetRange.Start
			}
			mapped = append(mapped, definitionLocationWire{
				URI: uri,
				Range: rangeWire{
					Start: start,
				},
			})
		}
		return normalizeDefinitionLocations(mapped), nil
	}

	return nil, fmt.Errorf("lsp: parse definition payload")
}

type definitionLocationWire struct {
	URI   string    `json:"uri"`
	Range rangeWire `json:"range"`
}

type definitionLinkWire struct {
	TargetURI            string    `json:"targetUri"`
	TargetRange          rangeWire `json:"targetRange"`
	TargetSelectionRange rangeWire `json:"targetSelectionRange"`
}

type rangeWire struct {
	Start positionWire `json:"start"`
	End   positionWire `json:"end"`
}

type positionWire struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

func normalizeDefinitionLocations(items []definitionLocationWire) []DefinitionLocation {
	if len(items) == 0 {
		return nil
	}

	out := make([]DefinitionLocation, 0, len(items))
	for _, item := range items {
		if item.URI == "" {
			continue
		}
		loc := DefinitionLocation{
			URI:       item.URI,
			Line:      item.Range.Start.Line,
			Character: item.Range.Start.Character,
		}
		if path, err := URIToFilePath(item.URI); err == nil {
			loc.FilePath = path
		}
		out = append(out, loc)
	}
	if len(out) == 0 {
		return nil
	}

	return out
}
