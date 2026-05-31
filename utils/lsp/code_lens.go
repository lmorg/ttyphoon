package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// CodeLensItem is a frontend-ready summary of an LSP code lens.
type CodeLensItem struct {
	Index     int    `json:"index"`
	Title     string `json:"title"`
	Line      int    `json:"line"`
	Character int    `json:"character"`
}

type codeLensWire struct {
	Range   rangeWire       `json:"range"`
	Command *commandWire    `json:"command,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// RequestCodeLens sends textDocument/codeLens and returns summarized items.
func RequestCodeLens(ctx context.Context, t *Transport, uri, content string, serverPosEnc PositionEncoding) ([]CodeLensItem, error) {
	rawItems, err := requestCodeLensRaw(ctx, t, uri)
	if err != nil {
		return nil, err
	}

	items, err := parseCodeLensResult(rawItems, content, serverPosEnc)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	return items, nil
}

// ApplyCodeLens executes one code lens command by index.
func ApplyCodeLens(ctx context.Context, t *Transport, uri string, index int) (bool, error) {
	rawItems, err := requestCodeLensRaw(ctx, t, uri)
	if err != nil {
		return false, err
	}
	if index < 0 || index >= len(rawItems) {
		return false, fmt.Errorf("lsp: code lens index out of range")
	}

	lens, err := parseCodeLensWire(rawItems[index])
	if err != nil {
		return false, err
	}

	if lens.Command == nil || lens.Command.Command == "" {
		resolved, resolveErr := resolveCodeLens(ctx, t, lens)
		if resolveErr != nil {
			return false, resolveErr
		}
		lens = resolved
	}

	if lens.Command == nil || lens.Command.Command == "" {
		return false, nil
	}

	params := map[string]any{"command": lens.Command.Command}
	if len(lens.Command.Arguments) > 0 && string(lens.Command.Arguments) != "null" {
		var args any
		if err := json.Unmarshal(lens.Command.Arguments, &args); err == nil {
			params["arguments"] = args
		}
	}

	if _, err := t.Call(ctx, "workspace/executeCommand", params, 1500*time.Millisecond); err != nil {
		return false, err
	}

	return true, nil
}

func requestCodeLensRaw(ctx context.Context, t *Transport, uri string) ([]json.RawMessage, error) {
	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
	}

	resp, err := t.Call(ctx, "textDocument/codeLens", params, 1500*time.Millisecond)
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

	var rawItems []json.RawMessage
	if err := json.Unmarshal(resp.Result, &rawItems); err != nil {
		return nil, fmt.Errorf("lsp: parse codeLens payload: %w", err)
	}

	if len(rawItems) == 0 {
		return nil, nil
	}

	return rawItems, nil
}

func parseCodeLensResult(rawItems []json.RawMessage, content string, serverPosEnc PositionEncoding) ([]CodeLensItem, error) {
	items := make([]CodeLensItem, 0, len(rawItems))
	for index, raw := range rawItems {
		lens, err := parseCodeLensWire(raw)
		if err != nil {
			return nil, err
		}

		character := lens.Range.Start.Character
		if serverPosEnc != PositionEncodingUTF16 {
			character = convertCharacterAtLine(content, lens.Range.Start.Line, character, serverPosEnc, PositionEncodingUTF16)
		}

		items = append(items, CodeLensItem{
			Index:     index,
			Title:     codeLensTitle(lens.Command),
			Line:      lens.Range.Start.Line,
			Character: character,
		})
	}

	return items, nil
}

func parseCodeLensWire(raw json.RawMessage) (codeLensWire, error) {
	var lens codeLensWire
	if err := json.Unmarshal(raw, &lens); err != nil {
		return codeLensWire{}, fmt.Errorf("lsp: parse codeLens item: %w", err)
	}

	return lens, nil
}

func resolveCodeLens(ctx context.Context, t *Transport, lens codeLensWire) (codeLensWire, error) {
	resp, err := t.Call(ctx, "textDocument/resolveCodeLens", lens, 1500*time.Millisecond)
	if err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code == -32601 {
			return lens, nil
		}
		return codeLensWire{}, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return lens, nil
	}

	resolved, err := parseCodeLensWire(resp.Result)
	if err != nil {
		return codeLensWire{}, err
	}

	return resolved, nil
}

func codeLensTitle(cmd *commandWire) string {
	if cmd == nil {
		return "Code lens"
	}
	if cmd.Title != "" {
		return cmd.Title
	}
	if cmd.Command != "" {
		return cmd.Command
	}

	return "Code lens"
}
