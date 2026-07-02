package jq

import "testing"

func TestParseJqXMLInput(t *testing.T) {
	input := `<xml>
		<query>.example.jq.query</query>
		<json>{ "example": "this is example json" }</json>
	</xml>`

	parsed, err := parseJqXMLInput(input)
	if err != nil {
		t.Fatalf("ParseJqXMLInput() error = %v", err)
	}

	if got, want := parsed.Query, ".example.jq.query"; got != want {
		t.Fatalf("query mismatch: got %q want %q", got, want)
	}

	if got, want := parsed.JSON, `{ "example": "this is example json" }`; got != want {
		t.Fatalf("json mismatch: got %q want %q", got, want)
	}
}
