package notes

import (
	"fmt"
	"testing"
)

func TestAddFindFieldValue_PromotesAndCapsAtThirty(t *testing.T) {
	field := "notes-find-input-" + t.Name()

	for i := 1; i <= 32; i++ {
		AddFindFieldValue(field, fmt.Sprintf("term-%02d", i))
	}

	values := GetFindFieldValues(field)
	if len(values) != 30 {
		t.Fatalf("len = %d, want 30", len(values))
	}

	if values[0] != "term-32" {
		t.Fatalf("first value = %q, want %q", values[0], "term-32")
	}

	if values[29] != "term-03" {
		t.Fatalf("last value = %q, want %q", values[29], "term-03")
	}
}

func TestAddFindFieldValue_MovesExistingToTop(t *testing.T) {
	field := "notes-replace-input-" + t.Name()

	AddFindFieldValue(field, "alpha")
	AddFindFieldValue(field, "beta")
	AddFindFieldValue(field, "alpha")

	values := GetFindFieldValues(field)
	if len(values) < 2 {
		t.Fatalf("len = %d, want at least 2", len(values))
	}

	if values[0] != "alpha" {
		t.Fatalf("first value = %q, want %q", values[0], "alpha")
	}

	if values[1] != "beta" {
		t.Fatalf("second value = %q, want %q", values[1], "beta")
	}
}
