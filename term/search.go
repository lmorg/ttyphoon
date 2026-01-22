package virtualterm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/runewidth"
)

const _SEARCH_OFFSET = 0

func (term *Term) Search(mode types.SearchMode) {
	if term.IsAltBuf() {
		term.renderer.DisplayNotification(types.NOTIFY_WARN, "Search is not supported in alt buffer")
		return
	}

	switch mode {
	case types.SEARCH_REGEX:
		term.search()

	case types.SEARCH_RESULTS:
		term.SearchShowResults()

	case types.SEARCH_CLEAR:
		term.SearchClearResults()

	case types.SEARCH_CMD_LINES:
		term.SearchCmdLines()

	case types.SEARCH_AI_PROMPTS:
		term.SearchAiPrompts()

	default:
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, "invalid search mode")
	}
}

func (term *Term) search() {
	term.renderer.DisplayInputBox("Value to search for", term._searchLastString, term.searchBuf, nil)
}

func (term *Term) searchBuf(search string) {
	if len(search) == 1 {
		term.renderer.DisplayNotification(types.NOTIFY_WARN, "Search string too short. Minimum search length is 2")
		return
	}

	if search == "" {
		return
	}

	search = strings.ToLower(search)
	term._searchLastString = search

	term._mutex.Lock()
	defer term._mutex.Unlock()

	/*fnMatch := func(phrase string) bool {
		return strings.Contains(phrase, search)
	}*/

	rxMatch, err := regexp.Compile(search)
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_WARN, err.Error())
		return
	}

	fnMatch := func(phrase string) []int {
		index := rxMatch.FindStringIndex(phrase)
		if index == nil {
			return nil
		}
		return []int{index[0], runewidth.StringWidth(phrase[index[0]:index[1]])}
	}

	normOk := term._searchBuf(term._normBuf, fnMatch)
	scrollOk := term._searchBuf(term._scrollBuf, fnMatch)

	term._searchHighlight = term._searchHighlight || normOk || scrollOk

	if normOk || scrollOk {
		term.SearchShowResults()
		return
	}

	term.renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Search string not found: '%s'", search))
}

func (term *Term) SearchClearResults() {
	term._searchHighlight = false
	for _, cell := range term._searchHlHistory {
		if cell != nil && cell.Sgr != nil {
			cell.Sgr.Bitwise.Unset(types.SGR_HIGHLIGHT_SEARCH_RESULT)
		}
	}
	term._searchHlHistory = []*types.Cell{}
	term._searchResults = nil
	term._scrollOffset = 0
	term.updateScrollback()
}

func (term *Term) _searchBuf(buf types.Screen, fnSearch func(string) []int) bool {
	firstMatch := -1
	for y := len(buf) - 1; y >= 0; y-- {

		row, _ := buf.Phrase(y)
		inStr := fnSearch(strings.ToLower(row))
		if inStr != nil {
			// add to search results
			phrase := strings.TrimSpace(row)
			term._searchResults = append(term._searchResults, searchResult{
				rowId:  buf[y].Id,
				phrase: phrase,
			})

			// highlight
			x, z := inStr[0], 0
			for i := range inStr[1] {
				if x+i >= len(buf[y+z].Cells) {
					x = 0
					z++
				}
				buf[y+z].Cells[x+i].Sgr = buf[y+z].Cells[x+i].Sgr.Copy()
				buf[y+z].Cells[x+i].Sgr.Bitwise.Set(types.SGR_HIGHLIGHT_SEARCH_RESULT)
				term._searchHlHistory = append(term._searchHlHistory, buf[y].Cells[x+i])
			}

			if firstMatch == -1 {
				firstMatch = y
			}
		}
	}
	return firstMatch != -1
}

func (term *Term) SearchShowResults() {
	offset := term._scrollOffset
	sr := make([]searchResult, len(term._searchResults))
	results := make([]string, len(term._searchResults))

	for i := range term._searchResults {
		sr[i] = term._searchResults[i]
		results[i] = term._searchResults[i].phrase
	}

	cbHighlight := func(i int) {
		term.scrollToRowId(sr[i].rowId, _SEARCH_OFFSET)
	}
	cbCancel := func(int) {
		term._scrollOffset = offset
		term.updateScrollback()
	}
	cbSelect := func(int) {}
	term.renderer.DisplayMenu("Search results", results, cbHighlight, cbSelect, cbCancel)
}

type rowTupleT struct {
	rowId   uint64
	query   string
	exitNum int
}

type rowTuplesT []rowTupleT

func (t *rowTuplesT) Slice() []string {
	var s []string
	for i := range *t {
		s = append(s, (*t)[i].query)
	}
	return s
}

func (term *Term) SearchCmdLines()  { term.searchCmdLines("Commands", false) }
func (term *Term) SearchAiPrompts() { term.searchCmdLines("AI Queries", true) }

func (term *Term) searchCmdLines(menuTitle string, ai bool) {
	if term.IsAltBuf() {
		term.renderer.DisplayNotification(types.NOTIFY_WARN, "Search is not supported in alt buffer")
		return
	}

	term._mutex.Lock()
	defer term._mutex.Unlock()

	offset := term._scrollOffset

	tuples := _searchCmdLinesBuf(term._normBuf, ai)
	tuples = append(tuples, _searchCmdLinesBuf(term._scrollBuf, ai)...)

	fnHighlight := func(i int) {
		term.scrollToRowId(tuples[i].rowId, _SEARCH_OFFSET)
	}

	fnOk := func(int) {
		// do nothing
	}

	fnCancel := func(int) {
		term._scrollOffset = offset
		term.updateScrollback()
	}

	term.renderer.DisplayMenu(menuTitle, tuples.SliceWithExitNum(), fnHighlight, fnOk, fnCancel)
}

func (t *rowTuplesT) SliceWithExitNum() []string {
	var s []string
	for i := range *t {
		s = append(s, fmt.Sprintf("%-3d: %s", (*t)[i].exitNum, (*t)[i].query))
	}
	return s
}

func _searchCmdLinesBuf(buf types.Screen, ai bool) rowTuplesT {
	var tuples rowTuplesT

	for i := len(buf) - 1; i >= 0; i-- {
		if buf[i].RowMeta.Is(types.META_ROW_BEGIN_BLOCK) && buf[i].Block.Meta.Is(types.META_BLOCK_AI) == ai {
			tup := rowTupleT{
				rowId:   buf[i].Id,
				query:   string(buf[i].Block.Query),
				exitNum: buf[i].Block.ExitNum,
			}
			if len(tup.query) < 3 {
				tup.query += "   "
			}
			tuples = append(tuples, tup)
		}
	}

	return tuples
}
