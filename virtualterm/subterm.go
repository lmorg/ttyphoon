package virtualterm

import (
	"fmt"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/tty"
	"github.com/lmorg/ttyphoon/types"
)

type subTermTileT struct {
	//parentTerm types.Term
	parentTile types.Tile
	term       types.Term
	curPath    string
}

func (stt *subTermTileT) Name() string      { return stt.parentTile.Name() }
func (stt *subTermTileT) SetName(string)    {}
func (stt *subTermTileT) GroupName() string { return stt.parentTile.GroupName() }
func (stt *subTermTileT) Id() string        { return stt.parentTile.Id() }
func (stt *subTermTileT) Left() int32       { return stt.parentTile.Left() }
func (stt *subTermTileT) Top() int32        { return stt.parentTile.Top() }
func (stt *subTermTileT) Right() int32      { return stt.parentTile.Right() }
func (stt *subTermTileT) Bottom() int32     { return stt.parentTile.Bottom() }
func (stt *subTermTileT) AtBottom() bool    { return stt.parentTile.AtBottom() }
func (stt *subTermTileT) Close()            {}

func (stt *subTermTileT) GetTerm() types.Term     { return stt.term }
func (stt *subTermTileT) SetTerm(term types.Term) { stt.term = term }
func (stt *subTermTileT) Pwd() string             { return stt.curPath }

func (term *Term) newSubTerm(cmdLine, content string, meta types.BlockMetaFlag, aiMeta *types.AiMetaT, startTime time.Time) types.Screen {
	tile := subTermTileT{
		parentTile: term.tile,
		curPath:    term.tile.Pwd(),
	}

	content = strings.ReplaceAll(content, "\n", "\r\n")
	pty := tty.NewMock()

	subTerm := NewTerminal(&tile, term.renderer, &types.XY{X: term.size.X, Y: 1}, false)
	subTerm.Start(pty)

	subTerm.beginOutputBlock([]rune(cmdLine))

	err := pty.Write([]byte(content))
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("unable to write content to sub-term: %v", err))
	}

	for {
		if pty.BufSize() == 0 {
			time.Sleep(250 * time.Millisecond) // a bit of a kludge
			break
		}
	}

	subTerm.endOutputBlock(&endOutputBlockT{MetaFlag: meta}, aiMeta)

	subTerm.Close()
	subTerm.tile.SetTerm(term) // also a bit of a kludge

	return append(subTerm._scrollBuf, subTerm._normBuf...)
}

func (term *Term) InsertSubTerm(query, content string, insertAtRowId uint64, meta types.BlockMetaFlag, aiMeta *types.AiMetaT, startTime time.Time) error {
	rows := term.newSubTerm(query, content, meta, aiMeta, startTime)
	return term.insertRowsAtRowId(insertAtRowId, rows)
}
