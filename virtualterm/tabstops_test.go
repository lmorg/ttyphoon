package virtualterm

import (
	"fmt"
	"testing"
)

func TestAddTabStop(t *testing.T) {
	term := NewTestTerminal()

	tests := []struct {
		CurPos int32
	}{
		{
			CurPos: 1,
		},
		{
			CurPos: 2,
		},
		{
			CurPos: 4,
		},
		{
			CurPos: 2,
		},
		{
			CurPos: 7,
		},
		{
			CurPos: 3,
		},
	}

	// term is 10 chars wide
	expected := "[1 2 2 3 4 7]"

	for _, test := range tests {
		term._curPos.X = test.CurPos
		term.c1AddTabStop()
	}

	if fmt.Sprintf("%v", term._tabStops) != expected {
		t.Errorf("Expected does not match actual in test:")
		t.Logf("  expected: %s", expected)
		t.Logf("  actual:   %v", term._tabStops)
	}
}

func TestAddTabStopOverflow(t *testing.T) {
	term := NewTestTerminal()

	tests := []struct {
		CurPos int32
	}{
		{
			CurPos: 1,
		},
		{
			CurPos: 2,
		},
		{
			CurPos: 9,
		},
		{
			CurPos: 23,
		},
		{
			CurPos: 2,
		},
		{
			CurPos: 50,
		},
		{
			CurPos: 2,
		},
	}

	// term is 8 chars wide
	expected := "[1 2 2 2 9 9 9]"

	for _, test := range tests {
		term._curPos.X = test.CurPos
		term.c1AddTabStop()
	}

	if fmt.Sprintf("%v", term._tabStops) != expected {
		t.Errorf("Expected does not match actual in test:")
		t.Logf("  expected: %s", expected)
		t.Logf("  actual:   %v", term._tabStops)
	}
}

func TestAddClearTabStop(t *testing.T) {
	term := NewTestTerminal()

	tests := []struct {
		CurPos int32
	}{
		{
			CurPos: 1,
		},
		{
			CurPos: 2,
		},
		{
			CurPos: 9,
		},
		{
			CurPos: 7,
		},
		{
			CurPos: 2,
		},
		{
			CurPos: 5,
		},
		{
			CurPos: 2,
		},
	}

	expected := "[1 5 7 9]"

	for _, test := range tests {
		term._curPos.X = test.CurPos
		term.c1AddTabStop()
	}

	term.csiClearTabStop()

	if fmt.Sprintf("%v", term._tabStops) != expected {
		t.Errorf("Expected does not match actual in test:")
		t.Logf("  expected: %s", expected)
		t.Logf("  actual:   %v", term._tabStops)
	}
}

func TestNextTabStop(t *testing.T) {
	term := NewTestTerminal()

	tests := []struct {
		CurPos   int32
		Expected int32
	}{
		{
			CurPos:   0,
			Expected: 8,
		},
		{
			CurPos:   2,
			Expected: 6,
		},
		{
			CurPos:   8,
			Expected: 8,
		},
		{
			CurPos:   23,
			Expected: 7,
		},
	}

	for i, test := range tests {
		term._curPos.X = test.CurPos
		actual := term.nextTabStop()

		if actual != test.Expected {
			t.Errorf("Expected does not match actual in test %d:", i)
			t.Logf("  curPos.X: %d", term._curPos.X)
			t.Logf("  tabStops: %v", term._tabStops)
			t.Logf("  Expected: %d", test.Expected)
			t.Logf("  Actual:   %d", actual)
		}
	}
}
