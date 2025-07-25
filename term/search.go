package virtualterm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lmorg/mxtty/types"
)

const _SEARCH_OFFSET = 3

func (term *Term) Search() {
	if term.IsAltBuf() {
		term.renderer.DisplayNotification(types.NOTIFY_WARN, "Search is not supported in alt buffer")
		return
	}

	if len(term._searchResults) == 0 {
		term.search()
	} else {
		term.ShowSearchResults()
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

	term.searchClearResults()

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

	fnMatch := func(phrase string) bool {
		return rxMatch.MatchString(phrase)
	}

	normOk := term._searchBuf(term._normBuf, fnMatch)
	scrollOk := term._searchBuf(term._scrollBuf, fnMatch)

	term._searchHighlight = term._searchHighlight || normOk || scrollOk

	if normOk || scrollOk {
		term.ShowSearchResults()
		return
	}

	term.renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Search string not found: '%s'", search))
}

func (term *Term) searchClearResults() {
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

func (term *Term) _searchBuf(buf types.Screen, fnSearch func(string) bool) bool {
	firstMatch := -1
	for y := len(buf) - 1; y >= 0; y-- {
		for x := len(buf[y].Cells) - 1; x >= 0; x-- {
			if buf[y].Cells[x].Phrase == nil {
				continue
			}

			s := strings.ToLower(string(*buf[y].Cells[x].Phrase))
			if fnSearch(s) {
				phrase := string(*buf[y].Cells[x].Phrase)
				phrase = strings.ReplaceAll(phrase, "\n", "")
				phrase = strings.TrimSpace(phrase)
				term._searchResults = append(term._searchResults, searchResult{
					rowId:  buf[y].Id,
					phrase: phrase,
				})
				i, j, l := 0, 0, 0
			highlight:
				for ; x+i < len(buf[y].Cells); i++ {
					buf[y+j].Cells[x+i].Sgr = buf[y+j].Cells[x+i].Sgr.Copy()
					buf[y+j].Cells[x+i].Sgr.Bitwise.Set(types.SGR_HIGHLIGHT_SEARCH_RESULT)
					term._searchHlHistory = append(term._searchHlHistory, buf[y+j].Cells[x+i])
					l++
				}
				if l < len(*buf[y].Cells[x].Phrase) {
					i = 0
					j++
					goto highlight
				}

				if firstMatch == -1 {
					firstMatch = y
				}
			}
		}
	}
	return firstMatch != -1
}

func (term *Term) ShowSearchResults() {
	offset := term._scrollOffset
	sr := make([]searchResult, len(term._searchResults))
	results := make([]string, len(term._searchResults))
	j := len(term._searchResults) - 1

	for i := range term._searchResults {
		sr[j] = term._searchResults[i]
		results[j] = term._searchResults[i].phrase
		j--
	}

	cbHighlight := func(i int) {
		term.scrollToRowId(sr[i].rowId, _SEARCH_OFFSET)
	}
	cbCancel := func(int) {
		term._scrollOffset = offset
		term.updateScrollback()
		term.search()
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
		if buf[i].RowMeta.Is(types.META_ROW_BEGIN) && buf[i].Block.Meta.Is(types.META_BLOCK_AI) == ai {
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
