package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type signatureHelpWire struct {
	Signatures      []signatureInformationWire `json:"signatures"`
	ActiveSignature *int                       `json:"activeSignature,omitempty"`
	ActiveParameter *int                       `json:"activeParameter,omitempty"`
}

type signatureInformationWire struct {
	Label         string           `json:"label"`
	Documentation json.RawMessage  `json:"documentation,omitempty"`
	Parameters    []parameterLabel `json:"parameters,omitempty"`
}

type parameterLabel struct {
	Label json.RawMessage `json:"label"`
}

// RequestSignatureHelp sends textDocument/signatureHelp and returns flattened text.
func RequestSignatureHelp(ctx context.Context, t *Transport, uri, content string, line, character, triggerKind int, triggerChar string, serverPosEnc PositionEncoding) (string, error) {
	serverChar := convertCharacterAtLine(content, line, character, PositionEncodingUTF16, serverPosEnc)

	sigContext := map[string]any{"triggerKind": triggerKind}
	if triggerKind == 2 && triggerChar != "" {
		sigContext["triggerCharacter"] = triggerChar
	}

	params := map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]int{"line": line, "character": serverChar},
		"context":      sigContext,
	}

	resp, err := t.Call(ctx, "textDocument/signatureHelp", params, 1200*time.Millisecond)
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Result) == 0 || string(resp.Result) == "null" {
		return "", nil
	}

	text, err := parseSignatureHelpResult(resp.Result)
	if err != nil {
		return "", err
	}

	return text, nil
}

func parseSignatureHelpResult(raw json.RawMessage) (string, error) {
	var payload signatureHelpWire
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("lsp: parse signatureHelp payload: %w", err)
	}

	if len(payload.Signatures) == 0 {
		return "", nil
	}

	activeSig := 0
	if payload.ActiveSignature != nil {
		activeSig = *payload.ActiveSignature
	}
	if activeSig < 0 || activeSig >= len(payload.Signatures) {
		activeSig = 0
	}

	sig := payload.Signatures[activeSig]
	if sig.Label == "" {
		return "", nil
	}

	text := sig.Label
	doc := HoverTextFromContents(sig.Documentation)
	if doc != "" {
		text += "\n\n" + doc
	}

	if payload.ActiveParameter != nil && *payload.ActiveParameter >= 0 && *payload.ActiveParameter < len(sig.Parameters) {
		if label := parameterLabelText(sig.Label, sig.Parameters[*payload.ActiveParameter].Label); label != "" {
			text += "\n\nParameter: " + label
		}
	}

	if len(payload.Signatures) > 1 {
		text += fmt.Sprintf("\n\nSignature %d of %d", activeSig+1, len(payload.Signatures))
	}

	return text, nil
}

func parameterLabelText(signatureLabel string, rawLabel json.RawMessage) string {
	if len(rawLabel) == 0 || string(rawLabel) == "null" {
		return ""
	}

	var direct string
	if err := json.Unmarshal(rawLabel, &direct); err == nil {
		return direct
	}

	var span []int
	if err := json.Unmarshal(rawLabel, &span); err == nil && len(span) == 2 {
		start, end := span[0], span[1]
		if start >= 0 && end > start && end <= len(signatureLabel) {
			return signatureLabel[start:end]
		}
	}

	return ""
}
