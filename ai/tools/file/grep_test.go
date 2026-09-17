package filetools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGrepCallReturnsMatchingFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "alpha.txt"), []byte("before\nneedle here\nafter\n"), 0644); err != nil {
		t.Fatalf("WriteFile(alpha): %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "beta.txt"), []byte("nothing\n"), 0644); err != nil {
		t.Fatalf("WriteFile(beta): %v", err)
	}

	tool := &Grep{agent: &pathTestAgent{projectRoot: root}}
	response, err := tool.Call(context.Background(), `{"query":"needle"}`)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	var got grepReturnT
	if err := json.Unmarshal([]byte(response), &got); err != nil {
		t.Fatalf("Unmarshal(%q): %v", response, err)
	}
	if got.Error != "" {
		t.Fatalf("grep returned error: %s", got.Error)
	}
	if got.PageCount != 1 {
		t.Fatalf("PageCount = %d, want 1", got.PageCount)
	}
	if got.PageNumber != 1 {
		t.Fatalf("PageNumber = %d, want 1", got.PageNumber)
	}
	if len(got.Results) != 1 {
		t.Fatalf("Results = %d, want 1: %q", len(got.Results), response)
	}
	result := got.Results[0]
	if result.FileName != "alpha.txt" {
		t.Fatalf("FileName = %q, want alpha.txt", result.FileName)
	}
	if result.Path != filepath.Join(root, "alpha.txt") {
		t.Fatalf("Path = %q, want alpha path", result.Path)
	}
	if result.Line != 2 {
		t.Fatalf("Line = %d, want 2", result.Line)
	}
	if len(result.Context) != 3 || result.Context[1] != "needle here" {
		t.Fatalf("Context = %#v, want matched line with surrounding context", result.Context)
	}
}

func TestGrepCallReturnsRequestedPage(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 51; i++ {
		name := filepath.Join(root, "matches", "file.txt")
		if i == 0 {
			if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
				t.Fatalf("MkdirAll: %v", err)
			}
		}
		f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			t.Fatalf("OpenFile: %v", err)
		}
		if _, err := f.WriteString("needle\n"); err != nil {
			_ = f.Close()
			t.Fatalf("WriteString: %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}

	tool := &Grep{agent: &pathTestAgent{projectRoot: root}}
	response, err := tool.Call(context.Background(), `{"query":"needle"}`)
	if err != nil {
		t.Fatalf("Call(search) error = %v", err)
	}
	var first grepReturnT
	if err := json.Unmarshal([]byte(response), &first); err != nil {
		t.Fatalf("Unmarshal(search): %v", err)
	}
	if first.PageCount != 2 || first.PageNumber != 1 || len(first.Results) != 50 {
		t.Fatalf("first page = pageCount %d pageNumber %d results %d, want 2, 1, 50", first.PageCount, first.PageNumber, len(first.Results))
	}

	response, err = tool.Call(context.Background(), `{"page":2}`)
	if err != nil {
		t.Fatalf("Call(page): %v", err)
	}
	var second grepReturnT
	if err := json.Unmarshal([]byte(response), &second); err != nil {
		t.Fatalf("Unmarshal(page): %v", err)
	}
	if second.Error != "" {
		t.Fatalf("page returned error: %s", second.Error)
	}
	if second.PageCount != 2 || second.PageNumber != 2 || len(second.Results) != 1 {
		t.Fatalf("second page = pageCount %d pageNumber %d results %d, want 2, 2, 1", second.PageCount, second.PageNumber, len(second.Results))
	}
}
