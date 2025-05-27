package virtualterm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/ptty"
	"github.com/lmorg/mxtty/types"
)

const (
	_SUBTERM_META_BEGIN = "\x1b_begin;output-block;%s\x1b\\"
	_SUBTERM_META_END   = "\x1b_end;output-block;{\"MetaFlag\":%d}\x1b\\"
)

type subTermTileT struct {
	parentTerm types.Term
	parentTile types.Tile
	curPath    string
}

func (stt *subTermTileT) Name() string   { return stt.parentTile.Name() }
func (stt *subTermTileT) Id() string     { return stt.parentTile.Id() }
func (stt *subTermTileT) Left() int32    { return stt.parentTile.Left() }
func (stt *subTermTileT) Top() int32     { return stt.parentTile.Top() }
func (stt *subTermTileT) Right() int32   { return stt.parentTile.Right() }
func (stt *subTermTileT) Bottom() int32  { return stt.parentTile.Bottom() }
func (stt *subTermTileT) AtBottom() bool { return stt.parentTile.AtBottom() }
func (stt *subTermTileT) Close()         {}

func (stt *subTermTileT) GetTerm() types.Term     { return stt.parentTerm }
func (stt *subTermTileT) SetTerm(term types.Term) { stt.parentTerm = term }
func (stt *subTermTileT) Pwd() string             { return stt.curPath }

func (term *Term) newSubTerm(query, content string, meta types.RowMetaFlag) types.Screen {
	debug.Log(content)

	tile := subTermTileT{
		parentTerm: term,
		parentTile: term.tile,
		curPath:    term.tile.Pwd(),
	}

	beginPayloadMap := map[string]string{
		"CmdLine": query,
	}
	beginPayloadBytes, _ := json.Marshal(beginPayloadMap)

	content = strings.ReplaceAll(content, "\n", "\r\n")
	pty := ptty.NewMock()

	subTerm := NewTerminal(&tile, term.renderer, &types.XY{X: term.size.X, Y: 10000}, false)
	subTerm.Start(pty)

	b := fmt.Appendf(nil, _SUBTERM_META_BEGIN, beginPayloadBytes)
	b = append(b, []byte(content)...)
	err := pty.Write(fmt.Appendf(b, _SUBTERM_META_END, meta))
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("unable to write content to sub-term: %v", err))
	}

	for {
		if pty.BufSize() == 0 {
			time.Sleep(1 * time.Second) // a bit of a kludge
			break
		}
	}

	debug.Log(subTerm.curPos())
	subTerm.Close()
	return subTerm._normBuf[0:subTerm.curPos().Y]
}

func (term *Term) InsertSubTerm(query, content string, insertAtRowId uint64, meta types.RowMetaFlag) error {
	rows := term.newSubTerm(query, content, meta)
	return term.insertRowsAtRowId(insertAtRowId, rows)
}
