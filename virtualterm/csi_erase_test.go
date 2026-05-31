package virtualterm

import (
	"testing"
)

/*
	ERASE DISPLAY
*/

func TestCsiEraseDisplayAfter(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\nkl",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiEraseDisplayAfter()
				},
			},
		},
	}

	test.RunTests(t)
}

func TestCsiEraseDisplayBefore(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "..........\n..........\n...nopqrst\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiEraseDisplayBefore()
				},
			},
		},
	}

	test.RunTests(t)
}

/*
	ERASE LINE
*/

func TestCsiEraseLineAfter(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\nkl........\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiEraseLineAfter()
				},
			},
		},
	}

	test.RunTests(t)
}

func TestCsiEraseLineBefore(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\n...nopqrst\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiEraseLineBefore()
				},
			},
		},
	}

	test.RunTests(t)
}

func TestCsiEraseLine(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\n..........\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiEraseLine()
				},
			},
		},
	}

	test.RunTests(t)
}

/*
	ERASE CHARACTERS
*/

func TestCsiEraseCharacters(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\nkl.nopqrst\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiEraseCharacters(1)
				},
			},
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\nkl..opqrst\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiEraseCharacters(2)
				},
			},
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\nkl...pqrst\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiEraseCharacters(3)
				},
			},
		},
	}

	test.RunTests(t)
}

/*
	DELETE
*/

func TestCsiDeleteCharacters(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\nklnopqrst.\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiDeleteCharacters(1)
				},
			},
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\nklopqrst..\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiDeleteCharacters(2)
				},
			},
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\nklpqrst...\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiDeleteCharacters(3)
				},
			},
		},
	}

	test.RunTests(t)
}

func TestCsiDeleteLines(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiDeleteLines(-1)
				},
			},
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiDeleteLines(0)
				},
			},
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiDeleteLines(1)
				},
			},
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiDeleteLines(2)
				},
			},
			{
				Screen:   "1234567890\nabcdefghij\nklmnopqrst\n0987654321",
				Expected: "1234567890\nabcdefghij",
				Operation: func(t *testing.T, term *Term) {
					term.moveCursorToPos(3, 3)
					term.csiDeleteLines(3)
				},
			},
		},
	}

	test.RunTests(t)
}
