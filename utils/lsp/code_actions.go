package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// CodeActionItem is a frontend-ready summary of an LSP code action.
type CodeActionItem struct {
	Title string `json:"title"`
	Kind  string `json:"kind,omitempty"`
}

// ApplyCodeActionResult describes post-apply document content.
type ApplyCodeActionResult struct {
	Content string `json:"content"`
	Changed bool   `json:"changed"`
}

type commandWire struct {
	Title     string          `json:"title"`
	Command   string          `json:"command"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type workspaceEditWire struct {
	Changes         map[string][]TextEdit `json:"changes,omitempty"`
	DocumentChanges []json.RawMessage     `json:"documentChanges,omitempty"`
}

type textDocumentEditWire struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Edits []TextEdit `json:"edits"`
}

func (w *workspaceEditWire) editsForURI(uri string) []TextEdit {
	if w == nil {
		return nil
	}

	// documentChanges preserves operation order and should be preferred when present.
	if len(w.DocumentChanges) > 0 {
		out := make([]TextEdit, 0)
		for _, raw := range w.DocumentChanges {
			var docChange textDocumentEditWire
			if err := json.Unmarshal(raw, &docChange); err != nil {
				continue
			}
			if docChange.TextDocument.URI != uri || len(docChange.Edits) == 0 {
				continue
			}
			out = append(out, docChange.Edits...)
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}

	return w.Changes[uri]
}

type codeActionWire struct {
	Title   string             `json:"title"`
	Kind    string             `json:"kind,omitempty"`
	Edit    *workspaceEditWire `json:"edit,omitempty"`
	Command *commandWire       `json:"command,omitempty"`
}

type resolvedCodeAction struct {
	Title   string
	Kind    string
	Edit    *workspaceEditWire
	Command *commandWire
}

// RequestCodeActions sends textDocument/codeAction and returns summarized actions.
func RequestCodeActions(ctx context.Context, t *Transport, uri, content string, line, character int, diagnostics []Diagnostic, serverPosEnc PositionEncoding) ([]CodeActionItem, error) {
	resolved, err := requestCodeActionsRaw(ctx, t, uri, content, line, character, diagnostics, serverPosEnc)
	if err != nil {
		return nil, err
	}

	items := make([]CodeActionItem, 0, len(resolved))
	for _, action := range resolved {
		if action.Title == "" {
			continue
		}
		items = append(items, CodeActionItem{Title: action.Title, Kind: action.Kind})
	}
	if len(items) == 0 {
		return nil, nil
	}

	return items, nil
}

// ApplyCodeAction requests code actions and applies one by index.
func ApplyCodeAction(ctx context.Context, t *Transport, uri, content string, line, character int, diagnostics []Diagnostic, index int, serverPosEnc PositionEncoding) (ApplyCodeActionResult, error) {
	resolved, err := requestCodeActionsRaw(ctx, t, uri, content, line, character, diagnostics, serverPosEnc)
	if err != nil {
		return ApplyCodeActionResult{}, err
	}
	if index < 0 || index >= len(resolved) {
		return ApplyCodeActionResult{}, fmt.Errorf("lsp: code action index out of range")
	}

	action := resolved[index]
	updated := content
	changed := false

	if action.Edit != nil {
		edits := action.Edit.editsForURI(uri)
		if len(edits) > 0 {
			edits = convertTextEdits(content, edits, serverPosEnc, PositionEncodingUTF16)
			next := ApplyTextEdits(content, edits)
			if next != content {
				updated = next
				changed = true
			}
		}
	}

	if action.Command != nil && action.Command.Command != "" {
		params := map[string]any{"command": action.Command.Command}
		if len(action.Command.Arguments) > 0 && string(action.Command.Arguments) != "null" {
			var args any
			if err := json.Unmarshal(action.Command.Arguments, &args); err == nil {
				params["arguments"] = args
			}
		}
		if _, err := t.Call(ctx, "workspace/executeCommand", params, 1500*time.Millisecond); err != nil {
			return ApplyCodeActionResult{}, err
		}
	}

	return ApplyCodeActionResult{Content: updated, Changed: changed}, nil
}

func requestCodeActionsRaw(ctx context.Context, t *Transport, uri, content string, line, character int, diagnostics []Diagnostic, serverPosEnc PositionEncoding) ([]resolvedCodeAction, error) {
	serverChar := convertCharacterAtLine(content, line, character, PositionEncodingUTF16, serverPosEnc)
	serverDiagnostics := diagnostics
	if serverPosEnc != PositionEncodingUTF16 {
		serverDiagnostics = make([]Diagnostic, len(diagnostics))
		for i := range diagnostics {
			serverDiagnostics[i] = diagnostics[i]
			serverDiagnostics[i].Range = convertRangeAtURI(content, diagnostics[i].Range, PositionEncodingUTF16, serverPosEnc)
		}
	}

	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"range": map[string]any{
			"start": map[string]int{"line": line, "character": serverChar},
			"end":   map[string]int{"line": line, "character": serverChar},
		},
		"context": map[string]any{"diagnostics": serverDiagnostics},
	}

	resp, err := t.Call(ctx, "textDocument/codeAction", params, 1500*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil, nil
	}

	var rawItems []json.RawMessage
	if err := json.Unmarshal(resp.Result, &rawItems); err != nil {
		return nil, fmt.Errorf("lsp: parse codeAction payload: %w", err)
	}

	resolved := make([]resolvedCodeAction, 0, len(rawItems))
	for _, raw := range rawItems {
		var action codeActionWire
		if err := json.Unmarshal(raw, &action); err == nil && action.Title != "" {
			resolved = append(resolved, resolvedCodeAction{
				Title:   action.Title,
				Kind:    action.Kind,
				Edit:    action.Edit,
				Command: action.Command,
			})
			continue
		}

		var cmd commandWire
		if err := json.Unmarshal(raw, &cmd); err == nil && cmd.Command != "" {
			title := cmd.Title
			if title == "" {
				title = cmd.Command
			}
			resolved = append(resolved, resolvedCodeAction{
				Title:   title,
				Command: &cmd,
			})
		}
	}

	if len(resolved) == 0 {
		return nil, nil
	}

	return resolved, nil
}
