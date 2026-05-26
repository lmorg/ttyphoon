package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// RenameResult describes the post-rename content and whether it changed.
type RenameResult struct {
	Content string `json:"content"`
	Changed bool   `json:"changed"`
}

// PrepareRenameResult describes whether rename is valid at cursor and a suggested placeholder.
type PrepareRenameResult struct {
	CanRename   bool   `json:"canRename"`
	Placeholder string `json:"placeholder,omitempty"`
}

type prepareRenameWire struct {
	Range           *rangeWire `json:"range,omitempty"`
	Placeholder     string     `json:"placeholder,omitempty"`
	DefaultBehavior bool       `json:"defaultBehavior,omitempty"`
}

// RequestPrepareRename sends textDocument/prepareRename.
func RequestPrepareRename(ctx context.Context, t *Transport, uri, content string, line, character int, serverPosEnc PositionEncoding) (PrepareRenameResult, error) {
	serverChar := convertCharacterAtLine(content, line, character, PositionEncodingUTF16, serverPosEnc)

	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]int{"line": line, "character": serverChar},
	}

	resp, err := t.Call(ctx, "textDocument/prepareRename", params, 1500*time.Millisecond)
	if err != nil {
		return PrepareRenameResult{}, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return PrepareRenameResult{}, nil
	}

	return parsePrepareRenameResult(resp.Result)
}

// RequestRename sends textDocument/rename and applies returned edits for the active URI.
func RequestRename(ctx context.Context, t *Transport, uri, content string, line, character int, newName string, serverPosEnc PositionEncoding) (RenameResult, error) {
	serverChar := convertCharacterAtLine(content, line, character, PositionEncodingUTF16, serverPosEnc)

	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]int{"line": line, "character": serverChar},
		"newName":      newName,
	}

	resp, err := t.Call(ctx, "textDocument/rename", params, 1500*time.Millisecond)
	if err != nil {
		return RenameResult{}, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return RenameResult{Content: content, Changed: false}, nil
	}

	var edit workspaceEditWire
	if err := json.Unmarshal(resp.Result, &edit); err != nil {
		return RenameResult{}, fmt.Errorf("lsp: parse rename payload: %w", err)
	}

	edits := edit.editsForURI(uri)
	if len(edits) == 0 {
		return RenameResult{Content: content, Changed: false}, nil
	}
	edits = convertTextEdits(content, edits, serverPosEnc, PositionEncodingUTF16)

	updated := ApplyTextEdits(content, edits)
	return RenameResult{Content: updated, Changed: updated != content}, nil
}

func parsePrepareRenameResult(raw json.RawMessage) (PrepareRenameResult, error) {
	var withMeta prepareRenameWire
	if err := json.Unmarshal(raw, &withMeta); err == nil {
		if withMeta.DefaultBehavior || withMeta.Range != nil {
			return PrepareRenameResult{
				CanRename:   true,
				Placeholder: withMeta.Placeholder,
			}, nil
		}
	}

	var directRange map[string]json.RawMessage
	if err := json.Unmarshal(raw, &directRange); err == nil {
		_, hasStart := directRange["start"]
		_, hasEnd := directRange["end"]
		if hasStart && hasEnd {
			return PrepareRenameResult{CanRename: true}, nil
		}
	}

	return PrepareRenameResult{}, fmt.Errorf("lsp: parse prepareRename payload")
}
