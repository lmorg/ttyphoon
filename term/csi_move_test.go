package virtualterm

import (
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

func TestReverseLineFeed(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "..........\n..........\nfoo",
				Expected: "..........\n..........\n...bar....\nfoo",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.reverseLineFeed()
					term.writeCells("bar")
				},
			},
			{
				Screen:   "..........\n..........\nfoo",
				Expected: "..........\n..........\n......baz.\n...bar",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.reverseLineFeed()
					term.writeCells("bar")
					term.reverseLineFeed()
					term.writeCells("baz")
				},
			},
		},
	}

	test.RunTests(t)
}

func TestScrollingRegion(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "..........\n..........\nfoo.......\nbar.......\nbaz",
				Expected: "..........\n..........\nbar.......\n..........\nbaz",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.csiScrollUp(0)
				},
			},
			{
				Screen:   "..........\n..........\nfoo.......\nbar.......\nbaz",
				Expected: "..........\n..........\nbar.......\n..........\nbaz",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.csiScrollUp(1)
				},
			},
			{
				Screen:   "..........\n..........\nfoo.......\nbar.......\nbaz",
				Expected: "..........\n..........\n..........\nfoo.......\nbaz",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.csiScrollDown(0)
				},
			},
			{
				Screen:   "..........\n..........\nfoo.......\nbar.......\nbaz",
				Expected: "..........\n..........\n..........\nfoo.......\nbaz",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.csiScrollDown(1)
				},
			},
			/////
			{
				Screen:   "..........\n..........\n1234567890",
				Expected: "..........\n..........\nabcdefghij",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 3})
					term.writeCells("abcdefghij")
				},
			},
			{
				Screen:   "..........\n..........\n1234567890\n0987654321",
				Expected: "..........\n..........\nklmnopqrst\n0987654321",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 3})
					term.writeCells("abcdefghijklmnopqrst")
				},
			},
			{
				Screen:   "..........\n..........\n1234567890\n0987654321",
				Expected: "..........\n..........\nabcdefghij\nklmnopqrst",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.writeCells("abcdefghijklmnopqrst")
				},
			},
			{
				Screen:   "..........\n..........\nfoo.......\nbar",
				Expected: "..........\n..........\nbar.......\n",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.carriageReturn()
					term.lineFeed(_LINEFEED_CURSOR_MOVED)
				},
			},
			{
				Screen:   "..........\n..........\nfoo.......\nbar",
				Expected: "..........\n..........\nbar.......\nbaz8",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.carriageReturn()
					term.lineFeed(_LINEFEED_CURSOR_MOVED)
					term.writeCells("baz8")
				},
			},
			{
				Screen:   "..........\n..........\nfoo.......\nbar",
				Expected: "..........\n..........\nfoobaz....\nbar",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.csiMoveCursorUpwards(20)
					term.writeCells("baz")
				},
			},
			///// scroll downwards
			{
				Screen:   "..........\n..........\nfoo.......\nbar",
				Expected: "..........\n..........\nbaz.......\nfoo",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.carriageReturn()
					term.csiMoveCursorUpwards(1)
					term.csiScrollDown(1)
					term.csiMoveCursorUpwards(1)
					term.writeCells("baz")
				},
			},
			{
				Screen:   "..........\n..........\nfoo.......\nbar",
				Expected: "..........\n..........\nbaz.......\nfoo",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.carriageReturn()
					term.csiMoveCursorUpwards(10)
					term.csiScrollDown(1)
					term.csiMoveCursorUpwards(10)
					term.writeCells("baz")
				},
			},
			{
				Screen:   "..........\n..........\nfoo.......\nbar",
				Expected: "..........\n..........\n..........\nfoobaz",
				Operation: func(t *testing.T, term *Term) {
					term.setScrollingRegion([]int32{3, 4})
					term.csiMoveCursorDownwards(10)
					term.csiScrollDown(1)
					term.csiMoveCursorDownwards(10)
					term.writeCells("baz")
				},
			},
		},
	}

	test.RunTests(t)
}

func TestCsiScrollUp(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "2222222222\n3333333333\n4444444444",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollUp(-1)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "2222222222\n3333333333\n4444444444",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollUp(0)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "2222222222\n3333333333\n4444444444",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollUp(1)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "3333333333\n4444444444",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollUp(2)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "4444444444",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollUp(3)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollUp(4)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollUp(5)
				},
			},
		},
	}

	test.RunTests(t)
}

func TestCsiScrollDown(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n1111111111\n2222222222\n3333333333\n4444444444",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollDown(-1)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n1111111111\n2222222222\n3333333333\n4444444444",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollDown(0)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n1111111111\n2222222222\n3333333333\n4444444444",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollDown(1)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n..........\n1111111111\n2222222222\n3333333333",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollDown(2)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n..........\n..........\n1111111111\n2222222222",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollDown(3)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n..........\n..........\n..........\n1111111111",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollDown(4)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n..........\n..........\n..........",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollDown(5)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n..........\n..........\n..........",
				Operation: func(t *testing.T, term *Term) {
					term.csiScrollDown(6)
				},
			},
		},
	}

	test.RunTests(t)
}

func Test_scrollDown(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n1111111111\n2222222222\n3333333333\n4444444444",
				Operation: func(t *testing.T, term *Term) {
					top, bottom := term.getScrollingRegionExcOrigin()
					term._scrollDown(top, bottom, 1)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n..........\n1111111111\n2222222222\n3333333333",
				Operation: func(t *testing.T, term *Term) {
					top, bottom := term.getScrollingRegionExcOrigin()
					term._scrollDown(top, bottom, 2)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n..........\n..........\n1111111111\n2222222222",
				Operation: func(t *testing.T, term *Term) {
					top, bottom := term.getScrollingRegionExcOrigin()
					term._scrollDown(top, bottom, 3)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n..........\n..........\n..........\n1111111111",
				Operation: func(t *testing.T, term *Term) {
					top, bottom := term.getScrollingRegionExcOrigin()
					term._scrollDown(top, bottom, 4)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n..........\n..........\n..........",
				Operation: func(t *testing.T, term *Term) {
					top, bottom := term.getScrollingRegionExcOrigin()
					term._scrollDown(top, bottom, 5)
				},
			},
			{
				Screen:   "1111111111\n2222222222\n3333333333\n4444444444",
				Expected: "..........\n..........\n..........\n..........",
				Operation: func(t *testing.T, term *Term) {
					top, bottom := term.getScrollingRegionExcOrigin()
					term._scrollDown(top, bottom, 6)
				},
			},
		},
	}

	test.RunTests(t)
}

func TestCsiInsertLines(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1234567890\nabcde",
				Expected: "1234567890\n..........\nabcde",
			},
			{
				Screen:   "1234567890\n          \nabcde",
				Expected: "1234567890\n          \n..........\nabcde",
			},
		},
		Operation: func(t *testing.T, term *Term) {
			term.csiInsertLines(1)
		},
	}

	test.RunTests(t)
}

func TestCsiInsertCharacters(t *testing.T) {
	test := testTerm{
		Tests: []testCondition{
			{
				Screen:   "1234567890",
				Expected: "123..45678",
			},
		},
		Operation: func(t *testing.T, term *Term) {
			term._curPos = types.XY{X: 3, Y: 0}
			term.csiInsertCharacters(2)
		},
	}

	test.RunTests(t)
}
// TestMoveCursorToRow_DefaultParam verifies that row=0 (the default when no
// CSI parameter is given) is treated as row 1, placing the cursor at the top
// of the screen (row index 0).
func TestMoveCursorToRow_DefaultParam(t *testing.T) {
	term := NewTestTerminal()
	term._curPos = types.XY{X: 3, Y: 3}

	term.moveCursorToRow(0)

	if term._curPos.Y != 0 {
		t.Errorf("moveCursorToRow(0): expected Y=0, got Y=%d", term._curPos.Y)
	}
}

// TestMoveCursorToPos_DefaultParams verifies that (0,0) places the cursor at
// the top-left corner (row/col 1 in ANSI 1-based addressing).
func TestMoveCursorToPos_DefaultParams(t *testing.T) {
	term := NewTestTerminal()
	term._curPos = types.XY{X: 5, Y: 4}

	term.moveCursorToPos(0, 0)

	if term._curPos.X != 0 || term._curPos.Y != 0 {
		t.Errorf("moveCursorToPos(0,0): expected (0,0), got (%d,%d)",
			term._curPos.X, term._curPos.Y)
	}
}
