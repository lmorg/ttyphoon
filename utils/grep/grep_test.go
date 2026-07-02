package grep

import "testing"

// TestBuildRgArgs tests buildRgArgs function with various options.
func TestBuildRgArgs(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		opts    Options
		hasFlag []string
		noFlag  []string
	}{
		{
			name:    "plain text mode",
			query:   "test",
			opts:    Options{Regex: false},
			hasFlag: []string{"-F"},
			noFlag:  []string{"-E"},
		},
		{
			name:    "regex mode",
			query:   "test[0-9]",
			opts:    Options{Regex: true},
			hasFlag: []string{},
			noFlag:  []string{"-F"},
		},
		{
			name:    "case sensitive",
			query:   "test",
			opts:    Options{CaseSensitive: true},
			hasFlag: []string{},
			noFlag:  []string{"-i"},
		},
		{
			name:    "case insensitive",
			query:   "test",
			opts:    Options{CaseSensitive: false},
			hasFlag: []string{"-i"},
			noFlag:  []string{},
		},
		{
			name:    "whole word",
			query:   "test",
			opts:    Options{WholeWord: true},
			hasFlag: []string{"-w"},
			noFlag:  []string{},
		},
		{
			name:    "all options enabled",
			query:   "test",
			opts:    Options{Regex: true, CaseSensitive: true, WholeWord: true},
			hasFlag: []string{"-w"},
			noFlag:  []string{"-F", "-i"},
		},
		{
			name:    "query starts with hyphen",
			query:   "-x",
			opts:    Options{Regex: false},
			hasFlag: []string{"-F"},
			noFlag:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildRgArgs(tt.query, tt.opts)

			// Check for required flags
			for _, flag := range tt.hasFlag {
				found := false
				for _, arg := range args {
					if arg == flag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected flag %q not found in args: %v", flag, args)
				}
			}

			// Check that excluded flags are not present
			for _, flag := range tt.noFlag {
				found := false
				for _, arg := range args {
					if arg == flag {
						found = true
						break
					}
				}
				if found {
					t.Errorf("Unexpected flag %q found in args: %v", flag, args)
				}
			}

			// Check query and dot are present
			if args[len(args)-2] != tt.query {
				t.Errorf("Expected query %q at second-to-last position, got %v", tt.query, args)
			}
			if args[len(args)-1] != "." {
				t.Errorf("Expected '.' at last position, got %v", args)
			}

			// Ensure end-of-options separator is present before query.
			if args[len(args)-3] != "--" {
				t.Errorf("Expected '--' before query, got %v", args)
			}
		})
	}
}

// TestBuildGrepArgs tests buildGrepArgs function with various options.
func TestBuildGrepArgs(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		opts    Options
		hasFlag []string
		noFlag  []string
	}{
		{
			name:    "plain text mode",
			query:   "test",
			opts:    Options{Regex: false},
			hasFlag: []string{"-F"},
			noFlag:  []string{"-E"},
		},
		{
			name:    "regex mode",
			query:   "test[0-9]",
			opts:    Options{Regex: true},
			hasFlag: []string{"-E"},
			noFlag:  []string{"-F"},
		},
		{
			name:    "case sensitive",
			query:   "test",
			opts:    Options{CaseSensitive: true},
			hasFlag: []string{},
			noFlag:  []string{"-i"},
		},
		{
			name:    "case insensitive",
			query:   "test",
			opts:    Options{CaseSensitive: false},
			hasFlag: []string{"-i"},
			noFlag:  []string{},
		},
		{
			name:    "whole word",
			query:   "test",
			opts:    Options{WholeWord: true},
			hasFlag: []string{"-w"},
			noFlag:  []string{},
		},
		{
			name:    "all options enabled",
			query:   "test",
			opts:    Options{Regex: true, CaseSensitive: true, WholeWord: true},
			hasFlag: []string{"-E", "-w"},
			noFlag:  []string{"-F", "-i"},
		},
		{
			name:    "query starts with hyphen",
			query:   "-x",
			opts:    Options{Regex: false},
			hasFlag: []string{"-F"},
			noFlag:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildGrepArgs(tt.query, tt.opts)

			// Check for required flags
			for _, flag := range tt.hasFlag {
				found := false
				for _, arg := range args {
					if arg == flag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected flag %q not found in args: %v", flag, args)
				}
			}

			// Check that excluded flags are not present
			for _, flag := range tt.noFlag {
				found := false
				for _, arg := range args {
					if arg == flag {
						found = true
						break
					}
				}
				if found {
					t.Errorf("Unexpected flag %q found in args: %v", flag, args)
				}
			}

			// Check query and dot are present
			if args[len(args)-2] != tt.query {
				t.Errorf("Expected query %q at second-to-last position, got %v", tt.query, args)
			}
			if args[len(args)-1] != "." {
				t.Errorf("Expected '.' at last position, got %v", args)
			}

			// Ensure end-of-options separator is present before query.
			if args[len(args)-3] != "--" {
				t.Errorf("Expected '--' before query, got %v", args)
			}
		})
	}
}
