package virtualterm

import (
	"testing"
)

func TestC1DecalnTestAlignment(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "12345",
				Expected: "EEEEEEEEEE\nEEEEEEEEEE\nEEEEEEEEEE\nEEEEEEEEEE\nEEEEEEEEEE",
				Operation: func(t *testing.T, term *Term) {
					term.c1DecalnTestAlignment()
				},
			},
		},
	}

	test.RunTests(t)
}
