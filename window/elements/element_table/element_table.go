package element_table

import (
	"database/sql"
	"fmt"
	"runtime"
	"time"

	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/elements/element_table/tablecore"
)

type elementType int

const (
	_ELEMENT_TYPE_CSV      = 1 + iota
	_ELEMENT_TYPE_MARKDOWN = 2
)

type ElementTable struct {
	elType     elementType
	renderer   types.Renderer
	tile       types.Tile
	size       types.XY
	headings   [][]rune // columns
	table      [][]rune // rendered rows
	top        []rune   // rendered headings
	width      []int    // columns
	boundaries []int32  // column lines

	//parameters parametersT

	name   string
	buf    []rune
	lines  int32
	notify types.Notification

	// Sort and filter semantics live in core so the terminal and Notes cannot
	// drift apart; db is retained only because this element owns its lifetime.
	db   *sql.DB
	core *tablecore.Table

	renderOffset int32 // negative value
	limitOffset  int32
}

var arrowGlyph = map[bool]rune{
	false: '↑',
	true:  '↓',
}

const notifyLoading = "Loading table. Line %d..."

func NewCsv(renderer types.Renderer, tile types.Tile) *ElementTable {
	return newTable(renderer, tile, _ELEMENT_TYPE_CSV)
}

func NewMarkdown(renderer types.Renderer, tile types.Tile) *ElementTable {
	return newTable(renderer, tile, _ELEMENT_TYPE_MARKDOWN)
}

func newTable(renderer types.Renderer, tile types.Tile, elType elementType) *ElementTable {
	el := &ElementTable{
		renderer: renderer,
		tile:     tile,
		elType:   elType,
	}

	el.notify = renderer.DisplaySticky(types.NOTIFY_INFO, fmt.Sprintf(notifyLoading, el.lines), func() {})

	db, err := tablecore.NewMemoryDB()
	if err != nil {
		panic(err)
	}
	el.db = db

	// close DB upon deallocation and garbage collection
	runtime.AddCleanup(el, func(db *sql.DB) { db.Close() }, el.db)

	return el
}

func (el *ElementTable) Write(r rune) error {
	el.buf = append(el.buf, r)
	if r == '\n' {
		el.lines++
		el.notify.SetMessage(fmt.Sprintf(notifyLoading, el.lines))
	}
	return nil
}

type parametersT struct {
	CreateHeadings bool `json:"CreateHeadings"`
}

func (el *ElementTable) Generate(apc *types.ApcSlice) error {
	defer el.notify.Close()

	var (
		recs [][]string
		err  error
	)

	params := new(parametersT)
	apc.Parameters(params)
	debug.Log(params)

	switch el.elType {
	case _ELEMENT_TYPE_CSV:
		recs, err = fromCsv(el)
	case _ELEMENT_TYPE_MARKDOWN:
		recs, err = fromMarkdown(el, params)
	default:
		panic("unknown table type")
	}
	if err != nil {
		return err
	}

	firstRecord := 1
	if params.CreateHeadings {
		firstRecord = 0
		el.lines++
	}

	if len(recs) <= firstRecord {
		return fmt.Errorf("too few rows") // TODO: this shouldn't error
	}

	headings := recs[0]
	if params.CreateHeadings {
		headings = make([]string, len(recs[0]))
		for i := range headings {
			headings[i] = string([]rune{'A' + int32(i)}) // A, B, C, D, etc...
		}
	}

	n := len(headings)

	el.headings = make([][]rune, n)
	for i := range headings {
		el.headings[i] = []rune(headings[i])
	}

	el.name = fmt.Sprintf("term_%d", time.Now().UnixMicro())
	el.core, err = tablecore.New(el.db, el.name, headings, recs[firstRecord:])
	if err != nil {
		return err
	}

	el.size = *el.tile.GetTerm().GetSize()
	if el.size.Y > 8 {
		el.size.Y -= 5
	}
	if el.size.Y > el.lines {
		el.size.Y = el.lines
	}

	err = el.runQuery()
	if err != nil {
		return err
	}

	return nil
}

func (el *ElementTable) Size() *types.XY {
	return &el.size
}

func (el *ElementTable) Rune(pos *types.XY) rune {
	pos.X -= el.renderOffset

	if pos.Y == 0 {
		if int(pos.X) >= len(el.top) {
			return ' '
		}
		return el.top[pos.X]
	}

	if int(pos.Y) > len(el.table) {
		return ' '
	}

	if int(pos.X) >= len(el.table[pos.Y-1]) {
		return ' '
	}

	return el.table[pos.Y-1][pos.X]
}
