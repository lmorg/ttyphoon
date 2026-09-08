package main

import "testing"

func TestLSPReferenceContextIncludesAdjacentLines(t *testing.T) {
	got := lspReferenceContext(2, "file:///main.go", func(string) (string, bool) {
		return "before\nmatch\nafter\nlast", true
	})
	want := []string{"match", "after", "last"}
	if len(got) != len(want) {
		t.Fatalf("context = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("context = %v, want %v", got, want)
		}
	}
}
