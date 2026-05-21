package virtualterm

import (
	"testing"
)

func TestCsiRepeatPreceding(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "12345",
				Expected: "12345555",
				Operation: func(t *testing.T, term *Term) {
					term.csiRepeatPreceding(3)
				},
			},
			{
				Screen:   "12345",
				Expected: "123333",
				Operation: func(t *testing.T, term *Term) {
					term.csiMoveCursorBackwards(2)
					term.csiRepeatPreceding(3)
				},
			},
			{
				Screen:   "12345",
				Expected: "12225",
				Operation: func(t *testing.T, term *Term) {
					term.csiMoveCursorBackwards(3)
					term.csiRepeatPreceding(2)
				},
			},
		},
	}

	test.RunTests(t)
}
