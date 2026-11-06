package browser

import "testing"

func TestParseWebURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantURL string
		wantErr bool
	}{
		{name: "https", input: "https://example.com/path?q=1", wantURL: "https://example.com/path?q=1"},
		{name: "uppercase scheme", input: "HTTPS://example.com/path", wantURL: "https://example.com/path"},
		{name: "relative URL", input: "/path", wantErr: true},
		{name: "non-web scheme", input: "mailto:user@example.com", wantErr: true},
		{name: "missing host", input: "https:///path", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsedURL, err := parseWebURL(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseWebURL(%q) error = %v, wantErr %v", test.input, err, test.wantErr)
			}
			if err == nil && parsedURL.String() != test.wantURL {
				t.Errorf("parseWebURL(%q) = %q, want %q", test.input, parsedURL.String(), test.wantURL)
			}
		})
	}
}
