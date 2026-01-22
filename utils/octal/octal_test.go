package octal_test

import (
	"testing"

	"github.com/lmorg/ttyphoon/utils/octal"
)

func TestUnescape(t *testing.T) {
	src := `\033[1ml\033[0m\033[1D\015\015\012\033[J\033[1A\033[7C`
	expected := []byte{033, '[', '1', 'm', 'l', 033, '[', '0', 'm', 033, '[', '1', 'D', 015, 015, 012, 033, '[', 'J', 033, '[', '1', 'A', 033, '[', '7', 'C'}
	actual := octal.Unescape([]byte(src))

	if string(expected) != string(actual) {
		t.Errorf("Unescape is incorrectly decoding string:")
		t.Logf("  Source:   %s", src)
		t.Logf("  Expected: %s", string(expected))
		t.Logf("  Actual:   %s", string(actual))
		t.Logf("  exp byte: %v", expected)
		t.Logf("  act byte: %v", actual)
	}
}
