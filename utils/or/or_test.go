package or

import "testing"

func TestNotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "single non-empty string",
			input:    []string{"hello"},
			expected: "hello",
		},
		{
			name:     "single empty string",
			input:    []string{""},
			expected: "",
		},
		{
			name:     "no arguments",
			input:    []string{},
			expected: "",
		},
		{
			name:     "first non-empty",
			input:    []string{"first", "second", "third"},
			expected: "first",
		},
		{
			name:     "empty then non-empty",
			input:    []string{"", "second", "third"},
			expected: "second",
		},
		{
			name:     "multiple empty then non-empty",
			input:    []string{"", "", "", "fourth"},
			expected: "fourth",
		},
		{
			name:     "all empty strings",
			input:    []string{"", "", ""},
			expected: "",
		},
		{
			name:     "mixed with whitespace",
			input:    []string{"", "   ", "value"},
			expected: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NotEmpty(tt.input...)
			if result != tt.expected {
				t.Errorf("NotEmpty(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
