package element_content_editable

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/spelling"
	"github.com/lmorg/ttyphoon/window/backend/cursor"
)

type ElementContentEditable struct {
	renderer    types.Renderer
	tile        types.Tile
	cells       []*types.Cell
	spellingExc map[string]bool
	suggestions []spelling.SuggestionT
	mu          sync.RWMutex
	pos         *types.XY
	text        string
}

func New(renderer types.Renderer, tile types.Tile, spellingExc map[string]bool) *ElementContentEditable {
	return &ElementContentEditable{
		renderer:    renderer,
		tile:        tile,
		spellingExc: spellingExc,
	}
}

var rxAnsiSgr = regexp.MustCompile(`\x1b\[[:;0-9]+m`)

func (el *ElementContentEditable) Generate(apc *types.ApcSlice) error {
	el.mu.RLock()

	el.text = apc.Index(2)
	line := (&types.Row{Cells: el.cells}).String()
	el.mu.RUnlock()

	go func(text string) {
		out, err := spelling.ExecAspell(text)
		if err != nil {
			return
		}

		el.suggestions, err = spelling.ParseAspellOutput(out)
		if err != nil {
			return
		}

		el.suggestions = spelling.FilterExclusions(el.suggestions, el.spellingExc)

		el.mu.Lock()
		defer el.mu.Unlock()

		for i := range el.suggestions {
			start := el.suggestions[i].WordStart
			end := start + el.suggestions[i].WordLength
			for c := start; c < end && c < len(el.cells); c++ {
				if el.cells[c] == nil || el.cells[c].Sgr == nil {
					continue
				}
				el.cells[c].Sgr.Bitwise.SetUnderlineStyle(types.UNDERLINE_CURLY)
				el.cells[c].Sgr.UlC = types.SGR_COLOR_RED
			}
		}

		el.renderer.TriggerLazyRedraw()
	}(line)

	return nil
}

func (el *ElementContentEditable) Write(r rune) error {
	/*if r == '\n' {
		return nil
	}*/

	el.mu.Lock()
	el.cells = append(el.cells, &types.Cell{Char: r, Sgr: el.tile.GetTerm().GetSgr().Copy()})
	el.mu.Unlock()
	return nil
}

func (el *ElementContentEditable) Size() *types.XY {
	el.mu.RLock()
	defer el.mu.RUnlock()
	return &types.XY{X: int32(len(el.cells)), Y: 1}
}

func (el *ElementContentEditable) Draw(termPos *types.XY) {
	el.mu.Lock()
	el.pos = termPos
	el.mu.Unlock()
	el.renderer.PrintRow(el.tile, el.cells, termPos)
}

func (el *ElementContentEditable) Rune(pos *types.XY) rune {
	el.mu.RLock()
	defer el.mu.RUnlock()
	return el.cells[pos.X].Rune()
}

func (el *ElementContentEditable) MouseClick(pos *types.XY, button types.MouseButtonT, count uint8, state types.ButtonStateT, callback types.EventIgnoredCallback) {
	if state != types.BUTTON_RELEASED || button != types.MOUSE_BUTTON_LEFT || pos == nil || pos.X < 0 {
		callback()
		return
	}

	suggestion, ok := el.suggestionAt(pos.X)
	if !ok {
		if count > 1 {
			el.visualEditor()
			return
		}
		callback()
		return
	}

	menu := el.renderer.NewContextMenu()

	if len(suggestion.Suggestions) == 0 {
		menu.Append(types.MenuItem{Title: "No spelling suggestions"})
	} else {
		for i := range suggestion.Suggestions {
			replacement := suggestion.Suggestions[i]
			start, length := suggestion.WordStart, suggestion.WordLength
			menu.Append(types.MenuItem{
				Title: replacement,
				Fn: func() {
					el.reply(el.replaceWord(start, length, replacement))
				},
				Icon: 0xf040,
			})
		}
	}

	menu.DisplayMenu(suggestion.MisspeltWord, true)
}

func (el *ElementContentEditable) MouseWheel(_ *types.XY, _ *types.XY, callback types.EventIgnoredCallback) {
	callback()
}

func (el *ElementContentEditable) MouseMotion(pos *types.XY, size *types.XY, callback types.EventIgnoredCallback) {
	if pos != nil && pos.X >= 0 {
		if _, ok := el.suggestionAt(pos.X); ok {
			cursor.Hand()
			el.renderer.StatusBarText("[Click] View spelling suggestions  |  [2x Click] Visual editor")
			callback()
			return
		}
	}
	cursor.Arrow()
	el.renderer.StatusBarText("[2x Click] Visual editor")
	callback()
}

func (el *ElementContentEditable) MouseHover(_ *types.XY, _ *types.XY) func() {
	return func() {}
}

func (el *ElementContentEditable) MouseOut() {
	cursor.Arrow()
	el.renderer.StatusBarText("")
}

func isRedCurlyUnderline(sgr *types.Sgr) bool {
	if sgr == nil || sgr.UlC == nil {
		return false
	}

	if sgr.Bitwise.GetUnderlineStyle() != types.UNDERLINE_CURLY {
		return false
	}

	return sgr.UlC.Red == types.SGR_COLOR_RED.Red &&
		sgr.UlC.Green == types.SGR_COLOR_RED.Green &&
		sgr.UlC.Blue == types.SGR_COLOR_RED.Blue
}

func (el *ElementContentEditable) suggestionAt(x int32) (spelling.SuggestionT, bool) {
	el.mu.RLock()
	defer el.mu.RUnlock()

	idx := int(x)
	if idx < 0 || idx >= len(el.cells) {
		return spelling.SuggestionT{}, false
	}

	if el.cells[idx] == nil || el.cells[idx].Sgr == nil || !isRedCurlyUnderline(el.cells[idx].Sgr) {
		return spelling.SuggestionT{}, false
	}

	for i := range el.suggestions {
		start := el.suggestions[i].WordStart
		end := start + el.suggestions[i].WordLength
		if idx >= start && idx < end {
			return el.suggestions[i], true
		}
	}

	return spelling.SuggestionT{}, false
}

func (el *ElementContentEditable) replaceWord(start, length int, replacement string) string {
	el.mu.RLock()
	defer el.mu.RUnlock()

	if start < 0 || length <= 0 || start > len(el.cells) {
		return (&types.Row{Cells: el.cells}).String()
	}

	end := min(start+length, len(el.cells))

	result := make([]rune, 0, len(el.cells)+len(replacement))

	for i := range start {
		result = append(result, el.cells[i].Rune())
	}

	result = append(result, []rune(replacement)...)

	for i := end; i < len(el.cells); i++ {
		result = append(result, el.cells[i].Rune())
	}

	return string(result)
}

const (
	seqApc = "\x1b_"
	seqST  = "\x1b\\"
)

func (el *ElementContentEditable) reply(s string) {
	reply := fmt.Sprintf("%sreply;content-editable;%s%s", seqApc, s, seqST)
	el.tile.GetTerm().Reply([]byte(reply))
}

func (el *ElementContentEditable) visualEditor() {
	el.mu.RLock()
	text := el.text
	el.mu.RUnlock()
	text = rxAnsiSgr.ReplaceAllString(text, "")
	el.renderer.DisplayInputBox("Visual editor", text, el.reply, nil)
}
