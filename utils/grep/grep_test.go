package grep

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSearchWithOptionsCaseSensitive tests case-sensitive search option.
func TestSearchWithOptionsCaseSensitive(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "Hello\nhello\nHELLO"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Case-insensitive search (default)
	matches, err := SearchWithOptions(tmpDir, "hello", Options{CaseSensitive: false})
	if err != nil {
		t.Fatalf("SearchWithOptions failed: %v", err)
	}
	if len(matches) != 3 {
		t.Errorf("Expected 3 case-insensitive matches, got %d", len(matches))
	}

	// Case-sensitive search
	matches, err = SearchWithOptions(tmpDir, "hello", Options{CaseSensitive: true})
	if err != nil {
		t.Fatalf("SearchWithOptions failed: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("Expected 1 case-sensitive match, got %d", len(matches))
	}
	if matches[0].Line != 2 {
		t.Errorf("Expected match on line 2, got %d", matches[0].Line)
	}
}

// TestSearchWithOptionsRegex tests regex mode option.
func TestSearchWithOptionsRegex(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "test123\ntest456\nabcdef"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Plain text search (literal)
	matches, err := SearchWithOptions(tmpDir, "test[0-9]+", Options{Regex: false})
	if err != nil {
		t.Fatalf("SearchWithOptions failed: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("Expected 0 literal matches for regex pattern, got %d", len(matches))
	}

	// Regex search
	matches, err = SearchWithOptions(tmpDir, "test[0-9]+", Options{Regex: true})
	if err != nil {
		t.Fatalf("SearchWithOptions failed: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("Expected 2 regex matches, got %d", len(matches))
	}
}

// TestSearchWithOptionsWholeWord tests whole-word match option.
func TestSearchWithOptionsWholeWord(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "test\ntesting\ntest me\nretest"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Partial word match
	matches, err := SearchWithOptions(tmpDir, "test", Options{WholeWord: false})
	if err != nil {
		t.Fatalf("SearchWithOptions failed: %v", err)
	}
	if len(matches) != 4 {
		t.Errorf("Expected 4 partial matches, got %d", len(matches))
	}

	// Whole-word match only
	matches, err = SearchWithOptions(tmpDir, "test", Options{WholeWord: true})
	if err != nil {
		t.Fatalf("SearchWithOptions failed: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("Expected 2 whole-word matches, got %d", len(matches))
	}
}

// TestSearchWithOptionsCombined tests combining multiple options.
func TestSearchWithOptionsCombined(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "WORD test\nword test\nWORD testing\nword here"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Case-insensitive + whole-word
	matches, err := SearchWithOptions(tmpDir, "word", Options{
		CaseSensitive: false,
		WholeWord:     true,
	})
	if err != nil {
		t.Fatalf("SearchWithOptions failed: %v", err)
	}
	if len(matches) != 4 {
		t.Errorf("Expected 4 case-insensitive whole-word matches, got %d", len(matches))
	}

	// Case-sensitive + whole-word
	matches, err = SearchWithOptions(tmpDir, "word", Options{
		CaseSensitive: true,
		WholeWord:     true,
	})
	if err != nil {
		t.Fatalf("SearchWithOptions failed: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("Expected 2 case-sensitive whole-word matches, got %d", len(matches))
	}
}

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
		})
	}
}

// TestSearchWithOptionsBackwardCompatibility tests that Search() still works.
func TestSearchWithOptionsBackwardCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "test\nTEST\nanother"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Original Search function should do case-insensitive plain-text search
	matches, err := Search(tmpDir, "test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("Expected 2 case-insensitive matches, got %d", len(matches))
	}
}

// TestSearchAndReturn tests the API-oriented search function.
func TestSearchAndReturn(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "hello\nHELLO\nworld"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with empty query
	result := SearchAndReturn(tmpDir, "", Options{}, nil)
	if len(result.Results) != 0 {
		t.Errorf("Expected 0 results for empty query, got %d", len(result.Results))
	}
	if result.Error != "" {
		t.Errorf("Expected no error for empty query, got: %v", result.Error)
	}

	// Test with valid query
	result = SearchAndReturn(tmpDir, "hello", Options{CaseSensitive: false}, nil)
	if len(result.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result.Results))
	}
	if result.Error != "" {
		t.Errorf("Unexpected error: %v", result.Error)
	}

	// Test with path mapper
	mapperCalled := 0
	mapper := func(path string) string {
		mapperCalled++
		return "/mapped" + path
	}
	result = SearchAndReturn(tmpDir, "world", Options{}, mapper)
	if len(result.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result.Results))
	}
	if !starts(result.Results[0].Path, "/mapped") {
		t.Errorf("Expected path to be mapped, got %q", result.Results[0].Path)
	}
	if mapperCalled == 0 {
		t.Errorf("Expected path mapper to be called")
	}
}

func starts(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
