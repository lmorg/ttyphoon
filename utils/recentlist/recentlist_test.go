package recentlist

import "testing"

func TestPromote_MovesItemToFrontAndDeduplicates(t *testing.T) {
	got := Promote([]string{"a", "b", "c", "b"}, "b", 0)
	want := []string{"b", "a", "c", "b"}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPromote_RespectsLimit(t *testing.T) {
	got := Promote([]string{"b", "c", "d"}, "a", 3)
	want := []string{"a", "b", "c"}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d = %q, want %q", i, got[i], want[i])
		}
	}
}
