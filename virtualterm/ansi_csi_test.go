package virtualterm

import (
	"strconv"
	"testing"
)

func TestMultiplyN(t *testing.T) {
	tests := []string{
		"0", "1", "12", "123", "4321", "9", "99", "999",
	}

	//count.Tests(t, len(tests))

	for i, test := range tests {
		var n int32
		for _, r := range test {
			multiplyN(&n, r)
		}

		exp, _ := strconv.Atoi(test)
		if n != int32(exp) {
			t.Errorf("Incorrect answer in test %d:", i)
			t.Logf("  Expected: %d", exp)
			t.Logf("  Actual:   %d", n)
		}
	}
}
