package virtualterm

import (
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

func TestSearchCmdLinesBuf(t *testing.T) {
	buf := types.Screen{
		{
			Id:      101,
			Block:   &types.BlockMeta{Query: []rune("ls"), ExitNum: 7},
			RowMeta: types.META_ROW_BEGIN_BLOCK,
		},
		{
			Id:      202,
			Block:   &types.BlockMeta{Query: []rune("pwd"), ExitNum: 0},
			RowMeta: types.META_ROW_NONE,
		},
		{
			Id:      303,
			Block:   &types.BlockMeta{Query: []rune("echo hello"), ExitNum: 1},
			RowMeta: types.META_ROW_BEGIN_BLOCK,
		},
		{
			Id:      404,
			Block:   &types.BlockMeta{Query: []rune("summarise logs"), ExitNum: 2, Meta: types.META_BLOCK_AI},
			RowMeta: types.META_ROW_BEGIN_BLOCK,
		},
	}

	commands := _searchCmdLinesBuf(buf, false)
	if len(commands) != 2 {
		t.Fatalf("_searchCmdLinesBuf(commands) returned %d results, want 2", len(commands))
	}

	if commands[0].rowId != 303 || commands[0].query != "echo hello" || commands[0].exitNum != 1 {
		t.Fatalf("first command result = %#v, want rowId=303 query=%q exitNum=1", commands[0], "echo hello")
	}

	if commands[1].rowId != 101 || commands[1].query != "ls   " || commands[1].exitNum != 7 {
		t.Fatalf("second command result = %#v, want rowId=101 query=%q exitNum=7", commands[1], "ls   ")
	}

	formatted := commands.SliceWithExitNum()
	if len(formatted) != 2 || formatted[1] != "7  : ls   " {
		t.Fatalf("SliceWithExitNum() = %#v, want second entry %q", formatted, "7  : ls   ")
	}

	ai := _searchCmdLinesBuf(buf, true)
	if len(ai) != 1 {
		t.Fatalf("_searchCmdLinesBuf(ai) returned %d results, want 1", len(ai))
	}

	if ai[0].rowId != 404 || ai[0].query != "summarise logs" || ai[0].exitNum != 2 {
		t.Fatalf("AI result = %#v, want rowId=404 query=%q exitNum=2", ai[0], "summarise logs")
	}
}
