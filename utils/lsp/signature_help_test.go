package lsp

import (
	"encoding/json"
	"testing"
)

func TestParseSignatureHelpResult_Basic(t *testing.T) {
	raw := json.RawMessage(`{
		"signatures": [
			{
				"label": "println(a ...any)",
				"documentation": {"kind":"markdown","value":"Prints values."},
				"parameters": [
					{"label": "a ...any"}
				]
			}
		],
		"activeSignature": 0,
		"activeParameter": 0
	}`)

	text, err := parseSignatureHelpResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "" {
		t.Fatalf("expected non-empty text")
	}
	if text != "println(a ...any)\n\nPrints values.\n\nParameter: a ...any" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestParseSignatureHelpResult_ActiveSignatureFallback(t *testing.T) {
	raw := json.RawMessage(`{
		"signatures": [
			{"label": "alpha(x int)", "parameters": [{"label": "x int"}]},
			{"label": "beta(y string)", "parameters": [{"label": "y string"}]}
		],
		"activeSignature": 9,
		"activeParameter": 0
	}`)

	text, err := parseSignatureHelpResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "" {
		t.Fatalf("expected non-empty text")
	}
	if text != "alpha(x int)\n\nParameter: x int\n\nSignature 1 of 2" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestParameterLabelText_Span(t *testing.T) {
	label := parameterLabelText("foo(first int, second string)", json.RawMessage(`[4,13]`))
	if label != "first int" {
		t.Fatalf("unexpected span label: %q", label)
	}
}
