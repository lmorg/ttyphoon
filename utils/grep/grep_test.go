package grep

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func collectSearch(ctx context.Context, root, query string) ([]*Result, error) {
	results := make(chan []*Result)
	var collected []*Result
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		for batch := range results {
			collected = append(collected, batch...)
		}
	}()
	err := BatchedStreamResults(ctx, root, query, Options{}, func(path string) string { return path }, results)
	wait.Wait()
	return collected, err
}

func TestBatchedStreamResults_ConcurrentSearchesDoNotCancelEachOther(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "one.txt"), []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "two.txt"), []byte("beta\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var alpha, beta []*Result
	var alphaErr, betaErr error
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		alpha, alphaErr = collectSearch(context.Background(), root, "alpha")
	}()
	go func() {
		defer wait.Done()
		beta, betaErr = collectSearch(context.Background(), root, "beta")
	}()
	wait.Wait()

	if alphaErr != nil || betaErr != nil {
		t.Fatalf("search errors = %v, %v", alphaErr, betaErr)
	}
	if len(alpha) != 1 || len(beta) != 1 {
		t.Fatalf("search result counts = %d, %d; want 1, 1", len(alpha), len(beta))
	}
}

func TestBatchedStreamResults_UsesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := collectSearch(ctx, t.TempDir(), "anything")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("BatchedStreamResults() error = %v, want context.Canceled", err)
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
