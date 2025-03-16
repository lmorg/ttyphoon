package elementCsv

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
)

type ElementCsv struct {
	renderer   types.Renderer
	tile       types.Tile
	size       types.XY
	headings   [][]rune // columns
	table      [][]rune // rendered rows
	top        []rune   // rendered headings
	width      []int    // columns
	boundaries []int32  // column lines
	isNumber   []bool   // columns

	//parameters parametersT

	name   string
	buf    []rune
	lines  int32
	notify types.Notification

	db   *sql.DB
	dbTx *sql.Tx

	filter       string
	orderByIndex int  // row
	orderDesc    bool // ASC or DESC

	renderOffset int32
	limitOffset  int32
	highlight    *types.XY
}

var arrowGlyph = map[bool]rune{
	false: '↑',
	true:  '↓',
}

const notifyLoading = "Loading CSV. Line %d..."

func New(renderer types.Renderer, tile types.Tile) *ElementCsv {
	el := &ElementCsv{renderer: renderer, tile: tile}

	el.notify = renderer.DisplaySticky(types.NOTIFY_INFO, fmt.Sprintf(notifyLoading, el.lines))

	err := el.createDb()
	if err != nil {
		panic(err)
	}

	return el
}

func (el *ElementCsv) Write(r rune) error {
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

func (el *ElementCsv) Generate(apc *types.ApcSlice) error {
	defer el.notify.Close()

	buf := bytes.NewBufferString(string(el.buf))
	r := csv.NewReader(buf)
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1
	recs, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading CSV: %v", err)
	}

	var params parametersT
	apc.Parameters(&params)
	debug.Log(params)

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
			headings[i] = string('A' + i) // A, B, C, D, etc...
		}
	}

	err = el.createTable(headings)
	if err != nil {
		return err
	}

	n := len(headings)

	el.headings = make([][]rune, n)
	for i := range headings {
		el.headings[i] = []rune(headings[i])
	}

	// figure out if number
	el.isNumber = make([]bool, n)
	for col := 0; col < n && col < len(recs[firstRecord]); col++ {
		_, e := strconv.ParseFloat(recs[firstRecord][col], 64)
		el.isNumber[col] = e == nil // if no error, then it's probably a number
	}

	for row := firstRecord; row < len(recs); row++ {
		if len(recs[row]) > n {
			recs[row][n-1] = strings.Join(recs[row][n-1:], " ")
			recs[row] = recs[row][:n]
		}
		err = el.insertRecords(recs[row])
		if err != nil {
			return err
		}
	}

	if el.dbTx.Commit() != nil {
		return fmt.Errorf("cannot commit sqlite3 transaction: %v", err)
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

func (el *ElementCsv) Size() *types.XY {
	return &el.size
}

func (el *ElementCsv) Rune(pos *types.XY) rune {
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

func (el *ElementCsv) Close() {
	// clear memory (if required)
	el.db.Close()
}
